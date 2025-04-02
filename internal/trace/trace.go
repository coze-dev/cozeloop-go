// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

import (
	"context"
	"sync"
	"time"

	"github.com/coze-dev/cozeloop-go/attribute/trace"
	"github.com/coze-dev/cozeloop-go/internal/consts"
	"github.com/coze-dev/cozeloop-go/internal/httpclient"
	"github.com/coze-dev/cozeloop-go/internal/logger"
	"github.com/coze-dev/cozeloop-go/internal/util"
)

type Provider struct {
	httpClient    *httpclient.Client
	opt           *Options
	spanProcessor SpanProcessor
}

type Options struct {
	WorkspaceID      string
	UltraLargeReport bool
}

type StartSpanOptions struct {
	StartTime     time.Time
	ParentSpanID  string
	TraceID       string
	Baggage       map[string]string
	StartNewTrace bool
	Scene         string
}

type loopSpanKey struct{}

func NewTraceProvider(httpClient *httpclient.Client, options Options) *Provider {
	c := &Provider{
		httpClient:    httpClient,
		opt:           &options,
		spanProcessor: NewBatchSpanProcessor(httpClient),
	}
	return c
}

func (t *Provider) GetOpts() *Options {
	return t.opt
}

func (t *Provider) StartSpan(ctx context.Context, name, spanType string, opts StartSpanOptions) (context.Context, *Span, error) {
	// 0. check param
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
		systemTagMap[trace.Runtime_] = trace.Runtime{
			Scene: options.Scene,
		}
	}

	// 2. create span and init
	s := &Span{
		SpanContext: SpanContext{
			SpanID:  util.Gen16CharID(),
			TraceID: traceID,
			Baggage: make(map[string]string),
		},
		SpanType:            spanType,
		Name:                spanName,
		WorkspaceID:         t.opt.WorkspaceID,
		ParentSpanID:        parentID,
		StartTime:           startTime,
		Duration:            0,
		TagMap:              make(map[string]interface{}),
		SystemTagMap:        systemTagMap,
		StatusCode:          0,
		ultraLargeReport:    t.opt.UltraLargeReport,
		multiModalityKeyMap: make(map[string]struct{}),
		spanProcessor:       t.spanProcessor,
		flags:               0,
		isFinished:          0,
		lock:                sync.RWMutex{},
		bytesSize:           0, // The initial value is 0. Default fields do not count towards the size.
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
