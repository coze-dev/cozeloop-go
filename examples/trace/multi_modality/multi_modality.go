// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/coze-dev/cozeloop-go"
	"github.com/coze-dev/cozeloop-go/internal/logger"
	"github.com/coze-dev/cozeloop-go/internal/util"
	"github.com/coze-dev/cozeloop-go/spec/tracespec"
)

type llmRunner struct {
	client cozeloop.Client
}

const (
	errCodeLLMCall  = 600789111
	errCodeInternal = 600789112
)

func main() {
	// Set the following environment variables first (Assuming you are using a PAT token.).
	// COZELOOP_WORKSPACE_ID=your workspace id
	// COZELOOP_API_TOKEN=your token

	// 0. new client span
	logger.SetLogLevel(logger.LogLevelInfo)
	client, err := cozeloop.NewClient(
		// upload file timeout. If you have enabled ultra-large report or multi-modality report, large text or
		// multi-modality files will be converted into files for upload. You can adjust this parameter, with the
		// default being 30 seconds.
		cozeloop.WithUploadTimeout(30 * time.Second),
	)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()

	llmRunner := llmRunner{
		client: client,
	}

	// 1. start span
	ctx, span := client.StartSpan(ctx, "root_span", "main_span", nil)

	// 2. span set tag or baggage
	// set custom tag
	span.SetTags(ctx, map[string]interface{}{
		"mode":                  "multi_modality",
		"node_id":               6076665,
		"node_process_duration": 228.6,
	})

	// set custom baggage, baggage can cover tag of sample key, and baggage will pass to child span automatically.
	span.SetBaggage(ctx, map[string]string{
		"product_id": "123456654321", // Assuming product_id is global field
	})
	// set baggage key: `user_id`, implicitly set tag key: `user_id`
	span.SetUserIDBaggage(ctx, "123456")

	// assuming call llm
	if err = llmRunner.llmCall(ctx); err != nil {
		span.SetStatusCode(ctx, errCodeLLMCall)
		span.SetError(ctx, err)
	}

	// 3. span finish
	span.Finish(ctx)

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
	ctx, span := r.client.StartSpan(ctx, "llmCall", tracespec.VModelSpanType, nil)
	defer span.Finish(ctx)

	// llm is processing
	//baseURL := "https://xxx"
	//ak := "****"
	modelName := "gpt-4o-2024-05-13"
	maxTokens := 1000 // range: [0, 4096]
	//transport := &MyTransport{
	//	DefaultTransport: &http.Transport{},
	//}
	//config := openai.DefaultAzureConfig(ak, baseURL)
	//config.HTTPClient = &http.Client{
	//	Transport: transport,
	//}
	//client := openai.NewClientWithConfig(config)
	imageBase64Str, err := getMDNBase64("https://www.w3schools.com/w3images/lights.jpg")
	if err != nil {
		logger.CtxErrorf(ctx, "get image failed: %v", err)
		return err
	}
	//resp, err := client.CreateChatCompletion(
	//	ctx,
	//	openai.ChatCompletionRequest{
	//		Model:            modelName,
	//		Messages:         chatCompletionMessage,
	//		MaxTokens:        maxTokens,
	//		TopP:             0.95,
	//		N:                1,
	//		PresencePenalty:  1.0,
	//		FrequencyPenalty: 1.0,
	//		Temperature:      0.6,
	//		Messages: []openai.ChatCompletionMessage{
	//			{
	//				Role: "user",
	//				MultiContent: []openai.ChatMessagePart{
	//					{
	//						Type: openai.ChatMessagePartTypeText,
	//						Text: "这是什么图片？",
	//					},
	//				},
	//			},
	//			{
	//				Role: "user",
	//				MultiContent: []openai.ChatMessagePart{
	//					{
	//						Type: openai.ChatMessagePartTypeImageURL,
	//						ImageURL: &openai.ChatMessageImageURL{
	//							URL: imageBase64Str,
	//						},
	//					},
	//				},
	//			},
	//		},
	//		MaxTokens: maxTokens,
	//	},
	//)

	// mock resp
	time.Sleep(1 * time.Second)
	respChoices := []string{
		"这是一张极光的图片。",
	}
	respPromptTokens := 11
	respCompletionTokens := 52

	// set tag key: `input`
	traceInput, _ := getMultiModalityInput(imageBase64Str)
	span.SetInput(ctx, traceInput)
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
	span.SetTags(ctx, map[string]interface{}{
		tracespec.CallOptions: tracespec.ModelCallOption{
			Temperature:      0.6,
			MaxTokens:        int64(maxTokens),
			TopP:             0.95,
			N:                1,
			PresencePenalty:  util.Ptr(float32(1.0)),
			FrequencyPenalty: util.Ptr(float32(1.0)),
		},
	})

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

func getMultiModalityInput(imageBase64Str string) (*tracespec.ModelInput, error) {
	return &tracespec.ModelInput{ // multi-modality input，must be ModelInput of attribute package
		Messages: []*tracespec.ModelMessage{
			{
				Parts: []*tracespec.ModelMessagePart{
					{
						Type:     "text",
						Text:     "这个图片是什么",
						ImageURL: nil,
						FileURL:  nil,
					},
					{
						Type: tracespec.ModelMessagePartTypeImage,
						Text: "",
						ImageURL: &tracespec.ModelImageURL{
							Name: "test image binary",
							// support MDN Base64 data image / file, or valid URL.
							URL:    imageBase64Str,
							Detail: "",
						},
						FileURL: nil,
					},
				},
			},
		},
		Tools:           nil,
		ModelToolChoice: nil,
	}, nil
}

// MDN: https://developer.mozilla.org/en-US/docs/Web/URI/Reference/Schemes/data#syntax
func getMDNBase64(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch image: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch image: status code %d", resp.StatusCode)
	}

	fileBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read image data: %v", err)
	}

	mimeType := http.DetectContentType(fileBytes)
	base64Data := base64.StdEncoding.EncodeToString(fileBytes)
	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)

	return dataURI, nil
}
