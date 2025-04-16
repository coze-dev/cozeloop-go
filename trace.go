// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package cozeloop

import (
	"context"
	"time"

	"github.com/coze-dev/cozeloop-go/internal/trace"
)

type TraceClient interface {
	// StartSpan Generate a span that automatically links to the previous span in the context.
	// The start time of the span starts counting from the call of StartSpan.
	// The generated span will be automatically written into the context.
	// Subsequent spans that need to be chained should call StartSpan based on the new context.
	StartSpan(ctx context.Context, name, spanType string, opts ...StartSpanOption) (context.Context, Span)
	// GetSpanFromContext Get the span from the context.
	GetSpanFromContext(ctx context.Context) Span
	// GetSpanFromHeader Get the span from the header.
	GetSpanFromHeader(ctx context.Context, header map[string]string) SpanContext
	// Flush Force the reporting of spans in the queue.
	Flush(ctx context.Context)
}

type startSpanOptions = trace.StartSpanOptions

// StartSpanOption is used to set options for the span.
type StartSpanOption = func(o *startSpanOptions)

// WithStartTime Set the start time of the span.
// This field is optional. If not specified, the time when StartSpan is called will be used as the default.
func WithStartTime(t time.Time) StartSpanOption {
	return func(ops *startSpanOptions) {
		ops.StartTime = t
	}
}

// WithChildOf Set the parent span of the span.
// This field is optional. If not specified, the parent span will
// be looked up from the context. If not found, the current span will have no parent.
func WithChildOf(s SpanContext) StartSpanOption {
	return func(ops *startSpanOptions) {
		if s == nil {
			return
		}
		if spanID := s.GetSpanID(); spanID != "" {
			ops.ParentSpanID = spanID
		}
		if traceID := s.GetTraceID(); traceID != "" {
			ops.TraceID = traceID
		}
		if baggage := s.GetBaggage(); len(baggage) > 0 {
			ops.Baggage = baggage
		}
	}
}

// WithStartNewTrace Set the parent span of the span.
// This field is optional. If specified, start a span of a new trace.
func WithStartNewTrace() StartSpanOption {
	return func(ops *startSpanOptions) {
		ops.StartNewTrace = true
	}
}

func WithWorkSpaceID(workSpaceID string) StartSpanOption {
	return func(ops *startSpanOptions) {
		ops.WorkSpaceID = workSpaceID
	}
}
