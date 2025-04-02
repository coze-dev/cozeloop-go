// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

import (
	"context"
	"testing"
	"time"

	"github.com/coze-dev/cozeloop-go/internal/httpclient"
	. "github.com/smartystreets/goconvey/convey"
)

func Test_StartSpan(t *testing.T) {
	ctx := context.Background()
	name, spanType := "test-span", "test-type"
	opts := StartSpanOptions{
		StartTime:    time.Now(),
		ParentSpanID: "parent-span-id",
		TraceID:      "trace-id",
		Baggage:      map[string]string{"key": "value"},
	}

	PatchConvey("Test buildLoopSpanImpl returns error", t, func() {
		t := &Provider{
			httpClient: &httpclient.Client{},
			opt: &Options{
				WorkspaceID:      "workspace-id",
				UltraLargeReport: true,
			},
		}
		actualCtx, actualSpan, err := t.StartSpan(ctx, name, spanType, opts)
		So(actualCtx, ShouldNotBeNil)
		So(actualSpan, ShouldNotBeNil)
		So(err, ShouldBeNil)
	})
}

func Test_GetSpanFromHeader(t *testing.T) {
	ctx := context.Background()
	name, spanType := "test-span", "test-type"
	opts := StartSpanOptions{
		StartTime:    time.Now(),
		ParentSpanID: "1433434",
		TraceID:      "1111111111111",
		Baggage:      map[string]string{"key": "value"},
	}
	PatchConvey("Test FromHeader failed", t, func() {
		t := &Provider{
			httpClient: &httpclient.Client{},
			opt: &Options{
				WorkspaceID:      "workspace-id",
				UltraLargeReport: true,
			},
		}
		_, actualSpan, err := t.StartSpan(ctx, name, spanType, opts)
		So(err, ShouldBeNil)
		header, err := actualSpan.ToHeader()
		So(err, ShouldBeNil)
		spanFromHeader := t.GetSpanFromHeader(ctx, header)
		So(spanFromHeader.GetSpanID(), ShouldEqual, actualSpan.GetSpanID())
	})

	PatchConvey("Test FromHeader success", t, func() {
		t := &Provider{}
		expectedSpan := &Span{
			SpanContext: SpanContext{
				TraceID: "1234567890",
				SpanID:  "0987654321",
			},
		}
		Mock(FromHeader).Return(expectedSpan).Build()
		actual := t.GetSpanFromHeader(nil, nil)
		So(actual, ShouldEqual, expectedSpan)
	})
}
