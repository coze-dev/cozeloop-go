// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/bytedance/mockey"
	"github.com/coze-dev/cozeloop-go/internal/consts"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewBatchSpanProcessor(t *testing.T) {
	for _, tc := range []struct {
		name        string
		conf        *QueueConf
		concurrency int
	}{
		{name: "unset", concurrency: 1},
		{name: "zero", conf: &QueueConf{}, concurrency: 1},
		{name: "negative", conf: &QueueConf{ExportConcurrency: -3}, concurrency: 1},
		{name: "one", conf: &QueueConf{ExportConcurrency: 1}, concurrency: 1},
		{name: "concurrent", conf: &QueueConf{ExportConcurrency: 4, SpanQueueLength: 16, SpanMaxExportBatchLength: 7}, concurrency: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			processor := NewBatchSpanProcessor(nil, nil, nil, nil, tc.conf).(*BatchSpanProcessor)
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := processor.Shutdown(ctx); err != nil {
					t.Error(err)
				}
			})
			for _, queue := range []QueueManager{processor.spanQM, processor.spanRetryQM, processor.fileQM, processor.fileRetryQM} {
				q := queue.(*BatchQueueManager)
				if q.o.exportConcurrency != tc.concurrency {
					t.Errorf("%s concurrency = %d, want %d", q.o.queueName, q.o.exportConcurrency, tc.concurrency)
				}
			}
			spanQueue := processor.spanQM.(*BatchQueueManager)
			wantCapacity, wantBatch := DefaultMaxQueueLength, DefaultMaxExportBatchLength
			if tc.name == "concurrent" {
				wantCapacity, wantBatch = 16, 7
			}
			if cap(spanQueue.queue) != wantCapacity || spanQueue.o.maxExportBatchLength != wantBatch {
				t.Errorf("span capacity/batch = %d/%d, want %d/%d",
					cap(spanQueue.queue), spanQueue.o.maxExportBatchLength, wantCapacity, wantBatch)
			}
		})
	}
}

func TestBatchQueueManagerEnqueue(t *testing.T) {
	for _, queueName := range []string{queueNameSpan, queueNameSpanRetry, queueNameFile, queueNameFileRetry} {
		t.Run(queueName, func(t *testing.T) {
			var events []*consts.FinishEventInfo
			queue := &BatchQueueManager{
				o: batchQueueManagerOptions{
					queueName:         queueName,
					exportConcurrency: 2,
					finishEventProcessor: func(ctx context.Context, info *consts.FinishEventInfo) {
						events = append(events, info)
					},
				},
				// No consumer: make a full queue deterministic.
				queue: make(chan interface{}, 1),
			}
			span := &Span{ParentSpanID: "0"}
			queue.Enqueue(context.Background(), span, 123)
			queue.Enqueue(context.Background(), span, 456)
			if atomic.LoadUint32(&queue.dropped) != 1 {
				t.Fatalf("dropped = %d, want 1", queue.dropped)
			}
			item := (<-queue.queue).(concurrentQueueItem)
			if item.value != span || item.byteSize != 123 {
				t.Fatalf("queued item = %+v, want original span and 123 bytes", item)
			}
			if len(events) != 2 || events[0].IsEventFail || !events[1].IsEventFail {
				t.Fatalf("unexpected enqueue events: %+v", events)
			}
			wantEvent := consts.SpanFinishEventFileQueueEntryRate
			if queueName == queueNameSpan || queueName == queueNameSpanRetry {
				wantEvent = consts.SpanFinishEventSpanQueueEntryRate
			}
			for _, event := range events {
				if event.EventType != wantEvent || event.QueueName != queueName || event.ItemNum != 1 {
					t.Fatalf("unexpected event: %+v", event)
				}
				if wantEvent == consts.SpanFinishEventSpanQueueEntryRate &&
					(event.ExtraParams == nil || !event.ExtraParams.IsRootSpan) {
					t.Fatalf("root span metadata missing: %+v", event)
				}
			}
			atomic.StoreInt32(&queue.stopped, 1)
			queue.Enqueue(context.Background(), span, 1)
			if len(events) != 2 || len(queue.queue) != 0 {
				t.Fatal("stopped queue accepted data or emitted an enqueue event")
			}
		})
	}
}

func Test_GetBatchSpanProcessor(t *testing.T) {
	ctx := context.Background()
	exporter := &retryTrackingExporter{
		spanAttempts: make(map[string]int),
		fileAttempts: make(map[string]int),
	}
	spanQM := NewBatchSpanProcessor(exporter, nil, nil, nil, nil)
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := spanQM.Shutdown(shutdownCtx); err != nil {
			t.Error(err)
		}
	})

	PatchConvey("Test GetBatchSpanProcessor", t, func() {
		PatchConvey("Test with valid inputs", func() {
			spanQM.OnSpanEnd(ctx, &Span{SpanContext: SpanContext{SpanID: "test-span"}})
			flushCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			err := spanQM.ForceFlush(flushCtx)
			So(err, ShouldBeNil)
			exporter.mu.Lock()
			defer exporter.mu.Unlock()
			So(exporter.spanAttempts["test-span"], ShouldEqual, 2)
			So(len(exporter.fileAttempts), ShouldEqual, 0)
		})
	})
}
