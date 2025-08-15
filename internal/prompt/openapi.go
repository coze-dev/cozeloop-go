// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package prompt

import (
	"context"
	"encoding/json"
	"sort"

	"golang.org/x/sync/singleflight"

	"github.com/coze-dev/cozeloop-go/internal/httpclient"
)

const (
	mpullPromptPath         = "/v1/loop/prompts/mget"
	maxPromptQueryBatchSize = 25
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
	Role    Role    `json:"role"`
	Content *string `json:"content,omitempty"`
}

type Role string

const (
	RoleSystem      Role = "system"
	RoleUser        Role = "user"
	RoleAssistant   Role = "assistant"
	RoleTool        Role = "tool"
	RolePlaceholder Role = "placeholder"
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
