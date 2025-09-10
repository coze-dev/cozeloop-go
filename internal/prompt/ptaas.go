// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package prompt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/consts"
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
	executeReq, err := buildExecuteRequest(req, p.config.WorkspaceID)
	if err != nil {
		return entity.ExecuteResult{}, err
	}

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
	executeReq, err := buildExecuteRequest(req, p.config.WorkspaceID)
	if err != nil {
		return nil, err
	}

	// 通过OpenAPIClient发送流式HTTP请求
	resp, err := p.openAPIClient.ExecuteStreaming(ctx, executeReq)
	if err != nil {
		return nil, err
	}

	// 创建新的流式读取器
	streamReader, err := NewExecuteStreamReader(ctx, resp)
	if err != nil {
		return nil, err
	}

	return streamReader, nil
}

// buildExecuteRequest 构建Execute请求体
func buildExecuteRequest(param *entity.ExecuteParam, workspaceID string) (ExecuteRequest, error) {
	if param == nil {
		return ExecuteRequest{}, consts.ErrInvalidParam.Wrap(fmt.Errorf("execute param is nil"))
	}
	if param.PromptKey == "" {
		return ExecuteRequest{}, consts.ErrInvalidParam.Wrap(fmt.Errorf("prompt key is empty"))
	}

	executeReq := ExecuteRequest{
		WorkspaceID: workspaceID,
		PromptIdentifier: &PromptQuery{
			PromptKey: param.PromptKey,
			Version:   param.Version,
			Label:     param.Label,
		},
		Messages: toOpenAPIMessages(param.Messages),
	}

	// 添加变量值
	var variableVals []*VariableVal
	for key, value := range param.VariableVals {
		if value == nil {
			return ExecuteRequest{}, consts.ErrInvalidParam.Wrap(fmt.Errorf("variable: %s val is nil", key))
		}

		variableVal := &VariableVal{Key: key}
		switch v := value.(type) {
		// string 类型
		case string:
			variableVal.Value = &v
		// string 指针类型
		case *string:
			if v == nil {
				return ExecuteRequest{}, consts.ErrInvalidParam.Wrap(fmt.Errorf("variable: %s val is nil", key))
			}
			variableVal.Value = v

		// Message 相关类型
		case entity.Message:
			variableVal.PlaceholderMessages = []*Message{toOpenAPIMessage(&v)}
		case *entity.Message:
			if v == nil {
				return ExecuteRequest{}, consts.ErrInvalidParam.Wrap(fmt.Errorf("variable: %s val is nil", key))
			}
			variableVal.PlaceholderMessages = []*Message{toOpenAPIMessage(v)}
		case []*entity.Message:
			var apiMsgs []*Message
			for _, msg := range v {
				apiMsgs = append(apiMsgs, toOpenAPIMessage(msg))
			}
			variableVal.PlaceholderMessages = apiMsgs
		case []entity.Message:
			var apiMsgs []*Message
			for _, msg := range v {
				apiMsgs = append(apiMsgs, toOpenAPIMessage(&msg))
			}
			variableVal.PlaceholderMessages = apiMsgs

		// ContentPart 相关类型
		case entity.ContentPart:
			variableVal.MultiPartValues = []*ContentPart{toOpenAPIContentPart(&v)}
		case *entity.ContentPart:
			if v == nil {
				return ExecuteRequest{}, consts.ErrInvalidParam.Wrap(fmt.Errorf("variable: %s val is nil", key))
			}
			variableVal.MultiPartValues = []*ContentPart{toOpenAPIContentPart(v)}
		case []*entity.ContentPart:
			var apiParts []*ContentPart
			for _, part := range v {
				apiParts = append(apiParts, toOpenAPIContentPart(part))
			}
			variableVal.MultiPartValues = apiParts
		case []entity.ContentPart:
			var apiParts []*ContentPart
			for _, part := range v {
				apiParts = append(apiParts, toOpenAPIContentPart(&part))
			}
			variableVal.MultiPartValues = apiParts

		// 其他类型序列化后传入 Value 字段
		default:
			jsonBytes, err := json.Marshal(value)
			if err != nil {
				return ExecuteRequest{}, consts.ErrInvalidParam.Wrap(fmt.Errorf("failed to marshal variable %s: %w", key, err))
			}
			jsonStr := string(jsonBytes)
			variableVal.Value = &jsonStr
		}

		variableVals = append(variableVals, variableVal)
	}
	executeReq.VariableVals = variableVals

	return executeReq, nil
}
