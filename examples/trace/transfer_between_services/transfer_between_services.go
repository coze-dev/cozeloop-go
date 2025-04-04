// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"net/http"
	"time"

	"github.com/coze-dev/cozeloop-go"
	"github.com/coze-dev/cozeloop-go/attribute/trace"
	"github.com/coze-dev/cozeloop-go/internal/logger"
)

type llmRunner struct {
	client cozeloop.Client
}

const (
	errCodeLLMCall  = 600789111
	errCodeInternal = 600789112
)

// this is main of service A
func main() {
	// Set the following environment variables first (Assuming you are using a PAT token.).
	// COZELOOP_WORKSPACE_ID=your workspace id
	// COZELOOP_API_TOKEN=your token

	// 0. new client rootSpan
	logger.SetLogLevel(logger.LogLevelInfo)
	client, err := cozeloop.NewClient()
	if err != nil {
		panic(err)
	}
	ctx := context.Background()

	llmRunner := llmRunner{
		client: client,
	}

	// 1. start rootSpan, because there is no rootSpan in the ctx, so parentSpan is root rootSpan of new trace
	ctx, rootSpan := client.StartSpan(ctx, "root_span_serviceA", "main_span", nil)

	// 2. rootSpan set tag or baggage
	// set custom tag
	rootSpan.SetTags(ctx, map[string]interface{}{
		"service_name": "serviceA",
	})

	// set custom baggage, baggage can cover tag of sample key, and baggage will pass to child rootSpan automatically.
	rootSpan.SetBaggage(ctx, map[string]string{
		"product_id":      "123456654321", // Assuming product_id is global field, need to be passed to child rootSpan automatically.
		"product_name":    "AI bot",       // Assuming product_name is global field, need to be passed to child rootSpan automatically.
		"product_version": "0.0.1",        // Assuming product_version is global field, need to be passed to child rootSpan automatically.
	})
	// set baggage key: `user_id`, implicitly set tag key: `user_id`
	rootSpan.SetUserIDBaggage(ctx, "123456")

	// assuming call llm
	if err = llmRunner.llmCall(ctx); err != nil {
		// set tag key: `_status_code`
		rootSpan.SetStatusCode(ctx, errCodeLLMCall)
		// set tag key: `error`, if `_status_code` value is not defined, `_status_code` value will be set -1.
		rootSpan.SetError(ctx, err.Error())
	}

	header, err := rootSpan.ToHeader()
	if err != nil {
		// set tag key: `_status_code`
		rootSpan.SetStatusCode(ctx, errCodeInternal)
		// set tag key: `error`, if `_status_code` value is not defined, `_status_code` value will be set -1.
		rootSpan.SetError(ctx, err.Error())
	}

	// 3. Assuming invoke another service, need to pass span header to another service for linking trace
	llmRunner.invokeServiceB(header)

	// 3. rootSpan finish
	rootSpan.Finish(ctx)

	// 4. (optional) flush or close
	// -- force flush, report all traces in the queue
	// Warning! In general, this method is not needed to be call, as spans will be automatically reported in batches.
	// Note that flush will block and wait for the report to complete, and it may cause frequent reporting,
	// affecting performance.
	client.Flush(ctx)

	// -- close trace, do flush and close client
	// Warning! Once Close is executed, the client will become unavailable and a new client needs
	// to be created via NewClient! Use it only when you need to release resources, such as shutting down an instance!
	//client.Close(ctx)
}

func (r *llmRunner) llmCall(ctx context.Context) (err error) {
	ctx, span := r.client.StartSpan(ctx, "llmCall", trace.VModelSpanType, nil)
	defer span.Finish(ctx)

	// llm is processing
	//baseURL := "https://xxx"
	//ak := "****"
	modelName := "gpt-4o-2024-05-13"
	//maxTokens := 1000 // range: [0, 4096]
	//transport := &MyTransport{
	//	DefaultTransport: &http.Transport{},
	//}
	//config := openai.DefaultAzureConfig(ak, baseURL)
	//config.HTTPClient = &http.Client{
	//	Transport: transport,
	//}
	//client := openai.NewClientWithConfig(config)
	input := "上海天气怎么样？"
	//resp, err := client.CreateChatCompletion(
	//	ctx,
	//	openai.ChatCompletionRequest{
	//		Model: modelName,
	//		Messages: []openai.ChatCompletionMessage{
	//			{
	//				Role:    "user",
	//				Content: input,
	//			},
	//		},
	//		MaxTokens: maxTokens,
	//	},
	//)

	// mock resp
	time.Sleep(1 * time.Second)
	respChoices := []string{
		"上海天气晴朗，气温25摄氏度。",
	}
	respPromptTokens := 11
	respCompletionTokens := 52

	// set tag key: `input`
	span.SetInput(ctx, input)
	// set tag key: `output`
	span.SetOutput(ctx, respChoices)
	// set tag key: `model_provider`, e.g., openai, etc.
	span.SetModelProvider(ctx, "openai")
	// set tag key: `start_time_first_resp`
	// Timestamp of the first packet return from LLM, unit: microseconds.
	// When `start_time_first_resp` is set, a tag named `latency_first_resp` calculated
	// based on the span's StartTime will be added, meaning the latency for the first packet.
	span.SetStartTimeFirstResp(ctx, time.Now().UnixMicro())
	// set tag key: `input_tokens`. The amount of input tokens.
	// when the `input_tokens` value is set, it will automatically sum with the `output_tokens` to calculate the `tokens` tag.
	span.SetInputTokens(ctx, respPromptTokens)
	// set tag key: `output_tokens`. The amount of output tokens.
	// when the `output_tokens` value is set, it will automatically sum with the `input_tokens` to calculate the `tokens` tag.
	span.SetOutputTokens(ctx, respCompletionTokens)
	// set tag key: `model_name`, e.g., gpt-4-1106-preview, etc.
	span.SetModelName(ctx, modelName)

	return nil
}

type MyTransport struct {
	Header           http.Header
	DefaultTransport http.RoundTripper
}

func (transport *MyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for key, values := range transport.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	return transport.DefaultTransport.RoundTrip(req)
}

// Assuming anotherService is another service.
// In reality, the client should be recreated via NewClient, but for convenience, the same client is used here.
func (r *llmRunner) invokeServiceB(reqHeader map[string]string) {
	ctx := context.Background()

	// 1. start rootSpan of service B,
	spanContext := cozeloop.GetSpanFromHeader(ctx, reqHeader)
	ctx, rootSpan := r.client.StartSpan(ctx, "root_span_serviceB", "main_span", []cozeloop.StartSpanOption{
		cozeloop.WithChildOf(spanContext),
	}...)
	defer rootSpan.Finish(ctx)

	// 2. rootSpan set tag or baggage
	// set custom tag
	rootSpan.SetTags(ctx, map[string]interface{}{
		"service_name": "serviceB",
	})
	// do something...
}
