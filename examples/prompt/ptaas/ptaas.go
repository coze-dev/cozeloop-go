// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"io"

	"github.com/coze-dev/cozeloop-go"
	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/util"
)

func main() {
	// 1.Create a prompt on the platform
	// Create a Prompt on the platform's Prompt development page (set Prompt Key to 'ptaas_demo'),
	// add the following messages to the template, submit a version.
	// System: You are a helpful assistant for {{topic}}.
	// User: Please help me with {{user_request}}
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
		Version:   "0.0.1",
		VariableVals: map[string]any{
			"topic":        "artificial intelligence",
			"user_request": "explain what is machine learning",
		},
		// You can also append messages to the prompt.
		Messages: []*entity.Message{
			{
				Role:    entity.RoleUser,
				Content: util.Ptr("Keep the answer brief."),
			},
		},
	}
	// 3.1 non stream
	nonStream(ctx, client, executeRequest)
	// 3.2 stream
	stream(ctx, client, executeRequest)
}

func nonStream(ctx context.Context, client cozeloop.Client, executeRequest *entity.ExecuteParam) {
	result, err := client.Execute(ctx, executeRequest)
	if err != nil {
		panic(err)
	}
	printExecuteResult(result)
}

func stream(ctx context.Context, client cozeloop.Client, executeRequest *entity.ExecuteParam) {
	streamReader, err := client.ExecuteStreaming(ctx, executeRequest)
	if err != nil {
		panic(err)
	}
	for {
		result, err := streamReader.Recv()
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nStream finished.")
				break
			}
			panic(err)
		}
		printExecuteResult(result)
	}
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
