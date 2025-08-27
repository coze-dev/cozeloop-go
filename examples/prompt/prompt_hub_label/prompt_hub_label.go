// start_aigc
// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/coze-dev/cozeloop-go"
	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/util"
	"github.com/coze-dev/cozeloop-go/spec/tracespec"
)

// Demo: How to get prompt by prompt_key and label
func main() {
	// 1.Create a prompt on the platform
	// Create a Prompt on the platform's Prompt development page (set Prompt Key to 'prompt_hub_label_demo'),
	// add the following messages to the template, submit a version, and set a label (e.g., 'production') for that version.
	// System: You are a helpful assistant for {{topic}}.
	// User: Please help me with {{user_request}}

	ctx := context.Background()

	// Set the following environment variables first.
	// COZELOOP_WORKSPACE_ID=your workspace id
	// COZELOOP_API_TOKEN=your token
	// 2.New loop client
	os.Setenv("x_tt_env", "boe_prompt_label")
	client, err := cozeloop.NewClient(
		// Set whether to report a trace span when get or format prompt.
		// Default value is false.
		cozeloop.WithWorkspaceID("7344808233917693996"),
		cozeloop.WithAPIBaseURL("https://api-bot-boe.bytedance.net"),
		cozeloop.WithAPIToken("pat_iweTAuo0jJoYDOhfYyuI4Iixqc23dOmLtTTbEqksad4ZUC0eLbzHoN0qQaDJgAVv"),
		cozeloop.WithPromptTrace(true))
	if err != nil {
		panic(err)
	}

	llmRunner := llmRunner{
		client: client,
	}

	// 1. start root span
	ctx, span := llmRunner.client.StartSpan(ctx, "root_span", "main_span", nil)

	// 3.Get prompt by prompt_key and label
	labelValue := "production" // Can be production, beta, test or custom labels
	prompt, err := llmRunner.client.GetPrompt(ctx, cozeloop.GetPromptParam{
		PromptKey: "prompt_hub_label_demo",
		Label:     labelValue, // Get prompt version by label
		// Note: When Version is specified, Label field will be ignored
	})
	if err != nil {
		fmt.Printf("get prompt failed: %v\n", err)
		return
	}

	if prompt != nil {
		fmt.Printf("successfully got prompt with label '%s'\n", labelValue)

		// Get messages of the prompt
		if prompt.PromptTemplate != nil {
			messages, err := json.Marshal(prompt.PromptTemplate.Messages)
			if err != nil {
				fmt.Printf("json marshal failed: %v\n", err)
				return
			}
			fmt.Printf("prompt messages=%s\n", string(messages))
		}

		// Get llm config of the prompt
		if prompt.LLMConfig != nil {
			llmConfig, err := json.Marshal(prompt.LLMConfig)
			if err != nil {
				fmt.Printf("json marshal failed: %v\n", err)
			}
			fmt.Printf("prompt llm config=%s\n", llmConfig)
		}

		// 4.Format messages of the prompt
		messages, err := llmRunner.client.PromptFormat(ctx, prompt, map[string]any{
			// Normal variable type should be string
			"topic":        "artificial intelligence",
			"user_request": "explain what is machine learning",
		})
		if err != nil {
			fmt.Printf("prompt format failed: %v\n", err)
			return
		}

		data, err := json.Marshal(messages)
		if err != nil {
			fmt.Printf("json marshal failed: %v\n", err)
			return
		}
		fmt.Printf("formatted messages=%s\n", string(data))

		// 5. llm call
		err = llmRunner.llmCall(ctx, messages)
		if err != nil {
			fmt.Printf("llm call failed: %v\n", err)
			return
		}
	}

	// 6. span finish
	span.Finish(ctx)

	// 7. (optional) flush or close
	// -- force flush, report all traces in the queue
	// Warning! In general, this method is not needed to be call, as spans will be automatically reported in batches.
	// Note that flush will block and wait for the report to complete, and it may cause frequent reporting,
	// affecting performance.
	llmRunner.client.Flush(ctx)
}

type llmRunner struct {
	client cozeloop.Client
}

func (r *llmRunner) llmCall(ctx context.Context, messages []*entity.Message) (err error) {
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
	//marshal, err := json.Marshal(messages)
	//if err != nil {
	//	return err
	//}
	//chatCompletionMessage := make([]openai.ChatCompletionMessage, 0)
	//err = json.Unmarshal(marshal, &chatCompletionMessage)
	//if err != nil {
	//	return err
	//}
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
	//	},
	//)

	// mock resp
	time.Sleep(1 * time.Second)
	respChoices := []string{
		"Hello! Can I help you?",
	}
	respPromptTokens := 11
	respCompletionTokens := 52

	// set tag key: `input`
	span.SetInput(ctx, convertModelInput(messages))
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

func convertModelInput(messages []*entity.Message) *tracespec.ModelInput {
	modelMessages := make([]*tracespec.ModelMessage, 0)
	for _, message := range messages {
		modelMessages = append(modelMessages, &tracespec.ModelMessage{
			Role:    string(message.Role),
			Content: util.PtrValue(message.Content),
		})
	}

	return &tracespec.ModelInput{
		Messages: modelMessages,
	}
}

// end_aigc
