// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

import (
	"context"
	"sync"
	"time"

	"github.com/coze-dev/cozeloop-go/internal/consts"
	"github.com/coze-dev/cozeloop-go/internal/httpclient"
	"github.com/coze-dev/cozeloop-go/internal/logger"
	"github.com/coze-dev/cozeloop-go/internal/util"
	"github.com/coze-dev/cozeloop-go/spec/tracespec"
)

type Provider struct {
	httpClient    *httpclient.Client
	opt           *Options
	spanProcessor SpanProcessor
}

type Options struct {
	WorkspaceID          string
	UltraLargeReport     bool
	Exporter             Exporter
	FinishEventProcessor func(ctx context.Context, info *consts.FinishEventInfo)
	TagTruncateConf      *TagTruncateConf
	SpanUploadPath       string
	FileUploadPath       string
	QueueConf            *QueueConf
}

type StartSpanOptions struct {
	StartTime     time.Time
	ParentSpanID  string
	SpanID        string
	TraceID       string
	Baggage       map[string]string
	StartNewTrace bool
	Scene         string
	WorkspaceID   string
}

type loopSpanKey struct{}

func NewTraceProvider(httpClient *httpclient.Client, options Options) *Provider {
	var uploadPath *UploadPath
	if options.SpanUploadPath != "" || options.FileUploadPath != "" {
		uploadPath = &UploadPath{
			spanUploadPath: options.SpanUploadPath,
			fileUploadPath: options.FileUploadPath,
		}
	}
	c := &Provider{
		httpClient: httpClient,
		opt:        &options,
		spanProcessor: NewBatchSpanProcessor(
			options.Exporter,
			httpClient,
			uploadPath,
			options.FinishEventProcessor,
			options.QueueConf,
		),
	}
	return c
}

func (t *Provider) GetOpts() *Options {
	return t.opt
}

func (t *Provider) StartSpan(ctx context.Context, name, spanType string, opts StartSpanOptions) (context.Context, *Span, error) {
	// 0. check param
	if name == "" {
		name = "unknown"
	}
	if spanType == "" {
		spanType = "unknown"
	}
	if len(name) > consts.MaxBytesOfOneTagValueDefault {
		logger.CtxWarnf(ctx, "Name is too long, will be truncated to %d bytes, original name: %s", consts.MaxBytesOfOneTagValueDefault, name)
		name = name[:consts.MaxBytesOfOneTagValueDefault]
	}
	if len(spanType) > consts.MaxBytesOfOneTagValueDefault {
		logger.CtxWarnf(ctx, "SpanType is too long, will be truncated to %d bytes, original span type: %s", consts.MaxBytesOfOneTagValueDefault, spanType)
		spanType = spanType[:consts.MaxBytesOfOneTagValueDefault]
	}

	// 1. get data from parent span
	// Prioritize using the data from opts, and fall back to parentSpan
	parentSpan := t.GetSpanFromContext(ctx)
	if parentSpan != nil && !opts.StartNewTrace {
		if opts.TraceID == "" {
			opts.TraceID = parentSpan.GetTraceID()
		}
		if opts.ParentSpanID == "" {
			opts.ParentSpanID = parentSpan.GetSpanID()
		}
		if opts.Baggage == nil {
			opts.Baggage = parentSpan.GetBaggage()
		}
	}

	// 2. internal start span
	loopSpan := t.startSpan(ctx, name, spanType, opts)

	// 3. inject ctx
	ctx = context.WithValue(ctx, loopSpanKey{}, loopSpan)

	return ctx, loopSpan, nil
}

func (t *Provider) GetSpanFromContext(ctx context.Context) *Span {
	s, ok := ctx.Value(loopSpanKey{}).(*Span)
	if !ok {
		return nil
	}

	return s
}

func (t *Provider) GetSpanFromHeader(ctx context.Context, header map[string]string) *SpanContext {
	return FromHeader(ctx, header)
}

func (t *Provider) startSpan(ctx context.Context, spanName string, spanType string, options StartSpanOptions) *Span {
	// 1. pack base data
	// get parentID from opt first, or set it to "0".
	// get TraceID from opt first, or generate new ID.
	parentID := "0"
	if options.ParentSpanID != "" {
		parentID = options.ParentSpanID
	}

	spanID := options.SpanID
	if len(spanID) == 0 {
		spanID = util.Gen16CharID()
	}

	traceID := ""
	if options.TraceID != "" {
		traceID = options.TraceID
	} else {
		traceID = util.Gen32CharID()
	}

	startTime := time.Now()
	if !options.StartTime.IsZero() {
		startTime = options.StartTime
	}

	systemTagMap := make(map[string]interface{})
	if options.Scene != "" {
		systemTagMap[tracespec.Runtime_] = tracespec.Runtime{
			Scene: options.Scene,
		}
	}

	workSpaceID := t.opt.WorkspaceID
	if options.WorkspaceID != "" {
		workSpaceID = options.WorkspaceID
	}

	// 2. create span and init
	s := &Span{
		SpanContext: SpanContext{
			SpanID:  spanID,
			TraceID: traceID,
			Baggage: make(map[string]string),
		},
		SpanType:            spanType,
		Name:                spanName,
		WorkspaceID:         workSpaceID,
		ParentSpanID:        parentID,
		StartTime:           startTime,
		Duration:            0,
		TagMap:              make(map[string]interface{}),
		SystemTagMap:        systemTagMap,
		StatusCode:          0,
		ultraLargeReport:    t.opt.UltraLargeReport,
		multiModalityKeyMap: make(map[string]struct{}),
		spanProcessor:       t.spanProcessor,
		flags:               1, // for W3C, sampled by default
		isFinished:          0,
		lock:                sync.RWMutex{},
		bytesSize:           0, // The initial value is 0. Default fields do not count towards the size.
		tagTruncateConf:     t.opt.TagTruncateConf,
	}

	// 3. set Baggage from parent span
	s.setBaggage(ctx, options.Baggage, false)

	return s
}

func (t *Provider) Flush(ctx context.Context) {
	_ = t.spanProcessor.ForceFlush(ctx)
}

func (t *Provider) CloseTrace(ctx context.Context) {
	_ = t.spanProcessor.Shutdown(ctx)
}

func DefaultFinishEventProcessor(ctx context.Context, info *consts.FinishEventInfo) {
	if info == nil {
		return
	}
	if info.IsEventFail {
		logger.CtxErrorf(ctx, "finish_event[%s] fail, msg: %s", info.EventType, info.DetailMsg)
	} else {
		logger.CtxDebugf(ctx, "finish_event[%s] success, item_num: %d, msg: %s", info.EventType, info.ItemNum, info.DetailMsg)
	}
}
