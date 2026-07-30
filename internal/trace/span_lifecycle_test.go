// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

import (
	"context"
	"errors"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/spec/tracespec"
)

type lifecycleTestSpanProcessor struct {
	ended int32
}

func (p *lifecycleTestSpanProcessor) OnSpanEnd(ctx context.Context, span *Span) {
	transferToUploadSpanAndFile(ctx, []*Span{span})
	span.getBytesSize()
	atomic.AddInt32(&p.ended, 1)
}

func (*lifecycleTestSpanProcessor) Shutdown(context.Context) error { return nil }

func (*lifecycleTestSpanProcessor) ForceFlush(context.Context) error { return nil }

type spanMutableState struct {
	maps        spanMapSnapshot
	baggage     map[string]string
	duration    int64
	bytesSize   int64
	statusCode  int32
	serviceName string
	logID       string
	finishTime  time.Time
}

func newLifecycleTestSpan(processor SpanProcessor) *Span {
	return &Span{
		SpanContext: SpanContext{
			SpanID:  "lifecycle-span-id",
			TraceID: "lifecycle-trace-id",
			Baggage: make(map[string]string),
		},
		SpanType:            "lifecycle-test",
		Name:                "lifecycle-test",
		WorkspaceID:         "lifecycle-workspace",
		StartTime:           time.Now().Add(-time.Second),
		TagMap:              make(map[string]interface{}),
		SystemTagMap:        make(map[string]interface{}),
		multiModalityKeyMap: make(map[string]struct{}),
		spanProcessor:       processor,
	}
}

func getSpanMutableState(span *Span) spanMutableState {
	return spanMutableState{
		maps:        span.getExportSnapshot(),
		baggage:     span.GetBaggage(),
		duration:    span.GetDuration(),
		bytesSize:   span.getBytesSize(),
		statusCode:  span.GetStatusCode(),
		serviceName: span.GetServiceName(),
		logID:       span.GetLogID(),
		finishTime:  span.GetFinishTime(),
	}
}

func Test_SpanSettersDoNotMutateAfterFinish(t *testing.T) {
	ctx := context.Background()
	processor := &lifecycleTestSpanProcessor{}
	span := newLifecycleTestSpan(processor)
	finishTime := time.Now()

	span.SetInput(ctx, "initial input")
	span.SetOutput(ctx, "initial output")
	span.SetTags(ctx, map[string]interface{}{"initial_tag": "value"})
	span.SetBaggage(ctx, map[string]string{"initial_baggage": "value"})
	span.SetMultiModalityMap("initial_multi_modality")
	span.SetStatusCode(ctx, 1)
	span.SetRuntime(ctx, tracespec.Runtime{Scene: tracespec.VSceneCustom})
	span.SetServiceName(ctx, "initial service")
	span.SetLogID(ctx, "initial log")
	span.SetFinishTime(finishTime)
	span.SetSystemTags(ctx, map[string]interface{}{"initial_system_tag": "value"})
	span.Finish(ctx)

	before := getSpanMutableState(span)
	span.SetInput(ctx, "late input")
	span.SetOutput(ctx, "late output")
	span.SetTags(ctx, map[string]interface{}{"late_tag": "value"})
	span.SetBaggage(ctx, map[string]string{"late_baggage": "value"})
	span.SetBaggageItem("late_baggage_item", "value")
	span.SetMultiModalityMap("late_multi_modality")
	span.SetError(ctx, errors.New("late error"))
	span.SetStatusCode(ctx, 2)
	span.SetUserID(ctx, "late user")
	span.SetUserIDBaggage(ctx, "late user")
	span.SetMessageID(ctx, "late message")
	span.SetMessageIDBaggage(ctx, "late message")
	span.SetThreadID(ctx, "late thread")
	span.SetThreadIDBaggage(ctx, "late thread")
	span.SetPrompt(ctx, entity.Prompt{PromptKey: "late prompt", Version: "1"})
	span.SetModelProvider(ctx, "late provider")
	span.SetModelName(ctx, "late model")
	span.SetModelCallOptions(ctx, map[string]interface{}{"late": true})
	span.SetInputTokens(ctx, 1)
	span.SetOutputTokens(ctx, 2)
	span.SetStartTimeFirstResp(ctx, time.Now().UnixMicro())
	span.SetRuntime(ctx, tracespec.Runtime{Scene: "late scene"})
	span.SetServiceName(ctx, "late service")
	span.SetLogID(ctx, "late log")
	span.SetFinishTime(finishTime.Add(time.Hour))
	span.SetSystemTags(ctx, map[string]interface{}{"late_system_tag": "value"})
	span.SetDeploymentEnv(ctx, "late deployment")
	after := getSpanMutableState(span)

	if !reflect.DeepEqual(before, after) {
		t.Fatalf("finished span mutated:\nbefore: %#v\nafter:  %#v", before, after)
	}
	if got := atomic.LoadInt32(&processor.ended); got != 1 {
		t.Fatalf("expected one finished span, got %d", got)
	}
}

func Test_SpanFinishConcurrentSetters(t *testing.T) {
	ctx := context.Background()
	const (
		rounds    = 200
		finishers = 4
	)

	for round := 0; round < rounds; round++ {
		processor := &lifecycleTestSpanProcessor{}
		span := newLifecycleTestSpan(processor)
		start := make(chan struct{})
		setters := []func(){
			func() { span.SetInput(ctx, "input "+strconv.Itoa(round)) },
			func() { span.SetOutput(ctx, "output "+strconv.Itoa(round)) },
			func() { span.SetTags(ctx, map[string]interface{}{"tag": round}) },
			func() { span.SetBaggage(ctx, map[string]string{"baggage": strconv.Itoa(round)}) },
			func() { span.SetBaggageItem("baggage_item", strconv.Itoa(round)) },
			func() { span.SetMultiModalityMap("multi_modality") },
			func() { span.SetStatusCode(ctx, round) },
			func() { span.SetSystemTags(ctx, map[string]interface{}{"system_tag": round}) },
			func() { span.SetServiceName(ctx, "service "+strconv.Itoa(round)) },
			func() { span.SetLogID(ctx, "log "+strconv.Itoa(round)) },
			func() { span.SetFinishTime(time.Now()) },
			func() { _, _ = span.ToHeader() },
		}

		var wg sync.WaitGroup
		wg.Add(len(setters) + finishers)
		for _, setter := range setters {
			setter := setter
			go func() {
				defer wg.Done()
				<-start
				for i := 0; i < 5; i++ {
					setter()
					runtime.Gosched()
				}
			}()
		}
		for finisher := 0; finisher < finishers; finisher++ {
			go func() {
				defer wg.Done()
				<-start
				runtime.Gosched()
				span.Finish(ctx)
			}()
		}

		close(start)
		wg.Wait()

		if !span.isSpanFinished() {
			t.Fatalf("span was not finished in round %d", round)
		}
		if got := atomic.LoadInt32(&processor.ended); got != 1 {
			t.Fatalf("expected one finished span in round %d, got %d", round, got)
		}
	}
}
