// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/coze-dev/cozeloop-go/spec/tracespec"
)

type raceTestSpanProcessor struct{}

func (raceTestSpanProcessor) OnSpanEnd(context.Context, *Span) {}

func (raceTestSpanProcessor) Shutdown(context.Context) error { return nil }

func (raceTestSpanProcessor) ForceFlush(context.Context) error { return nil }

func TestTransferToUploadSpanAndFileConcurrentMapMutation(t *testing.T) {
	ctx := context.Background()
	span := &Span{
		SpanContext: SpanContext{
			SpanID:  "race-span-id",
			TraceID: "race-trace-id",
			Baggage: make(map[string]string),
		},
		SpanType:            "race-test",
		Name:                "exporter-race",
		WorkspaceID:         "race-workspace",
		StartTime:           time.Now(),
		TagMap:              make(map[string]interface{}),
		SystemTagMap:        make(map[string]interface{}),
		multiModalityKeyMap: make(map[string]struct{}),
		spanProcessor:       raceTestSpanProcessor{},
	}
	span.SetTags(ctx, map[string]interface{}{
		tracespec.Input:  "initial input",
		tracespec.Output: "initial output",
		"race_key":       0,
	})
	for i := 0; i < 46; i++ {
		span.TagMap["padding_"+strconv.Itoa(i)] = i
	}
	span.Finish(ctx)

	const (
		iterations = 1000
		readers    = 4
	)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1 + readers)

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			// Simulate mutations that passed their finished check before Finish
			// and reached the lock after the span became exportable.
			span.lock.Lock()
			span.TagMap[tracespec.Input] = strconv.Itoa(i)
			span.TagMap[tracespec.Output] = strconv.Itoa(i)
			span.TagMap["race_key"] = i
			span.SystemTagMap["race_system_key"] = i
			if i%2 == 0 {
				span.multiModalityKeyMap["race_key"] = struct{}{}
			} else {
				delete(span.multiModalityKeyMap, "race_key")
			}
			span.lock.Unlock()
		}
	}()

	for reader := 0; reader < readers; reader++ {
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < iterations; i++ {
				uploadSpans, _ := transferToUploadSpanAndFile(ctx, []*Span{span})
				if len(uploadSpans) != 1 {
					t.Errorf("expected one upload span, got %d", len(uploadSpans))
					return
				}
			}
		}()
	}

	close(start)
	wg.Wait()
}
