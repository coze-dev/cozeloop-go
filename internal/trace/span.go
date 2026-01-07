// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

import (
	"context"
	"encoding/json"
	"fmt"
	"net/textproto"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal"
	"github.com/coze-dev/cozeloop-go/internal/consts"
	"github.com/coze-dev/cozeloop-go/internal/logger"
	"github.com/coze-dev/cozeloop-go/internal/util"
	"github.com/coze-dev/cozeloop-go/spec/tracespec"
)

const (
	spanUnFinished = 0
	spanFinished   = 1
)

type SpanContext struct {
	SpanID  string
	TraceID string
	Baggage map[string]string
}

func (s *SpanContext) GetSpanID() string {
	return s.SpanID
}

func (s *SpanContext) GetTraceID() string {
	return s.TraceID
}

func (s *SpanContext) GetBaggage() map[string]string {
	return s.Baggage
}

type Span struct {
	// span context param
	SpanContext

	// basic param
	SpanType string
	Name     string

	// These params can be changed, but remember locking when changed
	ServiceName  string
	LogID        string
	WorkspaceID  string
	ParentSpanID string
	StartTime    time.Time
	FinishTime   time.Time
	Duration     time.Duration
	TagMap       map[string]interface{} // custom tags
	SystemTagMap map[string]interface{} // system tags
	StatusCode   int32

	// These params is internal field
	multiModalityKeyMap    map[string]struct{}
	ultraLargeReportKeyMap map[string]struct{}
	ultraLargeReport       bool
	spanProcessor          SpanProcessor
	flags                  byte  // for W3C, useless now
	isFinished             int32 // avoid executing finish repeatedly.
	lock                   sync.RWMutex
	bytesSize              int64            // bytes size of span, note: it is an estimated value, may not be accurate.
	tagTruncateConf        *TagTruncateConf // tag truncate byte conf
}

type TagTruncateConf struct {
	NormalFieldMaxByte      int
	InputOutputFieldMaxByte int
}

func (s *Span) GetBaggage() map[string]string {
	var bg map[string]string
	s.lock.RLock()
	rawBag := s.SpanContext.GetBaggage()
	if rawBag != nil {
		bg = make(map[string]string)
		for k, v := range rawBag {
			bg[k] = v
		}
	}
	s.lock.RUnlock()
	return bg
}

func (s *Span) setCutOffTag(cutOffKeys []string) {
	if cutOffTags, ok := s.SystemTagMap[consts.CutOff]; ok {
		if value, ok := cutOffTags.([]string); ok {
			cutOffKeys = append(cutOffKeys, value...)
		}
	}
	s.SystemTagMap[consts.CutOff] = util.RmDupStrSlice(cutOffKeys)
}

func (s *Span) setTagItem(ctx context.Context, key string, value interface{}) {
	if int64(len(s.TagMap)) < consts.MaxTagKvCountInOneSpan {
		s.setTagUnlock(key, value)
	} else {
		logger.CtxErrorf(ctx, "tag count exceed limit:%d", consts.MaxTagKvCountInOneSpan)
	}
	return
}

func (s *Span) setTagUnlock(key string, value interface{}) {
	s.TagMap[key] = value
}

func (s *Span) GetParentID() string {
	if s == nil {
		return ""
	}
	return s.ParentSpanID
}

func (s *Span) GetTagMap() map[string]interface{} {
	if s == nil {
		return nil
	}

	var tagMap map[string]interface{}
	s.lock.RLock()
	if s.TagMap != nil {
		tagMap = make(map[string]interface{})
		for k, v := range s.TagMap {
			tagMap[k] = v
		}
	}
	s.lock.RUnlock()
	return tagMap
}

func (s *Span) GetDuration() int64 {
	if s == nil {
		return 0
	}
	return int64(s.Duration)
}

func (s *Span) GetSpaceID() string {
	if s == nil {
		return ""
	}

	return s.WorkspaceID
}

func (s *Span) GetSpanName() string {
	if s == nil {
		return ""
	}
	return s.Name
}

func (s *Span) GetSpanType() string {
	if s == nil {
		return ""
	}
	return s.SpanType
}

func (s *Span) GetStatusCode() int32 {
	if s == nil {
		return 0
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.StatusCode
}

func (s *Span) UltraLargeReport() bool {
	if s == nil {
		return false
	}
	return s.ultraLargeReport
}

func oneTag(k string, v interface{}) map[string]interface{} {
	return map[string]interface{}{k: v}
}

func oneBaggage(k string, v string) map[string]string {
	return map[string]string{k: v}
}

func FromHeader(ctx context.Context, h map[string]string) *SpanContext {
	header := make(map[string]string)
	for key, value := range h {
		newKey := textproto.CanonicalMIMEHeaderKey(key)
		header[newKey] = value
	}

	s := &SpanContext{}
	// W3C: https://www.w3.org/TR/trace-context/#tracestate-header
	if headerParent, ok := header[consts.TraceContextHeaderParent]; ok {
		traceID, spanID, err := fromHeaderParent(headerParent)
		if err != nil {
			// return null span context if failed to parse header parent
			logger.CtxWarnf(ctx, "failed to parse header parent: %v", err)
		} else {
			s.TraceID = traceID
			s.SpanID = spanID
		}
	}

	if headerBaggage, ok := header[consts.TraceContextHeaderBaggage]; ok {
		s.Baggage = fromHeaderBaggage(headerBaggage)
	}

	return s
}

func fromHeaderBaggage(h string) map[string]string {
	return parseCommaSeparatedMap(h, true)
}

func parseCommaSeparatedMap(src string, cover bool) (baggage map[string]string) {
	baggage = make(map[string]string)

	s := strings.Split(src, consts.Comma)
	for i := 0; i < len(s); i++ {
		kv := strings.Split(strings.TrimSpace(s[i]), consts.Equal)
		if len(kv) != 2 {
			continue
		}

		tempKey, err := url.QueryUnescape(kv[0])
		if err != nil {
			return baggage
		}
		tempValue, err := url.QueryUnescape(kv[1])
		if err != nil {
			return baggage
		}

		// empty value is invalid
		if tempKey == "" || tempValue == "" {
			continue
		}

		_, exist := baggage[tempKey]
		// cover old value if cover is true
		if !exist || cover {
			baggage[tempKey] = tempValue
		}
	}
	return baggage
}

func fromHeaderParent(h string) (traceID, spanID string, err error) {
	splits := strings.Split(h, "-")
	if len(splits) != 4 {
		return "", "", consts.ErrHeaderParent
	}

	traceIDTemp := splits[1]
	if len(traceIDTemp) != 32 || traceIDTemp == "00000000000000000000000000000000" {
		return "", "", consts.ErrHeaderParent.Wrap(fmt.Errorf("invalid trace id: %s", traceIDTemp))
	}
	if !util.IsValidHexStr(traceIDTemp) {
		return "", "", consts.ErrHeaderParent.Wrap(fmt.Errorf("invalid trace id: %s", traceIDTemp))
	}

	spanIDTemp := splits[2]
	if len(spanIDTemp) != 16 || spanIDTemp == "0000000000000000" {
		return "", "", consts.ErrHeaderParent.Wrap(fmt.Errorf("invalid span id: %s", spanIDTemp))
	}
	if !util.IsValidHexStr(spanIDTemp) {
		return "", "", consts.ErrHeaderParent.Wrap(fmt.Errorf("invalid span id: %s", spanIDTemp))
	}

	return traceIDTemp, spanIDTemp, nil
}

func (s *Span) SetInput(ctx context.Context, input interface{}) {
	if s == nil || s.isSpanFinished() {
		return
	}

	messageParts := make([]*tracespec.ModelMessagePart, 0)
	mContent := tracespec.ModelInput{}

	if tempContent, ok := input.(tracespec.ModelInput); ok {
		mContent = tempContent
		for _, message := range tempContent.Messages {
			messageParts = append(messageParts, message.Parts...)
		}
	}
	if tempContent, ok := input.(*tracespec.ModelInput); ok {
		mContent = *tempContent
		for _, message := range tempContent.Messages {
			messageParts = append(messageParts, message.Parts...)
		}
	}

	isMultiModality := parseModelMessageParts(messageParts)
	if isMultiModality {
		s.SetMultiModalityMap(tracespec.Input)
		size := getModelInputBytesSize(deepCopyMessageOfModelInput(mContent))
		s.lock.Lock()
		s.bytesSize += size
		s.lock.Unlock()
	}

	s.SetTags(ctx, oneTag(tracespec.Input, input))
}

func deepCopyMessageOfModelInput(src tracespec.ModelInput) tracespec.ModelInput {
	result := tracespec.ModelInput{}
	result.Messages = make([]*tracespec.ModelMessage, len(src.Messages))
	for i, message := range src.Messages {
		result.Messages[i] = &tracespec.ModelMessage{
			Parts: make([]*tracespec.ModelMessagePart, len(message.Parts)),
		}
		for j, part := range message.Parts {
			tempPart := &tracespec.ModelMessagePart{
				Type:     part.Type,
				Text:     part.Text,
				ImageURL: part.ImageURL,
				FileURL:  part.FileURL,
			}
			if part.ImageURL != nil {
				tempPart.ImageURL = &tracespec.ModelImageURL{
					Name:   part.ImageURL.Name,
					URL:    part.ImageURL.URL,
					Detail: part.ImageURL.Detail,
				}
			}
			if part.FileURL != nil {
				tempPart.FileURL = &tracespec.ModelFileURL{
					Name:   part.FileURL.Name,
					URL:    part.FileURL.URL,
					Detail: part.FileURL.Detail,
					Suffix: part.FileURL.Suffix,
				}
			}
			result.Messages[i].Parts[j] = tempPart
		}
	}

	result.Tools = src.Tools
	result.ModelToolChoice = src.ModelToolChoice

	return result
}

func getModelInputBytesSize(mContent tracespec.ModelInput) int64 {
	for _, message := range mContent.Messages {
		if message == nil {
			continue
		}
		for _, part := range message.Parts {
			if part == nil {
				continue
			}
			switch part.Type {
			case tracespec.ModelMessagePartTypeImage:
				if part.ImageURL != nil && part.ImageURL.URL != "" {
					part.ImageURL.URL = ""
				}
			case tracespec.ModelMessagePartTypeFile:
				if part.FileURL != nil && part.FileURL.URL != "" {
					part.FileURL.URL = ""
				}
			}
		}
	}
	mContentJson, err := json.Marshal(mContent)
	if err != nil {
		return 0
	}

	return int64(len(mContentJson))
}

func parseModelMessageParts(mContents []*tracespec.ModelMessagePart) (isMultiModality bool) {
	for _, content := range mContents {
		switch content.Type {
		case tracespec.ModelMessagePartTypeImage:
			if content.ImageURL != nil && content.ImageURL.URL != "" {
				if base64Data, isBase64 := util.ParseValidMDNBase64(content.ImageURL.URL); isBase64 {
					content.ImageURL.URL = base64Data
					isMultiModality = true
				}
				if isValidURL := util.IsValidURL(content.ImageURL.URL); isValidURL {
					isMultiModality = true
				}
			}
		case tracespec.ModelMessagePartTypeFile:
			if content.FileURL != nil && content.FileURL.URL != "" {
				if base64Data, isBase64 := util.ParseValidMDNBase64(content.FileURL.URL); isBase64 {
					content.FileURL.URL = base64Data
					isMultiModality = true
				}
				if isValidURL := util.IsValidURL(content.FileURL.URL); isValidURL {
					isMultiModality = true
				}
			}
		}
	}

	return isMultiModality
}

func (s *Span) SetOutput(ctx context.Context, output interface{}) {
	if s == nil || s.isSpanFinished() {
		return
	}
	mContent := tracespec.ModelOutput{}
	messageParts := make([]*tracespec.ModelMessagePart, 0)
	if mContents, ok := output.(tracespec.ModelOutput); ok {
		mContent = mContents
		for _, choice := range mContents.Choices {
			if choice.Message != nil {
				messageParts = append(messageParts, choice.Message.Parts...)
			}
		}
	}
	if mContents, ok := output.(*tracespec.ModelOutput); ok {
		mContent = *mContents
		for _, choice := range mContents.Choices {
			if choice.Message != nil {
				messageParts = append(messageParts, choice.Message.Parts...)
			}
		}
	}

	isMultiModality := parseModelMessageParts(messageParts)
	if isMultiModality {
		s.SetMultiModalityMap(tracespec.Output)
		size := getModelOutputBytesSize(deepCopyMessageOfModelOutput(mContent))
		s.lock.Lock()
		s.bytesSize += size
		s.lock.Unlock()
	}

	s.SetTags(ctx, oneTag(tracespec.Output, output))
}

func deepCopyMessageOfModelOutput(src tracespec.ModelOutput) tracespec.ModelOutput {
	result := tracespec.ModelOutput{}
	result.Choices = make([]*tracespec.ModelChoice, len(src.Choices))
	for i, choice := range src.Choices {
		result.Choices[i] = &tracespec.ModelChoice{
			Message: &tracespec.ModelMessage{
				Parts: make([]*tracespec.ModelMessagePart, len(choice.Message.Parts)),
			},
		}
		for j, part := range choice.Message.Parts {
			tempPart := &tracespec.ModelMessagePart{
				Type: part.Type,
				Text: part.Text,
			}
			if part.ImageURL != nil {
				tempPart.ImageURL = &tracespec.ModelImageURL{
					Name:   part.ImageURL.Name,
					URL:    part.ImageURL.URL,
					Detail: part.ImageURL.Detail,
				}
			}
			if part.FileURL != nil {
				tempPart.FileURL = &tracespec.ModelFileURL{
					Name:   part.FileURL.Name,
					URL:    part.FileURL.URL,
					Detail: part.FileURL.Detail,
					Suffix: part.FileURL.Suffix,
				}
			}
			result.Choices[i].Message.Parts[j] = tempPart
		}
	}

	return result
}

func getModelOutputBytesSize(mContent tracespec.ModelOutput) int64 {
	for _, choice := range mContent.Choices {
		if choice.Message == nil {
			continue
		}
		for _, part := range choice.Message.Parts {
			if part == nil {
				continue
			}
			switch part.Type {
			case tracespec.ModelMessagePartTypeImage:
				if part.ImageURL != nil && part.ImageURL.URL != "" {
					part.ImageURL.URL = ""
				}
			case tracespec.ModelMessagePartTypeFile:
				if part.FileURL != nil && part.FileURL.URL != "" {
					part.FileURL.URL = ""
				}
			}
		}
	}

	mContentJson, err := json.Marshal(mContent)
	if err != nil {
		return 0
	}

	return int64(len(mContentJson))
}

func (s *Span) SetError(ctx context.Context, err error) {
	if s == nil || s.isSpanFinished() {
		return
	}
	s.SetTags(ctx, oneTag(tracespec.Error, err.Error()))
}

func (s *Span) SetStatusCode(ctx context.Context, code int) {
	if s == nil || s.isSpanFinished() {
		return
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	s.StatusCode = int32(code)
}

func (s *Span) SetUserID(ctx context.Context, userID string) {
	if s == nil || s.isSpanFinished() {
		return
	}
	s.SetTags(ctx, oneTag(consts.UserID, userID))
}

func (s *Span) SetUserIDBaggage(ctx context.Context, userID string) {
	if s == nil || s.isSpanFinished() {
		return
	}
	s.SetBaggage(ctx, oneBaggage(consts.UserID, userID))
}

func (s *Span) SetMessageID(ctx context.Context, messageID string) {
	if s == nil || s.isSpanFinished() {
		return
	}
	s.SetTags(ctx, oneTag(consts.MessageID, messageID))
}

func (s *Span) SetMessageIDBaggage(ctx context.Context, messageID string) {
	if s == nil || s.isSpanFinished() {
		return
	}
	s.SetBaggage(ctx, oneBaggage(consts.MessageID, messageID))
}

func (s *Span) SetThreadID(ctx context.Context, threadID string) {
	if s == nil || s.isSpanFinished() {
		return
	}
	s.SetTags(ctx, oneTag(consts.ThreadID, threadID))
}

func (s *Span) SetThreadIDBaggage(ctx context.Context, threadID string) {
	if s == nil || s.isSpanFinished() {
		return
	}
	s.SetBaggage(ctx, oneBaggage(consts.ThreadID, threadID))
}

func (s *Span) SetPrompt(ctx context.Context, prompt entity.Prompt) {
	if s == nil || s.isSpanFinished() {
		return
	}
	if len(prompt.PromptKey) > 0 {
		s.SetTags(ctx, oneTag(tracespec.PromptKey, prompt.PromptKey))
		if len(prompt.Version) > 0 {
			s.SetTags(ctx, oneTag(tracespec.PromptVersion, prompt.Version))
		}
	}
}

func (s *Span) SetModelProvider(ctx context.Context, modelProvider string) {
	if s == nil || s.isSpanFinished() {
		return
	}
	s.SetTags(ctx, oneTag(tracespec.ModelProvider, modelProvider))
}

func (s *Span) SetModelName(ctx context.Context, modelName string) {
	if s == nil || s.isSpanFinished() {
		return
	}
	s.SetTags(ctx, oneTag(tracespec.ModelName, modelName))
}

func (s *Span) SetModelCallOptions(ctx context.Context, callOptions interface{}) {
	if s == nil || s.isSpanFinished() {
		return
	}
	s.SetTags(ctx, oneTag(tracespec.CallOptions, callOptions))
}

func (s *Span) SetInputTokens(ctx context.Context, inputTokens int) {
	if s == nil || s.isSpanFinished() {
		return
	}
	s.SetTags(ctx, oneTag(tracespec.InputTokens, inputTokens))
}

func (s *Span) SetOutputTokens(ctx context.Context, outputTokens int) {
	if s == nil || s.isSpanFinished() {
		return
	}
	s.SetTags(ctx, oneTag(tracespec.OutputTokens, outputTokens))
}

func (s *Span) SetStartTimeFirstResp(ctx context.Context, startTimeFirstResp int64) {
	if s == nil || s.isSpanFinished() {
		return
	}
	s.SetTags(ctx, oneTag(consts.StartTimeFirstResp, startTimeFirstResp))
}

func (s *Span) SetTags(ctx context.Context, tagKVs map[string]interface{}) {
	if s == nil || len(tagKVs) == 0 || s.isSpanFinished() {
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	s.addDefaultTag(ctx, tagKVs)
	rectifiedMap, cutOffKeys, byteSize := s.GetRectifiedMap(ctx, tagKVs)
	s.bytesSize += byteSize
	if len(cutOffKeys) > 0 {
		s.setCutOffTag(cutOffKeys)
	}
	for key, value := range rectifiedMap {
		s.setTagItem(ctx, key, value)
	}
}

func (s *Span) addDefaultTag(ctx context.Context, tagKVs map[string]interface{}) {
	for key := range tagKVs {
		switch key {
		case tracespec.Error:
			if s.StatusCode == 0 {
				s.StatusCode = int32(consts.StatusCodeErrorDefault)
			}
		default:
		}
	}
}

// GetRectifiedMap get rectified tag map and cut off keys
func (s *Span) GetRectifiedMap(ctx context.Context, inputMap map[string]interface{}) (map[string]interface{}, []string, int64) {
	validateMap := make(map[string]interface{})
	var cutOffKeys []string
	var bytesSize int64
	for key, value := range inputMap {
		// verify data type of reserve tag
		if expectedType, exists := consts.ReserveFieldTypes[key]; exists {
			if !isTagValidDataType(key, value) {
				logger.CtxErrorf(ctx, "The value for field [%s] is not in the correct format, type:%s, expectedType:%s", key, reflect.TypeOf(value), expectedType)
				continue
			}
		}
		var valueStr string
		if isCanCutOff(value) {
			valueStr = util.ToJSON(value)
			value = valueStr
		}
		// Truncate the value if a single tag's value is too large
		tagValueLengthLimit := s.getTagValueSizeLimit(key)
		isUltraLargeReport := false
		v, isTruncate := util.TruncateStringByByte(valueStr, tagValueLengthLimit)
		if isTruncate {
			if _, ok := s.multiModalityKeyMap[key]; !ok && s.UltraLargeReport() { // not multi-modality, enable ultra-large-report option, do ultra-large-report
				isUltraLargeReport = true
			}
			if _, ok := s.multiModalityKeyMap[key]; !ok && !s.UltraLargeReport() { // multi-modality or ultra large report, skip check value
				value = v
				cutOffKeys = append(cutOffKeys, key)
				logger.CtxWarnf(ctx, "field value [%s] is too long, and opt.EnableLongReport is false, so value has been truncated to %d size", key, tagValueLengthLimit)
			}
		}

		// Truncate the key if a single tag's key is too large
		tagKeyLengthLimit := util.GetTagKeySizeLimit()
		key, isTruncate := util.TruncateStringByByte(key, tagKeyLengthLimit)
		if isTruncate {
			cutOffKeys = append(cutOffKeys, key)
			logger.CtxWarnf(ctx, "field key [%s] is too long, and opt.EnableLongReport is false, so key has been truncated to %d size", key, tagKeyLengthLimit)
		}

		validateMap[key] = value

		bytesSize += int64(len(key))
		if _, ok := s.multiModalityKeyMap[key]; !ok && !isUltraLargeReport { // multi-modality has added, and ultra-large-report text is ignored
			bytesSize += int64(len(valueStr))
		}

	}
	return validateMap, cutOffKeys, bytesSize
}

func (s *Span) getTagValueSizeLimit(tagKey string) int {
	limit := util.GetTagValueSizeLimit(tagKey)

	switch tagKey {
	case tracespec.Input, tracespec.Output:
		if s.tagTruncateConf != nil && s.tagTruncateConf.InputOutputFieldMaxByte > 0 {
			limit = s.tagTruncateConf.InputOutputFieldMaxByte
		}
	default:
		if s.tagTruncateConf != nil && s.tagTruncateConf.NormalFieldMaxByte > 0 {
			limit = s.tagTruncateConf.NormalFieldMaxByte
		}
	}

	return limit
}

func isTagValidDataType(key string, value interface{}) bool {
	types, ok := consts.ReserveFieldTypes[key]
	if !ok {
		return false
	}
	for _, t := range types {
		if reflect.TypeOf(value) == t {
			return true
		}
	}
	return false
}

func isCanCutOff(value interface{}) bool {
	if value == nil {
		return false
	}
	kind := reflect.TypeOf(value).Kind()
	return kind == reflect.Ptr || kind == reflect.Struct || kind == reflect.Map || kind == reflect.Array ||
		kind == reflect.Slice || kind == reflect.String
}

func (s *Span) SetMultiModalityMap(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.multiModalityKeyMap == nil {
		s.multiModalityKeyMap = make(map[string]struct{})
	}
	s.multiModalityKeyMap[key] = struct{}{}
}

func (s *Span) SetBaggage(ctx context.Context, baggageItems map[string]string) {
	if s == nil || s.isSpanFinished() {
		return
	}
	if len(baggageItems) == 0 {
		return
	}

	s.setBaggage(ctx, baggageItems)
}

func (s *Span) setBaggage(ctx context.Context, baggageItems map[string]string) {
	if s == nil {
		return
	}
	if len(baggageItems) == 0 {
		return
	}

	for key, value := range baggageItems {
		if !isValidBaggageItem(ctx, key, value) {
			logger.CtxErrorf(ctx, "invalid baggageItems:%s:%s", key, value)
		} else {
			s.SetTags(ctx, map[string]interface{}{key: value})
			newKey := key
			newValue := value
			s.SetBaggageItem(newKey, newValue)
		}
	}
}

func isValidBaggageItem(ctx context.Context, key, value string) bool {
	// size check
	keyLimit := util.GetTagKeySizeLimit()
	valueLimit := util.GetTagValueSizeLimit(key)
	if len(key) > keyLimit || len(value) > valueLimit {
		logger.CtxInfof(ctx, "length of Baggage is too large, key:%s, value:%s", key, value)
		return false
	}
	// special char check
	if hasSpecialChar(key) {
		logger.CtxErrorf(ctx, "Baggage should not contain special characters, key:%s, value:%s", key, value)
		return false
	}
	return true
}

func hasSpecialChar(s string) bool {
	for _, specialString := range consts.BaggageSpecialChars {
		if strings.Contains(s, specialString) {
			return true
		}
	}
	return false
}

func (s *Span) SetBaggageItem(restrictedKey, value string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.Baggage == nil {
		s.Baggage = make(map[string]string)
	}
	s.Baggage[restrictedKey] = value
	return
}

func (s *Span) Finish(ctx context.Context) {
	if s == nil {
		return
	}
	if !s.isDoFinish() {
		return
	}
	s.setSystemTag(ctx)
	s.setStatInfo(ctx)
	s.spanProcessor.OnSpanEnd(ctx, s)
}

func (s *Span) isDoFinish() bool {
	return atomic.CompareAndSwapInt32(&s.isFinished, spanUnFinished, spanFinished)
}

func (s *Span) isSpanFinished() bool {
	return atomic.LoadInt32(&s.isFinished) == spanFinished
}

func (s *Span) setSystemTag(ctx context.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.SystemTagMap == nil {
		s.SystemTagMap = make(map[string]interface{})
	}

	runtime := tracespec.Runtime{}
	runtimeObj, ok := s.SystemTagMap[tracespec.Runtime_]
	if ok {
		if temp, ok := runtimeObj.(tracespec.Runtime); ok {
			runtime = temp
		}
	}

	runtime.Language = tracespec.VLangGo
	if runtime.Scene == "" {
		runtime.Scene = tracespec.VSceneCustom
	}
	runtime.LoopSDKVersion = internal.Version()

	s.SystemTagMap[tracespec.Runtime_] = util.ToJSON(runtime)
}

// SetStatInfo sets statistical data.
func (s *Span) setStatInfo(ctx context.Context) {
	tagMap := s.GetTagMap()
	if tempV, ok := tagMap[consts.StartTimeFirstResp]; ok {
		// latency_first_resp = start_time_first_resp - start_time
		s.SetTags(ctx, map[string]interface{}{consts.LatencyFirstResp: util.GetValueOfInt(tempV) - s.GetStartTime().UnixMicro()})
	}

	inputTokens, inputTokensExist := tagMap[tracespec.InputTokens]
	outputTokens, outputTokensExist := tagMap[tracespec.OutputTokens]
	if inputTokensExist || outputTokensExist {
		// tokens = input_tokens+output_tokens
		s.SetTags(ctx, map[string]interface{}{tracespec.Tokens: util.GetValueOfInt(inputTokens) + util.GetValueOfInt(outputTokens)})
	}

	// Duration = finish_time - start_time, unit: microseconds
	finishTime := time.Now()
	if !s.GetFinishTime().IsZero() {
		finishTime = s.GetFinishTime()
	}
	duration := finishTime.UnixNano()/1000 - s.GetStartTime().UnixNano()/1000
	s.lock.Lock()
	s.Duration = time.Duration(duration)
	s.lock.Unlock()
}

func (s *Span) GetStartTime() time.Time {
	if s == nil {
		return time.Time{}
	}
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.StartTime
}

func (s *Span) GetLogID() string {
	if s == nil {
		return ""
	}
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.LogID
}

func (s *Span) GetServiceName() string {
	if s == nil {
		return ""
	}
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.ServiceName
}

func (s *Span) ToHeader() (map[string]string, error) {
	if s == nil {
		return nil, nil
	}

	// W3C: https://www.w3.org/TR/trace-context/#tracestate-header
	var err error
	res := make(map[string]string, 2)

	res[consts.TraceContextHeaderParent] = s.toHeaderParent()
	res[consts.TraceContextHeaderBaggage], err = s.toHeaderBaggage()
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Span) toHeaderBaggage() (string, error) {
	if len(s.Baggage) == 0 {
		return "", nil
	}
	m := make(map[string]string)
	for k, v := range s.Baggage {
		tempK := k
		tempV := v
		// empty key or value is invalid
		if tempK != "" && tempV != "" {
			m[url.QueryEscape(tempK)] = url.QueryEscape(tempV)
		}
	}
	return util.MapToStringString(m), nil
}

func (s *Span) toHeaderParent() string {
	return fmt.Sprintf("%02x-%s-%s-%02x", consts.GlobalTraceVersion, s.TraceID, s.SpanID, s.flags)
}

func (s *Span) SetRuntime(ctx context.Context, runtime tracespec.Runtime) {
	if s == nil || s.isSpanFinished() {
		return
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.SystemTagMap == nil {
		s.SystemTagMap = make(map[string]interface{})
	}
	runtime.Scene = tracespec.VSceneCustom
	s.SystemTagMap[tracespec.Runtime_] = runtime
}

func (s *Span) SetServiceName(ctx context.Context, serviceName string) {
	if s == nil || s.isSpanFinished() {
		return
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	s.ServiceName = serviceName
}

func (s *Span) SetLogID(ctx context.Context, logID string) {
	if s == nil || s.isSpanFinished() {
		return
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	s.LogID = logID
}

func (s *Span) IsRootSpan() bool {
	return s.ParentSpanID == "" || s.ParentSpanID == "0"
}

// SetFinishTime
// Default is time.Now() when span Finish(). DO NOT set unless you do not use default time.
func (s *Span) SetFinishTime(finishTime time.Time) {
	if s == nil || s.isSpanFinished() {
		return
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	s.FinishTime = finishTime
}

func (s *Span) GetFinishTime() time.Time {
	if s == nil {
		return time.Time{}
	}
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.FinishTime
}

func (s *Span) SetSystemTags(ctx context.Context, systemTags map[string]interface{}) {
	if s == nil || s.isSpanFinished() {
		return
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	for key, value := range systemTags {
		s.SystemTagMap[key] = value
	}
}

func (s *Span) SetDeploymentEnv(ctx context.Context, deploymentEnv string) {
	if s == nil || s.isSpanFinished() {
		return
	}
	s.SetTags(ctx, oneTag(consts.DeploymentEnv, deploymentEnv))
}
