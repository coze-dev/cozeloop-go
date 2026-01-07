// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/consts"
	"github.com/coze-dev/cozeloop-go/internal/httpclient"
	"github.com/coze-dev/cozeloop-go/internal/logger"
	model2 "github.com/coze-dev/cozeloop-go/internal/trace/model"
	"github.com/coze-dev/cozeloop-go/internal/util"
	"github.com/coze-dev/cozeloop-go/spec/tracespec"
)

type Exporter interface {
	ExportSpans(ctx context.Context, spans []*entity.UploadSpan) error
	ExportFiles(ctx context.Context, files []*entity.UploadFile) error
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
	client     *httpclient.Client
	uploadPath UploadPath
}

type UploadPath struct {
	spanUploadPath string
	fileUploadPath string
}

func (e *SpanExporter) ExportFiles(ctx context.Context, files []*entity.UploadFile) error {
	uploadFiles := files
	for _, file := range uploadFiles {
		if file == nil {
			continue
		}
		logger.CtxDebugf(ctx, "uploadFile start, file name: %s", file.Name)
		resp := httpclient.BaseResponse{}
		err := e.client.UploadFile(ctx, e.uploadPath.fileUploadPath, file.TosKey, bytes.NewReader([]byte(file.Data)), map[string]string{"workspace_id": file.SpaceID}, &resp)
		if err != nil {
			return consts.NewError(fmt.Sprintf("export files[%s] fail", file.TosKey)).Wrap(err)
		}
		if resp.GetCode() != 0 { // todo: some err code do not need retry
			return consts.NewError(fmt.Sprintf("export files[%s] fail, code:[%v], msg:[%v] retry later", file.TosKey, resp.GetCode(), resp.GetMsg()))
		}
		logger.CtxDebugf(ctx, "uploadFile end, file name: %s", file.Name)
	}

	return nil
}

func (e *SpanExporter) ExportSpans(ctx context.Context, ss []*entity.UploadSpan) (err error) {
	if len(ss) == 0 {
		return
	}
	resp := httpclient.BaseResponse{}
	err = e.client.Post(ctx, e.uploadPath.spanUploadPath, UploadSpanData{ss}, &resp)
	if err != nil {
		return consts.NewError(fmt.Sprintf("export spans fail, span count: [%d]", len(ss))).Wrap(err)
	}
	if resp.GetCode() != 0 { // todo: some err code do not need retry
		return consts.NewError(fmt.Sprintf("export spans fail, span count: [%d], code:[%v], msg:[%v]", len(ss), resp.GetCode(), resp.GetMsg()))
	}

	return
}

func transferToUploadSpanAndFile(ctx context.Context, spans []*Span) ([]*entity.UploadSpan, []*entity.UploadFile) {
	resSpan := make([]*entity.UploadSpan, 0, len(spans))
	resFile := make([]*entity.UploadFile, 0, len(spans))

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

		tagStrM, tagLongM, tagDoubleM, tagBoolM := parseTag(span.GetTagMap(), false)
		systemTagStrM, systemTagLongM, systemTagDoubleM, _ := parseTag(span.SystemTagMap, true)
		resSpan = append(resSpan, &entity.UploadSpan{
			StartedATMicros:  span.GetStartTime().UnixMicro(),
			LogID:            span.GetLogID(),
			SpanID:           span.GetSpanID(),
			ParentID:         span.GetParentID(),
			TraceID:          span.GetTraceID(),
			DurationMicros:   span.GetDuration(),
			ServiceName:      span.GetServiceName(),
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
			TagsBool:         tagBoolM,
		})
	}

	return resSpan, resFile
}

func parseTag(spanTag map[string]interface{}, isSystemTag bool) (map[string]string, map[string]int64, map[string]float64, map[string]bool) {
	if len(spanTag) == 0 {
		return nil, nil, nil, nil
	}

	vStrMap := make(map[string]string)
	vLongMap := make(map[string]int64)
	vDoubleMap := make(map[string]float64)
	vBoolMap := make(map[string]bool)
	for key, value := range spanTag {
		if key == tracespec.Input || key == tracespec.Output {
			continue
		}
		switch v := value.(type) {
		case bool:
			if isSystemTag {
				vStrMap[key] = util.Stringify(value)
			} else {
				vBoolMap[key] = v
			}
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

	return vStrMap, vLongMap, vDoubleMap, vBoolMap
}

var tagValueConverterMap = map[string]*tagValueConverter{
	tracespec.Input: {
		convertFunc: convertInput,
	},
	tracespec.Output: {
		convertFunc: convertOutput,
	},
}

type tagValueConverter struct {
	convertFunc func(ctx context.Context, spanKey string, span *Span) (valueRes string, uploadFile []*entity.UploadFile, err error)
}

func convertInput(ctx context.Context, spanKey string, span *Span) (valueRes string, uploadFile []*entity.UploadFile, err error) {
	value, ok := span.TagMap[spanKey]
	if !ok {
		return
	}

	uploadFile = make([]*entity.UploadFile, 0)
	if _, ok := span.multiModalityKeyMap[spanKey]; !ok {
		// input/output is just text string
		var f *entity.UploadFile
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
			var f *entity.UploadFile
			valueRes, f = transferText(valueRes, span, spanKey)
			if f != nil {
				uploadFile = append(uploadFile, f)
			}
		}
	}

	return
}

func convertOutput(ctx context.Context, spanKey string, span *Span) (valueRes string, uploadFile []*entity.UploadFile, err error) {
	value, ok := span.TagMap[spanKey]
	if !ok {
		return
	}

	uploadFile = make([]*entity.UploadFile, 0)
	if _, ok := span.multiModalityKeyMap[spanKey]; !ok {
		// input/output is just text string
		var f *entity.UploadFile
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
			var f *entity.UploadFile
			valueRes, f = transferText(valueRes, span, spanKey)
			if f != nil {
				uploadFile = append(uploadFile, f)
			}
		}
	}

	return
}

func parseInputOutput(ctx context.Context, span *Span) (spanUploadFiles []*entity.UploadFile, putContentMap map[string]string, err error) {
	if span == nil {
		return
	}
	spanUploadFiles = make([]*entity.UploadFile, 0)
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

func transferObjectStorage(spanUploadFile []*entity.UploadFile) (string, error) {
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
		case entity.UploadTypeLong:
			if file.TagKey == tracespec.Input {
				objectStorage.InputTosKey = file.TosKey
			} else if file.TagKey == tracespec.Output {
				objectStorage.OutputTosKey = file.TosKey
			}
		case entity.UploadTypeMultiModality:
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

func transferMessagePart(src *tracespec.ModelMessagePart, span *Span, tagKey string) (uploadFiles []*entity.UploadFile) {
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

func transferText(src string, span *Span, tagKey string) (string, *entity.UploadFile) {
	if len(src) == 0 {
		return "", nil
	}

	if !span.UltraLargeReport() {
		return src, nil
	}

	if len(src) > consts.MaxBytesOfOneTagValueOfInputOutput {
		// key := "traceid/spanid/tagkey/filetype/large_text"
		key := fmt.Sprintf(KeyTemplateLargeText, span.GetTraceID(), span.GetSpanID(), tagKey, fileTypeText)
		return util.TruncateStringByChar(src, consts.TextTruncateCharLength), &entity.UploadFile{
			TosKey:     key,
			Data:       src,
			UploadType: entity.UploadTypeLong,
			TagKey:     tagKey,
			FileType:   fileTypeText,
			SpaceID:    span.GetSpaceID(),
		}
	}

	return src, nil
}

func transferImage(src *tracespec.ModelImageURL, span *Span, tagKey string) *entity.UploadFile {
	if src == nil || span == nil {
		return nil
	}
	if isValidURL := util.IsValidURL(src.URL); isValidURL {
		return nil
	}

	// key := "traceid_spanid_tagkey_filetype_randomid"
	key := fmt.Sprintf(KeyTemplateMultiModality, span.GetTraceID(), span.GetSpanID(), tagKey, fileTypeImage, util.Gen16CharID())
	bin, _ := base64.StdEncoding.DecodeString(src.URL)
	src.URL = key
	return &entity.UploadFile{
		TosKey:     key,
		Data:       string(bin),
		UploadType: entity.UploadTypeMultiModality,
		TagKey:     tagKey,
		Name:       src.Name,
		FileType:   fileTypeImage,
		SpaceID:    span.GetSpaceID(),
	}
}

func transferFile(src *tracespec.ModelFileURL, span *Span, tagKey string) *entity.UploadFile {
	if src == nil || span == nil {
		return nil
	}
	if isValidURL := util.IsValidURL(src.URL); isValidURL {
		return nil
	}

	// key := "traceid/spanid/tagkey/filetype/randomid"
	key := fmt.Sprintf(KeyTemplateMultiModality, span.GetTraceID(), span.GetSpanID(), tagKey, fileTypeFile, util.Gen16CharID())
	bin, _ := base64.StdEncoding.DecodeString(src.URL)
	src.URL = key
	return &entity.UploadFile{
		TosKey:     key,
		Data:       string(bin),
		UploadType: entity.UploadTypeMultiModality,
		TagKey:     tagKey,
		Name:       src.Name,
		FileType:   fileTypeFile,
		SpaceID:    span.GetSpaceID(),
	}
}

type UploadSpanData struct {
	Spans []*entity.UploadSpan `json:"spans"`
}
