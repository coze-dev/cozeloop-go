// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"

	"github.com/coze-dev/cozeloop-go"
	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/util"
)

// The explanation of jinja2 template is based on non-streaming execution, and it also applies to streaming execution.
func main() {
	// 1.Create a prompt using jinja2 template on the platform
	// Create a Prompt on the platform's Prompt development page (set Prompt Key to 'ptaas_demo'),
	// add the following messages to the template, submit a version, and set a label (e.g., 'production') for that version.
	// System: You are a helpful assistant for {{param.topic}}. Your audience is {{param.age}} years old.
	// User: Please help me with {{param.user_request}}
	ctx := context.Background()

	// Set the following environment variables first.
	// COZELOOP_WORKSPACE_ID=your workspace id
	// COZELOOP_API_TOKEN=your token
	// 2.New loop client
	client, err := cozeloop.NewClient()
	if err != nil {
		panic(err)
	}
	defer client.Close(ctx)

	// 3. Execute prompt
	executeRequest := &entity.ExecuteParam{
		PromptKey: "ptaas_demo",
		Version:   "0.0.2",
		VariableVals: map[string]any{
			"param": struct {
				Topic       string `json:"topic"`
				Age         int    `json:"age"`
				UserRequest string `json:"user_request"`
			}{
				Topic:       "artificial intelligence",
				Age:         10,
				UserRequest: "explain what is machine learning",
			},
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
