// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

import (
	"context"
	"testing"
	"time"

	. "github.com/bytedance/mockey"
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

func Test_NoSample(t *testing.T) {
	PatchConvey("Test NoSample spans", t, func() {
		provider := &Provider{
			httpClient: &httpclient.Client{},
			opt: &Options{
				WorkspaceID:      "workspace-id",
				UltraLargeReport: true,
			},
		}
		ctx := context.Background()

		// 1. Create a normal span
		normalCtx, normalSpan, err := provider.StartSpan(ctx, "normal-span", "test-type", StartSpanOptions{
			StartTime: time.Now(),
		})
		So(err, ShouldBeNil)
		So(normalSpan, ShouldNotEqual, DefaultNoopSpan)
		So(normalSpan, ShouldNotBeNil)

		// 2. Create a NoSample span
		noSampleCtx, noSampleSpan, err := provider.StartSpan(normalCtx, "nosample-span", "test-type", StartSpanOptions{
			StartTime: time.Now(),
			NoSample:  true,
		})
		So(err, ShouldBeNil)
		So(noSampleSpan, ShouldEqual, DefaultNoopSpan)

		// 3. Create a child span from NoSample span - should also be NoopSpan
		childCtx, childSpan, err := provider.StartSpan(noSampleCtx, "child-span", "test-type", StartSpanOptions{
			StartTime: time.Now(),
		})
		So(err, ShouldBeNil)
		So(childSpan, ShouldEqual, DefaultNoopSpan)
		So(childCtx, ShouldNotBeNil)
	})
}
