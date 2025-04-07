// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/coze-dev/cozeloop-go/internal/consts"
	"github.com/coze-dev/cozeloop-go/internal/httpclient"
	"github.com/coze-dev/cozeloop-go/internal/logger"
	model2 "github.com/coze-dev/cozeloop-go/internal/trace/model"
	"github.com/coze-dev/cozeloop-go/internal/util"
	"github.com/coze-dev/cozeloop-go/spec/tracespec"
)

type Exporter interface {
	ExportSpans(ctx context.Context, spans []*UploadSpan) error
	ExportFiles(ctx context.Context, files []*UploadFile) error
}

const (
	KeyTemplateLargeText     = "%s_%s_%s_%s_large_text"
	KeyTemplateMultiModality = "%s_%s_%s_%s_%s"

	fileTypeText  = "text"
	fileTypeImage = "image"
	fileTypeFile  = "file"

	pathIngestTrace = "/v1/loop/traces/ingest"
	pathUploadFile  = "/v1/loop/files/upload"
)

var _ Exporter = (*SpanExporter)(nil)

type SpanExporter struct {
	client *httpclient.Client
}

func (e *SpanExporter) ExportFiles(ctx context.Context, files []*UploadFile) error {
	uploadFiles := files
	for _, file := range uploadFiles {
		if file == nil {
			continue
		}
		logger.CtxDebugf(ctx, "uploadFile start, file name: %s", file.Name)
		resp := httpclient.BaseResponse{}
		err := e.client.UploadFile(ctx, pathUploadFile, file.TosKey, bytes.NewReader([]byte(file.Data)), map[string]string{"workspace_id": file.SpaceID}, &resp)
		if err != nil {
			logger.CtxErrorf(ctx, "export files[%s] fail, err:[%v], retry later", file.TosKey, err)
			return err
		}
		if resp.GetCode() != 0 { // todo: some err code do not need retry
			logger.CtxErrorf(ctx, "export files[%s] fail, code:[%v], msg:[%v] retry later", file.TosKey, resp.GetCode(), resp.GetMsg())
			return consts.ErrRemoteService
		}
		logger.CtxDebugf(ctx, "uploadFile end, file name: %s", file.Name)
	}

	return nil
}

func (e *SpanExporter) ExportSpans(ctx context.Context, ss []*UploadSpan) (err error) {
	if len(ss) == 0 {
		return
	}
	logger.CtxDebugf(ctx, "export spans, spans count: %d", len(ss))

	resp := httpclient.BaseResponse{}
	err = e.client.Post(ctx, pathIngestTrace, UploadSpanData{ss}, &resp)
	if err != nil {
		logger.CtxErrorf(ctx, "export spans fail, span count: [%d], err:[%v]", len(ss), err)
		return err
	}
	if resp.GetCode() != 0 { // todo: some err code do not need retry
		logger.CtxErrorf(ctx, "export spans fail, span count: [%d], code:[%v], msg:[%v]", len(ss), resp.GetCode(), resp.GetMsg())
		return consts.ErrRemoteService
	}

	return
}

func transferToUploadSpanAndFile(ctx context.Context, spans []*Span) ([]*UploadSpan, []*UploadFile) {
	resSpan := make([]*UploadSpan, 0, len(spans))
	resFile := make([]*UploadFile, 0, len(spans))

	for _, span := range spans {
		spanUploadFile, putContentMap, err := parseInputOutput(ctx, span)
		if err != nil {
			logger.CtxErrorf(ctx, "parseInputOutput failed, err: %v", err)
			continue
		}
		objectStorageByte, err := transferObjectStorage(spanUploadFile)
		if err != nil {
			logger.CtxErrorf(ctx, "transferObjectStorage failed, err: %v", err)
			continue
		}

		resFile = append(resFile, spanUploadFile...)

		tagStrM, tagLongM, tagDoubleM := parseTag(span.TagMap)
		systemTagStrM, systemTagLongM, systemTagDoubleM := parseTag(span.SystemTagMap)
		resSpan = append(resSpan, &UploadSpan{
			StartedATMicros:  span.GetStartTime().UnixMicro(),
			SpanID:           span.GetSpanID(),
			ParentID:         span.GetParentID(),
			TraceID:          span.GetTraceID(),
			Duration:         span.GetDuration(),
			WorkspaceID:      span.GetSpaceID(),
			SpanName:         span.GetSpanName(),
			SpanType:         span.GetSpanType(),
			StatusCode:       span.GetStatusCode(),
			Input:            putContentMap[tracespec.Input],
			Output:           putContentMap[tracespec.Output],
			ObjectStorage:    objectStorageByte,
			SystemTagsString: systemTagStrM,
			SystemTagsLong:   systemTagLongM,
			SystemTagsDouble: systemTagDoubleM,
			TagsString:       tagStrM,
			TagsLong:         tagLongM,
			TagsDouble:       tagDoubleM,
		})
	}

	return resSpan, resFile
}

func parseTag(spanTag map[string]interface{}) (map[string]string, map[string]int64, map[string]float64) {
	if len(spanTag) == 0 {
		return nil, nil, nil
	}

	vStrMap := make(map[string]string)
	vLongMap := make(map[string]int64)
	vDoubleMap := make(map[string]float64)
	for key, value := range spanTag {
		if key == tracespec.Input || key == tracespec.Output {
			continue
		}
		switch v := value.(type) {
		case string:
			vStrMap[key] = v
		case int:
			vLongMap[key] = int64(v)
		case uint:
			vLongMap[key] = int64(v)
		case int8:
			vLongMap[key] = int64(v)
		case uint8:
			vLongMap[key] = int64(v)
		case int16:
			vLongMap[key] = int64(v)
		case uint16:
			vLongMap[key] = int64(v)
		case int32:
			vLongMap[key] = int64(v)
		case uint32:
			vLongMap[key] = int64(v)
		case int64:
			vLongMap[key] = v
		case uint64:
			vLongMap[key] = int64(v)
		case float32:
			vDoubleMap[key] = float64(v)
		case float64:
			vDoubleMap[key] = v
		default:
			vStrMap[key] = util.Stringify(value)
		}
	}

	return vStrMap, vLongMap, vDoubleMap
}

var (
	tagValueConverterMap = map[string]*tagValueConverter{
		tracespec.Input: {
			convertFunc: convertInput,
		},
		tracespec.Output: {
			convertFunc: convertOutput,
		},
	}
)

type tagValueConverter struct {
	convertFunc func(ctx context.Context, spanKey string, span *Span) (valueRes string, uploadFile []*UploadFile, err error)
}

func convertInput(ctx context.Context, spanKey string, span *Span) (valueRes string, uploadFile []*UploadFile, err error) {
	value, ok := span.TagMap[spanKey]
	if !ok {
		return
	}

	uploadFile = make([]*UploadFile, 0)
	if _, ok := span.multiModalityKeyMap[spanKey]; !ok {
		// input/output is just text string
		var f *UploadFile
		valueRes, f = transferText(fmt.Sprintf("%v", value), span, spanKey)
		if f != nil {
			uploadFile = append(uploadFile, f)
		}
	} else {
		// multi-modality input/output
		modelInput := &tracespec.ModelInput{}
		if tempV, ok := value.(string); ok {
			if err = json.Unmarshal([]byte(tempV), modelInput); err != nil {
				logger.CtxErrorf(ctx, "unmarshal ModelInput failed, err: %v", err)
				return valueRes, nil, err
			}
		}
		for _, message := range modelInput.Messages {
			for _, part := range message.Parts {
				fs := transferMessagePart(part, span, spanKey)
				uploadFile = append(uploadFile, fs...)
			}
		}
		tempV, err := json.Marshal(modelInput)
		if err != nil {
			logger.CtxErrorf(ctx, "marshal multiModalityContent failed, err: %v", err)
			return valueRes, nil, err
		}
		valueRes = string(tempV)

		// If the content is still too long, truncate it, and
		// decide whether to report the oversized content based on the UltraLargeReport option.
		if len(valueRes) > consts.MaxBytesOfOneTagValueOfInputOutput {
			var f *UploadFile
			valueRes, f = transferText(valueRes, span, spanKey)
			if f != nil {
				uploadFile = append(uploadFile, f)
			}
		}
	}

	return
}

func convertOutput(ctx context.Context, spanKey string, span *Span) (valueRes string, uploadFile []*UploadFile, err error) {
	value, ok := span.TagMap[spanKey]
	if !ok {
		return
	}

	uploadFile = make([]*UploadFile, 0)
	if _, ok := span.multiModalityKeyMap[spanKey]; !ok {
		// input/output is just text string
		var f *UploadFile
		valueRes, f = transferText(fmt.Sprintf("%v", value), span, spanKey)
		uploadFile = append(uploadFile, f)
	} else {
		// multi-modality input/output
		modelOutput := &tracespec.ModelOutput{}
		if tempV, ok := value.(string); ok {
			if err = json.Unmarshal([]byte(tempV), modelOutput); err != nil {
				logger.CtxErrorf(ctx, "unmarshal ModelInput failed, err: %v", err)
				return valueRes, nil, err
			}
		}
		for _, choice := range modelOutput.Choices {
			if choice == nil || choice.Message == nil {
				continue
			}
			for _, part := range choice.Message.Parts {
				files := transferMessagePart(part, span, spanKey)
				uploadFile = append(uploadFile, files...)
			}
		}
		tempV, err := json.Marshal(modelOutput)
		if err != nil {
			logger.CtxErrorf(ctx, "marshal multiModalityContent failed, err: %v", err)
			return valueRes, nil, err
		}
		valueRes = string(tempV)

		// If the content is still too long, truncate it, and
		// decide whether to report the oversized content based on the UltraLargeReport option.
		if len(valueRes) > consts.MaxBytesOfOneTagValueOfInputOutput {
			var f *UploadFile
			valueRes, f = transferText(valueRes, span, spanKey)
			if f != nil {
				uploadFile = append(uploadFile, f)
			}
		}
	}

	return
}

func parseInputOutput(ctx context.Context, span *Span) (spanUploadFiles []*UploadFile, putContentMap map[string]string, err error) {
	if span == nil {
		return
	}
	spanUploadFiles = make([]*UploadFile, 0)
	putContentMap = make(map[string]string)

	for key, converter := range tagValueConverterMap {
		if _, ok := span.GetTagMap()[key]; !ok {
			continue
		}
		newInput, inputFiles, err := converter.convertFunc(ctx, key, span)
		if err != nil {
			return nil, nil, err
		}
		putContentMap[key] = newInput
		spanUploadFiles = append(spanUploadFiles, inputFiles...)
	}

	return
}

func transferObjectStorage(spanUploadFile []*UploadFile) (string, error) {
	objectStorage := model2.ObjectStorage{
		Attachments: make([]*model2.Attachment, 0),
	}
	isExist := false
	for _, file := range spanUploadFile {
		if file == nil {
			continue
		}
		isExist = true
		switch file.UploadType {
		case model2.UploadTypeLong:
			if file.TagKey == tracespec.Input {
				objectStorage.InputTosKey = file.TosKey
			} else if file.TagKey == tracespec.Output {
				objectStorage.OutputTosKey = file.TosKey
			}
		case model2.UploadTypeMultiModality:
			objectStorage.Attachments = append(objectStorage.Attachments, &model2.Attachment{
				Field:  file.TagKey,
				Name:   file.Name,
				Type:   file.FileType,
				TosKey: file.TosKey,
			})
		}
	}
	if !isExist {
		return "", nil
	}
	objectStorageByte, err := json.Marshal(objectStorage)
	if err != nil {
		return "", nil
	}

	return string(objectStorageByte), nil
}

func transferMessagePart(src *tracespec.ModelMessagePart, span *Span, tagKey string) (uploadFiles []*UploadFile) {
	if src == nil || span == nil {
		return nil
	}

	switch src.Type {
	case tracespec.ModelMessagePartTypeImage:
		if f := transferImage(src.ImageURL, span, tagKey); f != nil {
			uploadFiles = append(uploadFiles, f)
		}
	case tracespec.ModelMessagePartTypeFile:
		if f := transferFile(src.FileURL, span, tagKey); f != nil {
			uploadFiles = append(uploadFiles, f)
		}
	case tracespec.ModelMessagePartTypeText:
		return
	default:
		return
	}

	return
}

func transferText(src string, span *Span, tagKey string) (string, *UploadFile) {
	if len(src) == 0 {
		return "", nil
	}

	if !span.UltraLargeReport() {
		return src, nil
	}

	if len(src) > consts.MaxBytesOfOneTagValueOfInputOutput {
		//key := "traceid/spanid/tagkey/filetype/large_text"
		key := fmt.Sprintf(KeyTemplateLargeText, span.GetTraceID(), span.GetSpanID(), tagKey, fileTypeText)
		return util.TruncateStringByChar(src, consts.TextTruncateCharLength), &UploadFile{
			TosKey:     key,
			Data:       src,
			UploadType: model2.UploadTypeLong,
			TagKey:     tagKey,
			FileType:   fileTypeText,
			SpaceID:    span.GetSpaceID(),
		}
	}

	return src, nil
}

func transferImage(src *tracespec.ModelImageURL, span *Span, tagKey string) *UploadFile {
	if src == nil || span == nil {
		return nil
	}
	if isValidURL := util.IsValidURL(src.URL); isValidURL {
		return nil
	}

	//key := "traceid_spanid_tagkey_filetype_randomid"
	key := fmt.Sprintf(KeyTemplateMultiModality, span.GetTraceID(), span.GetSpanID(), tagKey, fileTypeImage, util.Gen16CharID())
	bin, _ := base64.StdEncoding.DecodeString(src.URL)
	src.URL = key
	return &UploadFile{
		TosKey:     key,
		Data:       string(bin),
		UploadType: model2.UploadTypeMultiModality,
		TagKey:     tagKey,
		Name:       src.Name,
		FileType:   fileTypeImage,
		SpaceID:    span.GetSpaceID(),
	}
}

func transferFile(src *tracespec.ModelFileURL, span *Span, tagKey string) *UploadFile {
	if src == nil || span == nil {
		return nil
	}
	if isValidURL := util.IsValidURL(src.URL); isValidURL {
		return nil
	}

	//key := "traceid/spanid/tagkey/filetype/randomid"
	key := fmt.Sprintf(KeyTemplateMultiModality, span.GetTraceID(), span.GetSpanID(), tagKey, fileTypeFile, util.Gen16CharID())
	bin, _ := base64.StdEncoding.DecodeString(src.URL)
	src.URL = key
	return &UploadFile{
		TosKey:     key,
		Data:       string(bin),
		UploadType: model2.UploadTypeMultiModality,
		TagKey:     tagKey,
		Name:       src.Name,
		FileType:   fileTypeFile,
		SpaceID:    span.GetSpaceID(),
	}
}

type UploadSpanData struct {
	Spans []*UploadSpan `json:"spans"`
}

type UploadSpan struct {
	StartedATMicros  int64              `json:"started_at_micros"`
	SpanID           string             `json:"span_id"`
	ParentID         string             `json:"parent_id"`
	TraceID          string             `json:"trace_id"`
	Duration         int64              `json:"duration"`
	WorkspaceID      string             `json:"workspace_id"`
	SpanName         string             `json:"span_name"`
	SpanType         string             `json:"span_type"`
	StatusCode       int32              `json:"status_code"`
	Input            string             `json:"input"`
	Output           string             `json:"output"`
	ObjectStorage    string             `json:"object_storage"`
	SystemTagsString map[string]string  `json:"system_tags_string"`
	SystemTagsLong   map[string]int64   `json:"system_tags_long"`
	SystemTagsDouble map[string]float64 `json:"system_tags_double"`
	TagsString       map[string]string  `json:"tags_string"`
	TagsLong         map[string]int64   `json:"tags_long"`
	TagsDouble       map[string]float64 `json:"tags_double"`
}

type UploadFile struct {
	TosKey     string
	Data       string
	UploadType model2.UploadType
	TagKey     string
	Name       string
	FileType   string
	SpaceID    string
}
