// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/coze-dev/cozeloop-go"
	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/util"
)

// The explanation of multi modal is based on non-streaming execution, and it also applies to streaming execution.
func main() {
	// 1.Create a prompt on the platform
	// Create a Prompt on the platform's Prompt development page (set Prompt Key to 'ptaas_demo'),
	// add the following messages to the template, submit a version. example1 and example2 are the multi modal variables.
	// System: You can quickly identify the location where a photo was taken.
	// User: 例如：{{example1}}
	// Assistant: {{city1}}
	// User: 例如：{{example2}}
	// Assistant: {{city2}}
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
	imagePath := "/Users/bytedance/Downloads/shanghai.jpeg"
	imageBytes, err := os.ReadFile(imagePath)
	if err != nil {
		panic(err)
	}
	base64Image := base64.StdEncoding.EncodeToString(imageBytes)
	base64data := fmt.Sprintf("data:image/jpeg;base64,%s", base64Image)
	executeRequest := &entity.ExecuteParam{
		PromptKey: "ptaas_demo",
		Version:   "0.0.8",
		// multi modal variable can be []*entity.ContentPart(recommend)/[]entity.ContentPart/*entity.ContentPart/entity.ContentPart
		// Images can be provided via URL or in base64 encoded format.
		// Image URL needs to be publicly accessible.
		// Base64-formatted data should follow the standard data URI format, like "data:[<mediatype>][;base64],<data>".
		VariableVals: map[string]any{
			"example1": []*entity.ContentPart{
				{
					Type:     entity.ContentTypeImageURL,
					ImageURL: util.Ptr("https://p8.itc.cn/q_70/images03/20221219/61785c89cd17421ca0d007c7a87d09fb.jpeg"),
				},
			},
			"city1": "Beijing",
			"example2": []*entity.ContentPart{
				{
					Type:       entity.ContentTypeBase64Data,
					Base64Data: util.Ptr(base64data),
				},
			},
			"city2": "Shanghai",
		},
		Messages: []*entity.Message{
			{
				Role: entity.RoleUser,
				Parts: []*entity.ContentPart{
					{
						Type:     entity.ContentTypeImageURL,
						ImageURL: util.Ptr("https://img0.baidu.com/it/u=1402951118,1660594928&fm=253&app=138&f=JPEG?w=800&h=1200"),
					},
					{
						Type: entity.ContentTypeText,
						Text: util.Ptr("Where is this photo taken?"),
					},
				},
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
