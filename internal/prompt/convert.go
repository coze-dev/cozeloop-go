// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package prompt

import (
	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/util"
	"github.com/coze-dev/cozeloop-go/spec/tracespec"
)

// toModelPrompt converts openapi.Prompt to entity.Prompt
func toModelPrompt(p *Prompt) *entity.Prompt {
	if p == nil {
		return nil
	}

	return &entity.Prompt{
		WorkspaceID:    p.WorkspaceID,
		PromptKey:      p.PromptKey,
		Version:        p.Version,
		PromptTemplate: toModelPromptTemplate(p.PromptTemplate),
		Tools:          toModelTools(p.Tools),
		ToolCallConfig: toModelToolCallConfig(p.ToolCallConfig),
		LLMConfig:      toModelLLMConfig(p.LLMConfig),
	}
}

func toModelPromptTemplate(pt *PromptTemplate) *entity.PromptTemplate {
	if pt == nil {
		return nil
	}
	return &entity.PromptTemplate{
		TemplateType: toModelTemplateType(pt.TemplateType),
		Messages:     toModelMessages(pt.Messages),
		VariableDefs: toModelVariableDefs(pt.VariableDefs),
	}
}

func toModelMessages(messages []*Message) []*entity.Message {
	if messages == nil {
		return nil
	}
	result := make([]*entity.Message, len(messages))
	for i, msg := range messages {
		if msg == nil {
			continue
		}
		result[i] = &entity.Message{
			Role:    toModelRole(msg.Role),
			Content: msg.Content,
			Parts:   toContentParts(msg.Parts),
		}
	}
	return result
}

func toContentParts(dos []*ContentPart) []*entity.ContentPart {
	if dos == nil {
		return nil
	}
	parts := make([]*entity.ContentPart, 0, len(dos))
	for _, do := range dos {
		if do == nil {
			continue
		}
		parts = append(parts, toContentPart(do))
	}
	return parts
}

func toContentPart(do *ContentPart) *entity.ContentPart {
	if do == nil {
		return nil
	}
	return &entity.ContentPart{
		Type: toContentType(util.PtrValue(do.Type)),
		Text: do.Text,
	}
}

func toContentType(do ContentType) entity.ContentType {
	switch do {
	case ContentTypeText:
		return entity.ContentTypeText
	case ContentTypeMultiPartVariable:
		return entity.ContentTypeMultiPartVariable
	default:
		return entity.ContentTypeText
	}
}

func toModelVariableDefs(defs []*VariableDef) []*entity.VariableDef {
	if defs == nil {
		return nil
	}
	result := make([]*entity.VariableDef, len(defs))
	for i, def := range defs {
		if def == nil {
			continue
		}
		result[i] = &entity.VariableDef{
			Key:  def.Key,
			Desc: def.Desc,
			Type: toModelVariableType(def.Type),
		}
	}
	return result
}

func toModelTools(tools []*Tool) []*entity.Tool {
	if tools == nil {
		return nil
	}
	result := make([]*entity.Tool, len(tools))
	for i, tool := range tools {
		if tool == nil {
			continue
		}
		result[i] = &entity.Tool{
			Type:     toModelToolType(tool.Type),
			Function: toModelFunction(tool.Function),
		}
	}
	return result
}

func toModelFunction(f *Function) *entity.Function {
	if f == nil {
		return nil
	}
	return &entity.Function{
		Name:        f.Name,
		Description: f.Description,
		Parameters:  f.Parameters,
	}
}

func toModelToolCallConfig(config *ToolCallConfig) *entity.ToolCallConfig {
	if config == nil {
		return nil
	}
	return &entity.ToolCallConfig{
		ToolChoice: toModelToolChoiceType(config.ToolChoice),
	}
}

func toModelLLMConfig(config *LLMConfig) *entity.LLMConfig {
	if config == nil {
		return nil
	}
	return &entity.LLMConfig{
		Temperature:      config.Temperature,
		MaxTokens:        config.MaxTokens,
		TopK:             config.TopK,
		TopP:             config.TopP,
		FrequencyPenalty: config.FrequencyPenalty,
		PresencePenalty:  config.PresencePenalty,
		JSONMode:         config.JSONMode,
	}
}

func toModelTemplateType(t TemplateType) entity.TemplateType {
	switch t {
	case TemplateTypeNormal:
		return entity.TemplateTypeNormal
	case TemplateTypeJinja2:
		return entity.TemplateTypeJinja2
	default:
		return entity.TemplateTypeNormal
	}
}

func toModelRole(r Role) entity.Role {
	switch r {
	case RoleSystem:
		return entity.RoleSystem
	case RoleUser:
		return entity.RoleUser
	case RoleAssistant:
		return entity.RoleAssistant
	case RoleTool:
		return entity.RoleTool
	case RolePlaceholder:
		return entity.RolePlaceholder
	default:
		return entity.RoleUser
	}
}

func toModelToolType(tt ToolType) entity.ToolType {
	switch tt {
	case ToolTypeFunction:
		return entity.ToolTypeFunction
	default:
		return entity.ToolTypeFunction
	}
}

func toModelVariableType(vt VariableType) entity.VariableType {
	switch vt {
	case VariableTypeString:
		return entity.VariableTypeString
	case VariableTypePlaceholder:
		return entity.VariableTypePlaceholder
	case VariableTypeBoolean:
		return entity.VariableTypeBoolean
	case VariableTypeFloat:
		return entity.VariableTypeFloat
	case VariableTypeInteger:
		return entity.VariableTypeInteger
	case VariableTypeObject:
		return entity.VariableTypeObject
	case VariableTypeArrayString:
		return entity.VariableTypeArrayString
	case VariableTypeArrayInteger:
		return entity.VariableTypeArrayInteger
	case VariableTypeArrayFloat:
		return entity.VariableTypeArrayFloat
	case VariableTypeArrayBoolean:
		return entity.VariableTypeArrayBoolean
	case VariableTypeArrayObject:
		return entity.VariableTypeArrayObject
	case VariableTypeMultiPart:
		return entity.VariableTypeMultiPart
	default:
		return entity.VariableTypeString
	}
}

func toModelToolChoiceType(tct ToolChoiceType) entity.ToolChoiceType {
	switch tct {
	case ToolChoiceTypeAuto:
		return entity.ToolChoiceTypeAuto
	case ToolChoiceTypeNone:
		return entity.ToolChoiceTypeNone
	default:
		return entity.ToolChoiceTypeAuto
	}
}

// ===============to span model================
func toSpanPromptInput(messages []*entity.Message, arguments map[string]any) *tracespec.PromptInput {
	return &tracespec.PromptInput{
		Templates: toSpanMessages(messages),
		Arguments: toSpanArguments(arguments),
	}
}

func toSpanArguments(arguments map[string]any) []*tracespec.PromptArgument {
	var result []*tracespec.PromptArgument
	for key, value := range arguments {
		result = append(result, &tracespec.PromptArgument{
			Key:   key,
			Value: value,
		})
	}
	return result
}

func toSpanMessages(messages []*entity.Message) []*tracespec.ModelMessage {
	var result []*tracespec.ModelMessage
	for _, msg := range messages {
		result = append(result, toSpanMessage(msg))
	}
	return result
}

func toSpanMessage(message *entity.Message) *tracespec.ModelMessage {
	if message == nil {
		return nil
	}
	return &tracespec.ModelMessage{
		Role:    string(message.Role),
		Content: util.PtrValue(message.Content),
	}
}
