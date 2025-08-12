// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package entity

import "github.com/coze-dev/cozeloop-go/internal/util"

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

func (p *Prompt) DeepCopy() *Prompt {
	if p == nil {
		return nil
	}

	return &Prompt{
		WorkspaceID:    p.WorkspaceID,
		PromptKey:      p.PromptKey,
		Version:        p.Version,
		PromptTemplate: p.PromptTemplate.DeepCopy(),
		Tools:          deepCopyTools(p.Tools),
		ToolCallConfig: p.ToolCallConfig.DeepCopy(),
		LLMConfig:      p.LLMConfig.DeepCopy(),
	}
}

func (pt *PromptTemplate) DeepCopy() *PromptTemplate {
	if pt == nil {
		return nil
	}

	return &PromptTemplate{
		TemplateType: pt.TemplateType,
		Messages:     deepCopyMessages(pt.Messages),
		VariableDefs: deepCopyVariableDefs(pt.VariableDefs),
	}
}

func (m *Message) DeepCopy() *Message {
	if m == nil {
		return nil
	}

	copied := &Message{
		Role: m.Role,
	}
	if m.Content != nil {
		copied.Content = util.Ptr(*m.Content)
	}
	return copied
}

func (v *VariableDef) DeepCopy() *VariableDef {
	if v == nil {
		return nil
	}

	return &VariableDef{
		Key:  v.Key,
		Desc: v.Desc,
		Type: v.Type,
	}
}

func (t *Tool) DeepCopy() *Tool {
	if t == nil {
		return nil
	}

	copied := &Tool{
		Type: t.Type,
	}
	if t.Function != nil {
		copied.Function = t.Function.DeepCopy()
	}
	return copied
}

func (f *Function) DeepCopy() *Function {
	if f == nil {
		return nil
	}

	copied := &Function{
		Name: f.Name,
	}
	if f.Description != nil {
		copied.Description = util.Ptr(*f.Description)
	}
	if f.Parameters != nil {
		copied.Parameters = util.Ptr(*f.Parameters)
	}
	return copied
}

func (tc *ToolCallConfig) DeepCopy() *ToolCallConfig {
	if tc == nil {
		return nil
	}

	return &ToolCallConfig{
		ToolChoice: tc.ToolChoice,
	}
}

func (mc *LLMConfig) DeepCopy() *LLMConfig {
	if mc == nil {
		return nil
	}

	copied := &LLMConfig{}

	if mc.Temperature != nil {
		copied.Temperature = util.Ptr(*mc.Temperature)
	}
	if mc.MaxTokens != nil {
		copied.MaxTokens = util.Ptr(*mc.MaxTokens)
	}
	if mc.TopK != nil {
		copied.TopK = util.Ptr(*mc.TopK)
	}
	if mc.TopP != nil {
		copied.TopP = util.Ptr(*mc.TopP)
	}
	if mc.FrequencyPenalty != nil {
		copied.FrequencyPenalty = util.Ptr(*mc.FrequencyPenalty)
	}
	if mc.PresencePenalty != nil {
		copied.PresencePenalty = util.Ptr(*mc.PresencePenalty)
	}
	if mc.JSONMode != nil {
		copied.JSONMode = util.Ptr(*mc.JSONMode)
	}

	return copied
}

func deepCopyMessages(messages []*Message) []*Message {
	if messages == nil {
		return nil
	}

	copied := make([]*Message, len(messages))
	for i, msg := range messages {
		copied[i] = msg.DeepCopy()
	}
	return copied
}

func deepCopyVariableDefs(defs []*VariableDef) []*VariableDef {
	if defs == nil {
		return nil
	}

	copied := make([]*VariableDef, len(defs))
	for i, def := range defs {
		copied[i] = def.DeepCopy()
	}
	return copied
}

func deepCopyTools(tools []*Tool) []*Tool {
	if tools == nil {
		return nil
	}

	copied := make([]*Tool, len(tools))
	for i, tool := range tools {
		copied[i] = tool.DeepCopy()
	}
	return copied
}
