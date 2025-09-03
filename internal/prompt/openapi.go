// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package prompt

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"

	"golang.org/x/sync/singleflight"

	"github.com/coze-dev/cozeloop-go/internal/httpclient"
)

const (
	mpullPromptPath            = "/v1/loop/prompts/mget"
	executePromptPath          = "/v1/loop/prompts/execute"
	executeStreamingPromptPath = "/v1/loop/prompts/execute_streaming"
	maxPromptQueryBatchSize    = 25
)

type Prompt struct {
	WorkspaceID    string          `json:"workspace_id"`
	PromptKey      string          `json:"prompt_key"`
	Version        string          `json:"version"`
	PromptTemplate *PromptTemplate `json:"prompt_template,omitempty"`
	Tools          []*Tool         `json:"tools,omitempty"`
	ToolCallConfig *ToolCallConfig `json:"tool_call_config,omitempty"`
	LLMConfig      *LLMConfig      `json:"llm_config,omitempty"`
}

type PromptTemplate struct {
	TemplateType TemplateType   `json:"template_type"`
	Messages     []*Message     `json:"messages,omitempty"`
	VariableDefs []*VariableDef `json:"variable_defs,omitempty"`
}

type TemplateType string

const (
	TemplateTypeNormal TemplateType = "normal"
	TemplateTypeJinja2 TemplateType = "jinja2"
)

type Message struct {
	Role             Role           `json:"role"`
	ReasoningContent *string        `json:"reasoning_content,omitempty"`
	Content          *string        `json:"content,omitempty"`
	Parts            []*ContentPart `json:"parts,omitempty"`
	ToolCallID       *string        `json:"tool_call_id,omitempty"`
	ToolCalls        []*ToolCall    `json:"tool_calls,omitempty"`
}

type Role string

const (
	RoleSystem      Role = "system"
	RoleUser        Role = "user"
	RoleAssistant   Role = "assistant"
	RoleTool        Role = "tool"
	RolePlaceholder Role = "placeholder"
)

type ContentPart struct {
	Type       *ContentType `json:"type"`
	Text       *string      `json:"text,omitempty"`
	ImageURL   *string      `json:"image_url,omitempty"`
	Base64Data *string      `json:"base64_data,omitempty"`
}

type ContentType string

const (
	ContentTypeText              ContentType = "text"
	ContentTypeImageURL          ContentType = "image_url"
	ContentTypeBase64Data        ContentType = "base64_data"
	ContentTypeMultiPartVariable ContentType = "multi_part_variable"
)

type ToolType string

const (
	ToolTypeFunction ToolType = "function"
)

type VariableDef struct {
	Key  string       `json:"key"`
	Desc string       `json:"desc"`
	Type VariableType `json:"type"`
}

type VariableType string

const (
	VariableTypeString       VariableType = "string"
	VariableTypePlaceholder  VariableType = "placeholder"
	VariableTypeBoolean      VariableType = "boolean"
	VariableTypeInteger      VariableType = "integer"
	VariableTypeFloat        VariableType = "float"
	VariableTypeObject       VariableType = "object"
	VariableTypeArrayString  VariableType = "array<string>"
	VariableTypeArrayBoolean VariableType = "array<boolean>"
	VariableTypeArrayInteger VariableType = "array<integer>"
	VariableTypeArrayFloat   VariableType = "array<float>"
	VariableTypeArrayObject  VariableType = "array<object>"
	VariableTypeMultiPart    VariableType = "multi_part"
)

type ToolChoiceType string

const (
	ToolChoiceTypeAuto ToolChoiceType = "auto"
	ToolChoiceTypeNone ToolChoiceType = "none"
)

type ToolCallConfig struct {
	ToolChoice ToolChoiceType `json:"tool_choice"`
}

type Tool struct {
	Type     ToolType  `json:"type"`
	Function *Function `json:"function,omitempty"`
}

type Function struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Parameters  *string `json:"parameters,omitempty"`
}

type LLMConfig struct {
	Temperature      *float64 `json:"temperature,omitempty"`
	MaxTokens        *int32   `json:"max_tokens,omitempty"`
	TopK             *int32   `json:"top_k,omitempty"`
	TopP             *float64 `json:"top_p,omitempty"`
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64 `json:"presence_penalty,omitempty"`
	JSONMode         *bool    `json:"json_mode,omitempty"`
}

type VariableVal struct {
	Key                 string         `json:"key"`
	Value               *string        `json:"value,omitempty"`
	PlaceholderMessages []*Message     `json:"placeholder_messages,omitempty"`
	MultiPartValues     []*ContentPart `json:"multi_part_values,omitempty"`
}

type ToolCall struct {
	Index        *int32        `json:"index,omitempty"`
	ID           *string       `json:"id,omitempty"`
	Type         ToolType      `json:"type"`
	FunctionCall *FunctionCall `json:"function_call,omitempty"`
}

type FunctionCall struct {
	Name      string  `json:"name"`
	Arguments *string `json:"arguments,omitempty"`
}

type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type OpenAPIClient struct {
	httpClient *httpclient.Client
	sf         singleflight.Group
}

type MPullPromptRequest struct {
	WorkSpaceID string        `json:"workspace_id"`
	Queries     []PromptQuery `json:"queries"`
}

type MPullPromptResponse struct {
	httpclient.BaseResponse
	Data PromptResultData `json:"data"`
}

type PromptQuery struct {
	PromptKey string `json:"prompt_key"`
	Version   string `json:"version"`
	Label     string `json:"label,omitempty"`
}

type PromptResultData struct {
	Items []*PromptResult `json:"items,omitempty"`
}

type PromptResult struct {
	Query  PromptQuery `json:"query"`
	Prompt *Prompt     `json:"prompt,omitempty"`
}

func (o *OpenAPIClient) MPullPrompt(ctx context.Context, req MPullPromptRequest) ([]*PromptResult, error) {
	// Sort the entire request's Queries
	sort.Slice(req.Queries, func(i, j int) bool {
		if req.Queries[i].PromptKey != req.Queries[j].PromptKey {
			return req.Queries[i].PromptKey < req.Queries[j].PromptKey
		}
		return req.Queries[i].Version < req.Queries[j].Version
	})

	// If the number of requests is less than or equal to the maximum batch size, directly use singleflight to execute
	if len(req.Queries) <= maxPromptQueryBatchSize {
		return o.singleflightMPullPrompt(ctx, req)
	}

	// Process the requests in batches
	var allPrompts []*PromptResult
	for i := 0; i < len(req.Queries); i += maxPromptQueryBatchSize {
		end := i + maxPromptQueryBatchSize
		if end > len(req.Queries) {
			end = len(req.Queries)
		}

		batchReq := MPullPromptRequest{
			WorkSpaceID: req.WorkSpaceID,
			Queries:     req.Queries[i:end],
		}

		prompts, err := o.singleflightMPullPrompt(ctx, batchReq)
		if err != nil {
			return nil, err
		}
		allPrompts = append(allPrompts, prompts...)
	}

	return allPrompts, nil
}

func (o *OpenAPIClient) singleflightMPullPrompt(ctx context.Context, req MPullPromptRequest) ([]*PromptResult, error) {
	// Queries are already sorted in the upper layer, so generate the key directly here
	b, _ := json.Marshal(req)
	key := string(b)

	v, err, _ := o.sf.Do(key, func() (interface{}, error) {
		return o.doMPullPrompt(ctx, req)
	})

	if err != nil {
		return nil, err
	}

	if v == nil {
		return nil, nil
	}

	return v.([]*PromptResult), nil
}

func (o *OpenAPIClient) doMPullPrompt(ctx context.Context, req MPullPromptRequest) ([]*PromptResult, error) {
	var resp MPullPromptResponse
	err := o.httpClient.Post(ctx, mpullPromptPath, req, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Data.Items, nil
}

type ExecuteRequest struct {
	WorkspaceID      string         `json:"workspace_id"`
	PromptIdentifier *PromptQuery   `json:"prompt_identifier,omitempty"`
	VariableVals     []*VariableVal `json:"variable_vals,omitempty"`
	Messages         []*Message     `json:"messages,omitempty"`
}

type ExecuteResponse struct {
	httpclient.BaseResponse
	Data *ExecuteData `json:"data"`
}

type ExecuteData struct {
	Message      *Message    `json:"message,omitempty"`
	FinishReason *string     `json:"finish_reason,omitempty"`
	Usage        *TokenUsage `json:"usage,omitempty"`
}

// ExecuteStreamingData 流式执行响应数据结构体
type ExecuteStreamingData struct {
	Code         *int32      `json:"code,omitempty"`
	Msg          *string     `json:"msg,omitempty"`
	Message      *Message    `json:"message,omitempty"`
	FinishReason *string     `json:"finish_reason,omitempty"`
	Usage        *TokenUsage `json:"usage,omitempty"`
}

// Execute 执行Prompt请求
func (o *OpenAPIClient) Execute(ctx context.Context, req ExecuteRequest) (*ExecuteData, error) {
	var response ExecuteResponse
	err := o.httpClient.Post(ctx, executePromptPath, req, &response)
	if err != nil {
		return nil, err
	}
	return response.Data, nil
}

// ExecuteStreaming 流式执行Prompt请求
func (o *OpenAPIClient) ExecuteStreaming(ctx context.Context, req ExecuteRequest) (*http.Response, error) {
	return o.httpClient.PostStream(ctx, executeStreamingPromptPath, req)
}
