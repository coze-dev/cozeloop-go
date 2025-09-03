// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/coze-dev/cozeloop-go"
	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/util"
)

// The explanation of placeholder variable is based on non-streaming execution, and it also applies to streaming execution.
func main() {
	// 1.Create a prompt on the platform
	// Create a Prompt on the platform's Prompt development page (set Prompt Key to 'ptaas_demo'),
	// add the following messages to the template, submit a version.
	// System: You are a helpful assistant for {{topic}}.
	// Placeholder: {{chat_history}}
	// User: Please help me with {{user_request}}
	ctx := context.Background()

	// Set the following environment variables first.
	// COZELOOP_WORKSPACE_ID=your workspace id
	// COZELOOP_API_TOKEN=your token
	// 2.New loop client
	client, err := cozeloop.NewClient(
		// Set http client time out, default is 3s, max is 10m.
		// Executing a prompt usually takes a considerable amount of time, so adjusting the timeout period is necessary.
		// If you want to adjust the timeout according to the method, you can do so using ctx; please refer to the
		// advanced usage examples.
		cozeloop.WithTimeout(time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer client.Close(ctx)

	// 3. Execute prompt
	executeRequest := &entity.ExecuteParam{
		PromptKey: "ptaas_demo",
		Version:   "0.0.5",
		VariableVals: map[string]any{
			"topic": "artificial intelligence",
			// chat_history is a placeholder variable, and it can be []*entity.Message(recommend)/[]entity.Message/*entity.Message/entity.Message.
			"chat_history": []*entity.Message{
				{
					Role:    entity.RoleUser,
					Content: util.Ptr("hello"),
				},
				{
					Role:    entity.RoleAssistant,
					Content: util.Ptr("hello"),
				},
			},
			"user_request": "explain what is machine learning",
		},
	}
	nonStream(ctx, client, executeRequest)
}

func nonStream(ctx context.Context, client cozeloop.Client, executeRequest *entity.ExecuteParam) {
	result, err := client.Execute(ctx, executeRequest)
	if err != nil {
		panic(err)
	}
	printExecuteResult(result)
}

func printExecuteResult(result entity.ExecuteResult) {
	if result.Message != nil {
		fmt.Printf("Message: %s\n", util.ToJSON(result.Message))
	}
	if util.PtrValue(result.FinishReason) != "" {
		fmt.Printf("FinishReason: %s\n", util.PtrValue(result.FinishReason))
	}
	if result.Usage != nil {
		fmt.Printf("Usage: %s\n", util.ToJSON(result.Usage))
	}
}
