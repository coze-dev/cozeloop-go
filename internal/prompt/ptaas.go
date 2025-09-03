// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package prompt

import (
	"context"
	"fmt"

	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/stream"
)

// ExecuteOptions Execute选项
type ExecuteOptions struct{}

// ExecuteStreamingOptions ExecuteStreaming选项
type ExecuteStreamingOptions struct{}

// ExecuteOption Execute选项函数
type ExecuteOption func(option *ExecuteOptions)

// ExecuteStreamingOption ExecuteStreaming选项函数
type ExecuteStreamingOption func(option *ExecuteStreamingOptions)

// Execute 执行Prompt并返回结果
func (p *Provider) Execute(ctx context.Context, req *entity.ExecuteParam, options ...ExecuteOption) (entity.ExecuteResult, error) {
	result := entity.ExecuteResult{}
	// 处理选项
	opts := &ExecuteOptions{}
	for _, option := range options {
		option(opts)
	}

	// 构建请求体
	executeReq := buildExecuteRequest(req, p.config.WorkspaceID)

	// 通过OpenAPIClient发送HTTP请求
	data, err := p.openAPIClient.Execute(ctx, executeReq)
	if err != nil {
		return result, err
	}

	if data != nil {
		result.Message = toModelMessage(data.Message)
		result.FinishReason = data.FinishReason
		result.Usage = toModelTokenUsage(data.Usage)
	}
	// 转换响应
	return result, nil
}

// ExecuteStreaming 流式执行Prompt并返回流式读取器
func (p *Provider) ExecuteStreaming(ctx context.Context, req *entity.ExecuteParam, options ...ExecuteStreamingOption) (entity.StreamReader[entity.ExecuteResult], error) {
	// 处理选项
	opts := &ExecuteStreamingOptions{}
	for _, option := range options {
		option(opts)
	}

	// 构建请求体
	executeReq := buildExecuteRequest(req, p.config.WorkspaceID)

	// 通过OpenAPIClient发送流式HTTP请求
	resp, err := p.openAPIClient.ExecuteStreaming(ctx, executeReq)
	if err != nil {
		return nil, err
	}
	streamReader, err := stream.NewStreamReader(ctx, resp, func(chunk *ExecuteData, err error) (entity.ExecuteResult, error) {
		if err != nil {
			return entity.ExecuteResult{}, err
		}
		result := entity.ExecuteResult{}
		if chunk != nil {
			result.Message = toModelMessage(chunk.Message)
			result.FinishReason = chunk.FinishReason
			result.Usage = toModelTokenUsage(chunk.Usage)
		}
		return result, err
	})
	if err != nil {
		return nil, err
	}

	// 创建流式读取器
	return streamReader, nil
}

// buildExecuteRequest 构建Execute请求体
func buildExecuteRequest(req *entity.ExecuteParam, workspaceID string) ExecuteRequest {
	executeReq := ExecuteRequest{
		WorkspaceID: workspaceID,
		PromptIdentifier: &PromptQuery{
			PromptKey: req.PromptKey,
			Version:   req.Version,
		},
	}

	// 添加变量值
	if req.VariableVals != nil && len(req.VariableVals) > 0 {
		var variableVals []*VariableVal
		for key, value := range req.VariableVals {
			valueStr := fmt.Sprintf("%v", value)
			variableVals = append(variableVals, &VariableVal{
				Key:   key,
				Value: &valueStr,
			})
		}
		executeReq.VariableVals = variableVals
	}

	return executeReq
}
