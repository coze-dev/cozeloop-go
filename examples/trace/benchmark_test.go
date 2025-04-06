// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/coze-dev/cozeloop-go"
	"github.com/coze-dev/cozeloop-go/internal/logger"
)

type llmRunner struct {
	client cozeloop.Client
}

func BenchmarkMyFunctionWithQPS(b *testing.B) {
	logger.SetLogLevel(logger.LogLevelDebug)
	client, err := cozeloop.NewClient()
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	runner := llmRunner{
		client: client,
	}

	qps := 500 // set QPS
	interval := time.Second / time.Duration(qps)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	done := make(chan bool)

	go monitorResources()

	// start goroutine invoke MyFunction
	go func() {
		for {
			select {
			case <-ticker.C:
				go func() {
					//logger.CtxInfof(ctx, "run span demo ######################################################################################")
					runner.llmRunner(ctx, "test input")
				}()
			case <-done:
				logger.CtxInfof(ctx, "done span demo ######################################################################################")
				return
			}
		}
	}()

	// run benchmark test
	b.ResetTimer()
	time.Sleep(1000 * time.Second) // wait
	close(done)
}

func monitorResources() {
	for {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		fmt.Printf("Memory Usage: %d KB\n", m.Alloc/1024)
		time.Sleep(1 * time.Second)
	}
}

func (r *llmRunner) llmRunner(ctx context.Context, input interface{}) (err error) {
	ctx, span := r.client.StartSpan(ctx, "llmCall", tracespec.tracespce.VModelSpanType, nil)
	defer span.Finish(ctx)

	// Assuming llm is processing
	time.Sleep(1 * time.Second)
	output := "我是机器人，没有具体的名字，你可以给我起个名字。"

	// Assuming llm return input_token and output_token
	inputToken := 232
	outputToken := 1211

	// set tag key: `input`
	span.SetInput(ctx, input)
	// set tag key: `output`
	span.SetOutput(ctx, output)
	// set tag key: `model_provider`, e.g., openai, etc.
	span.SetModelProvider(ctx, "openai")
	// set tag key: `start_time_first_resp`
	// Timestamp of the first packet return from LLM, unit: microseconds.
	// When `start_time_first_resp` is set, a tag named `latency_first_resp` calculated
	// based on the span's StartTime will be added, meaning the latency for the first packet.
	span.SetStartTimeFirstResp(ctx, time.Now().UnixMicro())
	// set tag key: `input_tokens`. The amount of input tokens.
	// when the `input_tokens` value is set, it will automatically sum with the `output_tokens` to calculate the `tokens` tag.
	span.SetInputTokens(ctx, inputToken)
	// set tag key: `output_tokens`. The amount of output tokens.
	// when the `output_tokens` value is set, it will automatically sum with the `input_tokens` to calculate the `tokens` tag.
	span.SetOutputTokens(ctx, outputToken)
	// set tag key: `model_name`, e.g., gpt-4-1106-preview, etc.
	span.SetModelName(ctx, "gpt-4-1106-preview")

	return nil
}
