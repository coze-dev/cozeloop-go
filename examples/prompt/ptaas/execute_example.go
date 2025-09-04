// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/coze-dev/coze-loop/backend/pkg/json"

	"github.com/coze-dev/cozeloop-go"
	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/util"
)

func main() {
	os.Setenv("x_tt_env", "boe_ptaas")
	// 创建客户端
	client, err := cozeloop.NewClient(
		cozeloop.WithAPIBaseURL("https://api-bot-boe.bytedance.net"),
		cozeloop.WithAPIToken("pat_tqclS1ltbJnuQCLlyydiyOg6kh859q4WBN5MTFVr8hGNkBUD2HY4JrNKujGx6XHH"),
		cozeloop.WithWorkspaceID("7344808233917693996"),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close(context.Background())

	ctx := context.Background()
	ctx, _ = context.WithTimeout(ctx, time.Minute)

	// Execute 示例
	fmt.Println("=== Execute Example ===")
	executeRequest := &entity.ExecuteParam{
		PromptKey: "prompt_hub_label_demo",
		Version:   "", // 可选，不设置则使用最新版本
		Label:     "",
		VariableVals: map[string]interface{}{
			"query": "上海天气如何",
		},
	}

	//response, err := client.Execute(ctx, executeRequest)
	//if err != nil {
	//	log.Fatalf("Execute failed: %v", err)
	//}
	//
	//fmt.Printf("Response: %+v\n", response)
	//if response.Message != nil && response.Message.Content != nil {
	//	fmt.Printf("Message Content: %s\n", *response.Message.Content)
	//}
	//if response.Usage != nil {
	//	fmt.Printf("Token Usage: %+v\n", response.Usage)
	//}

	// ExecuteStreaming 示例
	fmt.Println("\n=== ExecuteStreaming Example ===")
	streamReader, err := client.ExecuteStreaming(ctx, executeRequest)
	if err != nil {
		log.Fatalf("ExecuteStreaming failed: %v", err)
	}

	fmt.Println("Streaming response:")
	for {
		chunk, err := streamReader.Recv()
		if err == io.EOF {
			fmt.Println("\nStream finished.")
			break
		}
		if err != nil {
			log.Fatalf("Stream recv failed: %v", err)
		}

		// 处理流式响应
		if chunk.Message != nil {
			if reasoningContent := util.PtrValue(chunk.Message.ReasoningContent); reasoningContent != "" {
				fmt.Printf("ReasoningContent: %s\n", reasoningContent)
			}
			if content := util.PtrValue(chunk.Message.Content); content != "" {
				fmt.Printf("Content: %s\n", *chunk.Message.Content)
			}
			if len(chunk.Message.ToolCalls) > 0 {
				data, _ := json.Marshal(chunk.Message.ToolCalls)
				fmt.Printf("ToolCalls: %s\n", string(data))
			}
		}

		// 检查是否完成
		if finishReason := util.PtrValue(chunk.FinishReason); finishReason != "" {
			fmt.Printf("\nFinish reason: %s\n", finishReason)
		}
		if chunk.Usage != nil {
			fmt.Printf("Token Usage: %+v\n", chunk.Usage)
		}
	}

	// 也可以使用 AggregateMessage 获取完整响应
	//fmt.Println("\n=== AggregateMessage Example ===")
	//streamReader2, err := client.ExecuteStreaming(ctx, executeRequest)
	//if err != nil {
	//	log.Fatalf("ExecuteStreaming failed: %v", err)
	//}
	//defer streamReader2.Close()
	//
	//aggregatedResponse, err := streamReader2.AggregateMessage(ctx)
	//if err != nil {
	//	log.Fatalf("AggregateMessage failed: %v", err)
	//}
	//
	//fmt.Printf("Aggregated Response: %+v\n", aggregatedResponse)
	//if aggregatedResponse.Message != nil && aggregatedResponse.Message.Content != nil {
	//	fmt.Printf("Full Message: %s\n", *aggregatedResponse.Message.Content)
	//}
}

// 高级用法示例
//func advancedExample() {
//	client, err := cozeloop.NewClient(
//		cozeloop.WithAPIToken("your-api-token"),
//		cozeloop.WithWorkspaceID("your-workspace-id"),
//	)
//	if err != nil {
//		log.Fatalf("Failed to create client: %v", err)
//	}
//	defer client.Close(context.Background())
//
//	ctx := context.Background()
//
//	// 使用消息格式的请求
//	executeRequest := &entity.ExecuteParam{
//		PromptKey: "your-prompt-key",
//		Messages: []*entity.Message{
//			{
//				Role:    entity.RoleUser,
//				Content: stringPtr("Hello, I need help with something."),
//			},
//			{
//				Role:    entity.RoleAssistant,
//				Content: stringPtr("Of course! I'd be happy to help. What do you need assistance with?"),
//			},
//			{
//				Role:    entity.RoleUser,
//				Content: stringPtr("Can you explain quantum computing?"),
//			},
//		},
//	}
//
//	// 使用选项
//	response, err := client.Execute(ctx, executeRequest)
//	if err != nil {
//		log.Fatalf("Execute failed: %v", err)
//	}
//
//	fmt.Printf("Advanced Response: %+v\n", response)
//}
//
//func stringPtr(s string) *string {
//	return &s
//}
