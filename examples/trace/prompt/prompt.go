// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package main

import (
	"context"

	"github.com/coze-dev/cozeloop-go"
	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/consts"
	"github.com/coze-dev/cozeloop-go/internal/logger"
	"github.com/coze-dev/cozeloop-go/internal/util"
	"github.com/coze-dev/cozeloop-go/spec/tracespec"
)

type getPromptRunner struct {
	client cozeloop.Client
}

const (
	errCodeInternal = 600789111
)

func main() {
	// Set the following environment variables first (Assuming you are using a PAT token.).
	// COZELOOP_WORKSPACE_ID=your workspace id
	// COZELOOP_API_TOKEN=your token

	// 0. new client span
	logger.SetLogLevel(logger.LogLevelInfo)
	client, err := cozeloop.NewClient()
	if err != nil {
		panic(err)
	}
	ctx := context.Background()

	getPromptRunner := getPromptRunner{
		client: client,
	}

	// 1. start span
	ctx, span := client.StartSpan(ctx, "root_span", "main_span", nil)

	// 2. span set tag or baggage
	// set custom tag
	span.SetTags(ctx, map[string]interface{}{
		"mode":                  "simple",
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
	prompt, err := getPromptRunner.getPrompt(ctx)
	if err != nil {
		// set tag key: `_status_code`
		span.SetStatusCode(ctx, errCodeInternal)
		// set tag key: `error`, if `_status_code` value is not defined, `_status_code` value will be set -1.
		span.SetError(ctx, err)
	}
	if _, err = getPromptRunner.formatPrompt(ctx, prompt, map[string]any{"var1": "你会什么技能"}); err != nil {
		// set tag key: `_status_code`
		span.SetStatusCode(ctx, errCodeInternal)
		// set tag key: `error`, if `_status_code` value is not defined, `_status_code` value will be set -1.
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

func (r *getPromptRunner) getPrompt(ctx context.Context) (prompt *entity.Prompt, err error) {
	ctx, span := r.client.StartSpan(ctx, "get_prompt", tracespec.VPromptHubSpanType, nil)
	defer span.Finish(ctx)

	ctx, promptHubSpan := r.client.StartSpan(ctx, consts.TracePromptHubSpanName, tracespec.VPromptHubSpanType)
	defer func() {
		if promptHubSpan != nil {
			promptHubSpan.SetTags(ctx, map[string]any{
				tracespec.PromptKey: "test_demo",
				tracespec.Input: util.ToJSON(map[string]any{
					tracespec.PromptKey:     "test_demo",
					tracespec.PromptVersion: "v1.0.1",
				}),
			})
			promptHubSpan.SetTags(ctx, map[string]any{
				tracespec.PromptVersion: "v1.0.1", // mock version
				tracespec.Output:        prompt,
			})
			if err != nil {
				promptHubSpan.SetError(ctx, err)
			}
			promptHubSpan.Finish(ctx)
		}
	}()

	return getPrompt()
}

func getPrompt() (*entity.Prompt, error) {
	return &entity.Prompt{
		PromptTemplate: &entity.PromptTemplate{
			Messages: []*entity.Message{
				{
					Role:    entity.RoleSystem,
					Content: util.Ptr("Hello!"),
				},
				{
					Role:    entity.RoleUser,
					Content: util.Ptr("Hello! {{var1}}"),
				},
			},
		},
	}, nil
}

func (r *getPromptRunner) formatPrompt(ctx context.Context, prompt *entity.Prompt, variables map[string]any) (messages []*entity.Message, err error) {
	ctx, span := r.client.StartSpan(ctx, "format_prompt", tracespec.VPromptTemplateSpanType, nil)
	defer span.Finish(ctx)
	ctx, promptTemplateSpan := r.client.StartSpan(ctx, consts.TracePromptTemplateSpanName, tracespec.VPromptTemplateSpanType)
	defer func() {
		if promptTemplateSpan != nil {
			promptTemplateSpan.SetTags(ctx, map[string]any{
				tracespec.PromptKey:     "test_demo",
				tracespec.PromptVersion: "v1.0.1",
				tracespec.Input:         util.ToJSON(toSpanPromptInput(prompt.PromptTemplate.Messages, variables)),
				tracespec.Output:        util.ToJSON(toSpanMessages(messages)),
			})
			if err != nil {
				promptTemplateSpan.SetStatusCode(ctx, util.GetErrorCode(err))
				promptTemplateSpan.SetError(ctx, err)
			}
			promptTemplateSpan.Finish(ctx)
		}
	}()

	return doPromptFormat()
}

func doPromptFormat() ([]*entity.Message, error) { // mock data
	return []*entity.Message{}, nil
}

func toSpanPromptInput(messages []*entity.Message, arguments map[string]any) *tracespec.PromptInput {
	return &tracespec.PromptInput{
		Templates: toSpanMessages(messages),
		Arguments: toSpanArguments(arguments),
	}
}

func toSpanArguments(arguments map[string]any) []*tracespec.PromptArgument {
	var result []*tracespec.PromptArgument
	for key, value := range arguments {
		result = append(result, &tracespec.PromptArgument{
			Key:   key,
			Value: value,
		})
	}
	return result
}

func toSpanMessages(messages []*entity.Message) []*tracespec.ModelMessage {
	var result []*tracespec.ModelMessage
	for _, msg := range messages {
		result = append(result, toSpanMessage(msg))
	}
	return result
}

func toSpanMessage(message *entity.Message) *tracespec.ModelMessage {
	if message == nil {
		return nil
	}
	return &tracespec.ModelMessage{
		Role:    string(message.Role),
		Content: util.PtrValue(message.Content),
	}
}
