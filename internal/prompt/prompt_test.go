// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package prompt

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	. "github.com/bytedance/mockey"
	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/httpclient"
	"github.com/coze-dev/cozeloop-go/internal/trace"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewPromptProvider(t *testing.T) {
	Convey("Test NewPromptProvider", t, func() {
		httpClient := &httpclient.Client{}
		traceProvider := &trace.Provider{}
		options := Options{
			WorkspaceID:                "workspace1",
			PromptCacheMaxCount:        100,
			PromptCacheRefreshInterval: time.Minute,
			PromptTrace:                true,
		}

		provider := NewPromptProvider(httpClient, traceProvider, options)
		So(provider, ShouldNotBeNil)
		So(provider.config.WorkspaceID, ShouldEqual, "workspace1")
		So(provider.openAPIClient, ShouldNotBeNil)
		So(provider.traceProvider, ShouldEqual, traceProvider)
		So(provider.cache, ShouldNotBeNil)
	})
}

func TestGetPrompt(t *testing.T) {
	ctx := context.Background()
	httpClient := &httpclient.Client{}
	traceProvider := &trace.Provider{}
	options := Options{
		WorkspaceID:                "workspace1",
		PromptCacheMaxCount:        100,
		PromptCacheRefreshInterval: time.Minute,
		PromptTrace:                false,
	}
	provider := NewPromptProvider(httpClient, traceProvider, options)

	Convey("Test GetPrompt method", t, func() {
		Convey("When prompt is cached", func() {
			// Mock cache Get method
			cachedPrompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
			}
			mockGet := Mock((*PromptCache).Get).Return(cachedPrompt, true).Build()
			defer mockGet.UnPatch()

			param := GetPromptParam{
				PromptKey: "key1",
				Version:   "1.0",
			}
			options := GetPromptOptions{}

			prompt, err := provider.GetPrompt(ctx, param, options)
			So(err, ShouldBeNil)
			So(prompt, ShouldNotBeNil)
			So(prompt.WorkspaceID, ShouldEqual, "workspace1")
			So(prompt.PromptKey, ShouldEqual, "key1")
			So(prompt.Version, ShouldEqual, "1.0")
		})

		Convey("When prompt is not cached and fetched from server", func() {
			// Mock cache Get method
			mockGet := Mock((*PromptCache).Get).Return(nil, false).Build()
			defer mockGet.UnPatch()

			// Mock MPullPrompt method
			promptResult := &PromptResult{
				Query: PromptQuery{
					PromptKey: "key1",
					Version:   "1.0",
				},
				Prompt: &Prompt{
					WorkspaceID: "workspace1",
					PromptKey:   "key1",
					Version:     "1.0",
				},
			}
			mockMPull := Mock((*OpenAPIClient).MPullPrompt).Return([]*PromptResult{promptResult}, nil).Build()
			defer mockMPull.UnPatch()

			// Mock cache Set method
			mockSet := Mock((*PromptCache).Set).Return().Build()
			defer mockSet.UnPatch()

			param := GetPromptParam{
				PromptKey: "key1",
				Version:   "1.0",
			}
			options := GetPromptOptions{}

			prompt, err := provider.GetPrompt(ctx, param, options)
			So(err, ShouldBeNil)
			So(prompt, ShouldNotBeNil)
			So(prompt.WorkspaceID, ShouldEqual, "workspace1")
			So(prompt.PromptKey, ShouldEqual, "key1")
			So(prompt.Version, ShouldEqual, "1.0")
		})

		Convey("When API call fails", func() {
			// Mock cache Get method
			mockGet := Mock((*PromptCache).Get).Return(nil, false).Build()
			defer mockGet.UnPatch()

			// Mock MPullPrompt method to return error
			mockMPull := Mock((*OpenAPIClient).MPullPrompt).Return(nil, errors.New("API error")).Build()
			defer mockMPull.UnPatch()

			param := GetPromptParam{
				PromptKey: "key1",
				Version:   "1.0",
			}
			options := GetPromptOptions{}

			prompt, err := provider.GetPrompt(ctx, param, options)
			So(err, ShouldNotBeNil)
			So(prompt, ShouldBeNil)
		})

		Convey("When API returns empty results", func() {
			// Mock cache Get method
			mockGet := Mock((*PromptCache).Get).Return(nil, false).Build()
			defer mockGet.UnPatch()

			// Mock MPullPrompt method to return empty results
			mockMPull := Mock((*OpenAPIClient).MPullPrompt).Return([]*PromptResult{}, nil).Build()
			defer mockMPull.UnPatch()

			param := GetPromptParam{
				PromptKey: "key1",
				Version:   "1.0",
			}
			options := GetPromptOptions{}

			prompt, err := provider.GetPrompt(ctx, param, options)
			So(err, ShouldBeNil)
			So(prompt, ShouldBeNil)
		})

		Convey("When trace is enabled", func() {
			provider.config.PromptTrace = true
			Mock((*trace.Provider).StartSpan).Return(ctx, &trace.Span{}, nil).Build()
			Mock((*trace.Span).Finish).Return().Build()
			Mock((*trace.Span).SetTags).Return().Build()
			// Mock cache Get method
			cachedPrompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
			}
			mockGet := Mock((*PromptCache).Get).Return(cachedPrompt, true).Build()
			defer mockGet.UnPatch()

			param := GetPromptParam{
				PromptKey: "key1",
				Version:   "1.0",
			}
			options := GetPromptOptions{}

			prompt, err := provider.GetPrompt(ctx, param, options)
			So(err, ShouldBeNil)
			So(prompt, ShouldNotBeNil)
		})
	})
}

func TestPromptFormat(t *testing.T) {
	ctx := context.Background()
	httpClient := &httpclient.Client{}
	traceProvider := &trace.Provider{}
	options := Options{
		WorkspaceID:                "workspace1",
		PromptCacheMaxCount:        100,
		PromptCacheRefreshInterval: time.Minute,
		PromptTrace:                false,
	}
	provider := NewPromptProvider(httpClient, traceProvider, options)

	Convey("Test PromptFormat method", t, func() {
		Convey("When prompt is nil", func() {
			variables := map[string]any{"key1": "value1"}
			messages, err := provider.PromptFormat(ctx, nil, variables, PromptFormatOptions{})
			So(err, ShouldBeNil)
			So(messages, ShouldBeNil)
		})

		Convey("When prompt.PromptTemplate is nil", func() {
			prompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
			}
			variables := map[string]any{"key1": "value1"}
			messages, err := provider.PromptFormat(ctx, prompt, variables, PromptFormatOptions{})
			So(err, ShouldBeNil)
			So(messages, ShouldBeNil)
		})

		Convey("When prompt.PromptTemplate.Messages is empty", func() {
			prompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
				PromptTemplate: &entity.PromptTemplate{
					TemplateType: entity.TemplateTypeNormal,
					Messages:     []*entity.Message{},
				},
			}
			variables := map[string]any{"key1": "value1"}

			messages, err := provider.PromptFormat(ctx, prompt, variables, PromptFormatOptions{})
			So(err, ShouldBeNil)
			So(messages, ShouldBeNil)
		})

		Convey("When variables are valid", func() {
			content := "Hello {{key1}}"
			prompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
				PromptTemplate: &entity.PromptTemplate{
					TemplateType: entity.TemplateTypeNormal,
					Messages: []*entity.Message{
						{
							Role:    entity.RoleSystem,
							Content: &content,
						},
					},
					VariableDefs: []*entity.VariableDef{
						{
							Key:  "key1",
							Type: entity.VariableTypeString,
						},
					},
				},
			}
			variables := map[string]any{"key1": "world"}

			messages, err := provider.PromptFormat(ctx, prompt, variables, PromptFormatOptions{})
			So(err, ShouldBeNil)
			So(messages, ShouldNotBeNil)
			So(len(messages), ShouldEqual, 1)
			So(*messages[0].Content, ShouldEqual, "Hello world")
		})

		Convey("When variable type is invalid", func() {
			content := "Hello {{key1}}"
			prompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
				PromptTemplate: &entity.PromptTemplate{
					TemplateType: entity.TemplateTypeNormal,
					Messages: []*entity.Message{
						{
							Role:    entity.RoleSystem,
							Content: &content,
						},
					},
					VariableDefs: []*entity.VariableDef{
						{
							Key:  "key1",
							Type: entity.VariableTypeString,
						},
					},
				},
			}
			variables := map[string]any{"key1": 123} // Not a string

			messages, err := provider.PromptFormat(ctx, prompt, variables, PromptFormatOptions{})
			So(err, ShouldNotBeNil)
			So(messages, ShouldBeNil)
		})

		Convey("When template contains placeholder message", func() {
			placeholderContent := "placeholder_var"
			systemContent := "System prompt"
			prompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
				PromptTemplate: &entity.PromptTemplate{
					TemplateType: entity.TemplateTypeNormal,
					Messages: []*entity.Message{
						{
							Role:    entity.RoleSystem,
							Content: &systemContent,
						},
						{
							Role:    entity.RolePlaceholder,
							Content: &placeholderContent,
						},
					},
					VariableDefs: []*entity.VariableDef{
						{
							Key:  "placeholder_var",
							Type: entity.VariableTypePlaceholder,
						},
					},
				},
			}

			userContent := "User message"
			variables := map[string]any{
				"placeholder_var": []*entity.Message{
					{
						Role:    entity.RoleUser,
						Content: &userContent,
					},
				},
			}

			messages, err := provider.PromptFormat(ctx, prompt, variables, PromptFormatOptions{})
			So(err, ShouldBeNil)
			So(messages, ShouldNotBeNil)
			So(len(messages), ShouldEqual, 2)
			So(messages[0].Role, ShouldEqual, entity.RoleSystem)
			So(*messages[0].Content, ShouldEqual, "System prompt")
			So(messages[1].Role, ShouldEqual, entity.RoleUser)
			So(*messages[1].Content, ShouldEqual, "User message")
		})

		Convey("When trace is enabled", func() {
			// Mock StartSpan
			span := &trace.Span{}
			Mock((*trace.Provider).StartSpan).Return(ctx, span, nil).Build()

			// Mock span methods
			Mock((*trace.Span).SetTags).Return().Build()
			Mock((*trace.Span).Finish).Return().Build()

			content := "Hello world"
			prompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
				PromptTemplate: &entity.PromptTemplate{
					TemplateType: entity.TemplateTypeNormal,
					Messages: []*entity.Message{
						{
							Role:    entity.RoleSystem,
							Content: &content,
						},
					},
				},
			}
			variables := map[string]any{}

			defer UnPatchAll()

			messages, err := provider.PromptFormat(ctx, prompt, variables, PromptFormatOptions{})
			So(err, ShouldBeNil)
			So(messages, ShouldNotBeNil)
		})
	})
}

func TestValidateVariableValuesType(t *testing.T) {
	Convey("Test validateVariableValuesType", t, func() {
		Convey("When variableDefs is nil", func() {
			err := validateVariableValuesType(nil, map[string]any{"key1": "value1"})
			So(err, ShouldBeNil)
		})

		Convey("When variables is empty", func() {
			variableDefs := []*entity.VariableDef{
				{
					Key:  "key1",
					Type: entity.VariableTypeString,
				},
			}
			err := validateVariableValuesType(variableDefs, map[string]any{})
			So(err, ShouldBeNil)
		})

		Convey("When variable def is nil", func() {
			variableDefs := []*entity.VariableDef{nil}
			err := validateVariableValuesType(variableDefs, map[string]any{"key1": "value1"})
			So(err, ShouldBeNil)
		})

		Convey("When variable type is string and value is string", func() {
			variableDefs := []*entity.VariableDef{
				{
					Key:  "key1",
					Type: entity.VariableTypeString,
				},
			}
			err := validateVariableValuesType(variableDefs, map[string]any{"key1": "value1"})
			So(err, ShouldBeNil)
		})

		Convey("When variable type is string but value is not string", func() {
			variableDefs := []*entity.VariableDef{
				{
					Key:  "key1",
					Type: entity.VariableTypeString,
				},
			}
			err := validateVariableValuesType(variableDefs, map[string]any{"key1": 123})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "type of variable 'key1' should be string")
		})

		Convey("When variable type is placeholder and value is *Message", func() {
			variableDefs := []*entity.VariableDef{
				{
					Key:  "key1",
					Type: entity.VariableTypePlaceholder,
				},
			}
			content := "test content"
			message := &entity.Message{
				Role:    entity.RoleUser,
				Content: &content,
			}
			err := validateVariableValuesType(variableDefs, map[string]any{"key1": message})
			So(err, ShouldBeNil)
		})

		Convey("When variable type is placeholder and value is Message", func() {
			variableDefs := []*entity.VariableDef{
				{
					Key:  "key1",
					Type: entity.VariableTypePlaceholder,
				},
			}
			content := "test content"
			message := entity.Message{
				Role:    entity.RoleUser,
				Content: &content,
			}
			err := validateVariableValuesType(variableDefs, map[string]any{"key1": message})
			So(err, ShouldBeNil)
		})

		Convey("When variable type is placeholder and value is []*Message", func() {
			variableDefs := []*entity.VariableDef{
				{
					Key:  "key1",
					Type: entity.VariableTypePlaceholder,
				},
			}
			content := "test content"
			messages := []*entity.Message{
				{
					Role:    entity.RoleUser,
					Content: &content,
				},
			}
			err := validateVariableValuesType(variableDefs, map[string]any{"key1": messages})
			So(err, ShouldBeNil)
		})

		Convey("When variable type is placeholder and value is []Message", func() {
			variableDefs := []*entity.VariableDef{
				{
					Key:  "key1",
					Type: entity.VariableTypePlaceholder,
				},
			}
			content := "test content"
			messages := []entity.Message{
				{
					Role:    entity.RoleUser,
					Content: &content,
				},
			}
			err := validateVariableValuesType(variableDefs, map[string]any{"key1": messages})
			So(err, ShouldBeNil)
		})

		Convey("When variable type is placeholder but value is not Message", func() {
			variableDefs := []*entity.VariableDef{
				{
					Key:  "key1",
					Type: entity.VariableTypePlaceholder,
				},
			}
			err := validateVariableValuesType(variableDefs, map[string]any{"key1": "not a message"})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "type of variable 'key1' should be Message like object")
		})
	})
}

func TestFormatNormalMessages(t *testing.T) {
	Convey("Test formatNormalMessages", t, func() {
		Convey("When messages is empty", func() {
			results, err := formatNormalMessages(entity.TemplateTypeNormal, []*entity.Message{}, nil, nil)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 0)
		})

		Convey("When message is nil", func() {
			results, err := formatNormalMessages(entity.TemplateTypeNormal, []*entity.Message{nil}, nil, nil)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 0)
		})

		Convey("When message is placeholder", func() {
			content := "placeholder_var"
			messages := []*entity.Message{
				{
					Role:    entity.RolePlaceholder,
					Content: &content,
				},
			}
			results, err := formatNormalMessages(entity.TemplateTypeNormal, messages, nil, nil)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 1)
			So(results[0].Role, ShouldEqual, entity.RolePlaceholder)
		})

		Convey("When message content contains variables", func() {
			content := "Hello {{key1}}"
			messages := []*entity.Message{
				{
					Role:    entity.RoleSystem,
					Content: &content,
				},
			}
			variableDefs := []*entity.VariableDef{
				{
					Key:  "key1",
					Type: entity.VariableTypeString,
				},
			}
			variables := map[string]any{"key1": "world"}

			results, err := formatNormalMessages(entity.TemplateTypeNormal, messages, variableDefs, variables)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 1)
			So(*results[0].Content, ShouldEqual, "Hello world")
		})

		Convey("When message content is empty", func() {
			emptyContent := ""
			messages := []*entity.Message{
				{
					Role:    entity.RoleSystem,
					Content: &emptyContent,
				},
			}

			results, err := formatNormalMessages(entity.TemplateTypeNormal, messages, nil, nil)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 1)
			So(*results[0].Content, ShouldEqual, "")
		})

		Convey("When message content is nil", func() {
			messages := []*entity.Message{
				{
					Role:    entity.RoleSystem,
					Content: nil,
				},
			}

			results, err := formatNormalMessages(entity.TemplateTypeNormal, messages, nil, nil)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 1)
			So(results[0].Content, ShouldBeNil)
		})

		Convey("When template type is unknown", func() {
			content := "Hello world"
			messages := []*entity.Message{
				{
					Role:    entity.RoleSystem,
					Content: &content,
				},
			}

			results, err := formatNormalMessages("unknown", messages, nil, nil)
			So(err, ShouldNotBeNil)
			So(results, ShouldBeNil)
		})
	})
}

func TestFormatPlaceholderMessages(t *testing.T) {
	Convey("Test formatPlaceholderMessages", t, func() {
		Convey("When messages don't contain placeholders", func() {
			content := "Hello world"
			messages := []*entity.Message{
				{
					Role:    entity.RoleSystem,
					Content: &content,
				},
			}

			results, err := formatPlaceholderMessages(messages, nil)
			So(err, ShouldBeNil)
			So(results, ShouldNotBeNil)
			So(len(results), ShouldEqual, 1)
			So(*results[0].Content, ShouldEqual, "Hello world")
		})

		Convey("When messages contain nil", func() {
			messages := []*entity.Message{nil}

			results, err := formatPlaceholderMessages(messages, nil)
			So(err, ShouldBeNil)
			So(results, ShouldNotBeNil)
			So(len(results), ShouldEqual, 1)
			So(results[0], ShouldBeNil)
		})

		Convey("When messages contain placeholder with matching variable", func() {
			systemContent := "System prompt"
			placeholderContent := "placeholder_var"
			messages := []*entity.Message{
				{
					Role:    entity.RoleSystem,
					Content: &systemContent,
				},
				{
					Role:    entity.RolePlaceholder,
					Content: &placeholderContent,
				},
			}

			userContent := "User message"
			variables := map[string]any{
				"placeholder_var": []*entity.Message{
					{
						Role:    entity.RoleUser,
						Content: &userContent,
					},
				},
			}

			results, err := formatPlaceholderMessages(messages, variables)
			So(err, ShouldBeNil)
			So(results, ShouldNotBeNil)
			So(len(results), ShouldEqual, 2)
			So(results[0].Role, ShouldEqual, entity.RoleSystem)
			So(*results[0].Content, ShouldEqual, "System prompt")
			So(results[1].Role, ShouldEqual, entity.RoleUser)
			So(*results[1].Content, ShouldEqual, "User message")
		})

		Convey("When placeholder variable is not found", func() {
			placeholderContent := "placeholder_var"
			messages := []*entity.Message{
				{
					Role:    entity.RolePlaceholder,
					Content: &placeholderContent,
				},
			}

			variables := map[string]any{} // No matching variable

			results, err := formatPlaceholderMessages(messages, variables)
			So(err, ShouldBeNil)
			So(results, ShouldNotBeNil)
			So(len(results), ShouldEqual, 0)
		})

		Convey("When placeholder variable is nil", func() {
			placeholderContent := "placeholder_var"
			messages := []*entity.Message{
				{
					Role:    entity.RolePlaceholder,
					Content: &placeholderContent,
				},
			}

			variables := map[string]any{"placeholder_var": nil}

			results, err := formatPlaceholderMessages(messages, variables)
			So(err, ShouldBeNil)
			So(results, ShouldNotBeNil)
			So(len(results), ShouldEqual, 0)
		})

		Convey("When placeholder variable is invalid type", func() {
			placeholderContent := "placeholder_var"
			messages := []*entity.Message{
				{
					Role:    entity.RolePlaceholder,
					Content: &placeholderContent,
				},
			}

			variables := map[string]any{"placeholder_var": "not a message"}

			results, err := formatPlaceholderMessages(messages, variables)
			So(err, ShouldNotBeNil)
			So(results, ShouldBeNil)
		})
	})
}

func TestConvertMessageLikeObjectToMessages(t *testing.T) {
	Convey("Test convertMessageLikeObjectToMessages", t, func() {
		Convey("When object is []*entity.Message", func() {
			content := "test content"
			input := []*entity.Message{
				{
					Role:    entity.RoleUser,
					Content: &content,
				},
			}

			result, err := convertMessageLikeObjectToMessages(input)
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
			So(len(result), ShouldEqual, 1)
			So(result[0].Role, ShouldEqual, entity.RoleUser)
			So(*result[0].Content, ShouldEqual, content)
		})

		Convey("When object is []entity.Message", func() {
			content := "test content"
			input := []entity.Message{
				{
					Role:    entity.RoleUser,
					Content: &content,
				},
			}

			result, err := convertMessageLikeObjectToMessages(input)
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
			So(len(result), ShouldEqual, 1)
			So(result[0].Role, ShouldEqual, entity.RoleUser)
			So(*result[0].Content, ShouldEqual, content)
		})

		Convey("When object is *entity.Message", func() {
			content := "test content"
			input := &entity.Message{
				Role:    entity.RoleUser,
				Content: &content,
			}

			result, err := convertMessageLikeObjectToMessages(input)
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
			So(len(result), ShouldEqual, 1)
			So(result[0].Role, ShouldEqual, entity.RoleUser)
			So(*result[0].Content, ShouldEqual, content)
		})

		Convey("When object is entity.Message", func() {
			content := "test content"
			input := entity.Message{
				Role:    entity.RoleUser,
				Content: &content,
			}

			result, err := convertMessageLikeObjectToMessages(input)
			So(err, ShouldBeNil)
			So(result, ShouldNotBeNil)
			So(len(result), ShouldEqual, 1)
			So(result[0].Role, ShouldEqual, entity.RoleUser)
			So(*result[0].Content, ShouldEqual, content)
		})

		Convey("When object is invalid type", func() {
			input := "not a message"

			result, err := convertMessageLikeObjectToMessages(input)
			So(err, ShouldNotBeNil)
			So(result, ShouldBeNil)
		})
	})
}

func TestRenderTextContent(t *testing.T) {
	Convey("Test renderTextContent function", t, func() {
		Convey("When template is normal and variables are valid", func() {
			template := "Hello {{key1}}"
			variableDefs := map[string]*entity.VariableDef{
				"key1": {
					Key:  "key1",
					Type: entity.VariableTypeString,
				},
			}
			variables := map[string]any{"key1": "world"}

			result, err := renderTextContent(entity.TemplateTypeNormal, template, variableDefs, variables)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "Hello world")
		})

		Convey("When variable is not defined", func() {
			template := "Hello {{key1}}"
			variableDefs := map[string]*entity.VariableDef{} // No key1 defined
			variables := map[string]any{"key1": "world"}

			result, err := renderTextContent(entity.TemplateTypeNormal, template, variableDefs, variables)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "Hello {{key1}}")
		})

		Convey("When variable is defined but not provided", func() {
			template := "Hello {{key1}}"
			variableDefs := map[string]*entity.VariableDef{
				"key1": {
					Key:  "key1",
					Type: entity.VariableTypeString,
				},
			}
			variables := map[string]any{} // No key1 provided

			result, err := renderTextContent(entity.TemplateTypeNormal, template, variableDefs, variables)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "Hello ")
		})

		Convey("When template type is normal", func() {
			template := "Hello {{key1}}"
			variableDefs := map[string]*entity.VariableDef{
				"key1": {
					Key:  "key1",
					Type: entity.VariableTypeString,
				},
			}
			variables := map[string]any{"key1": "world"}

			result, err := renderTextContent(entity.TemplateTypeNormal, template, variableDefs, variables)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "Hello world")
		})

		Convey("When template type is unknown", func() {
			template := "Hello {{key1}}"
			variableDefs := map[string]*entity.VariableDef{
				"key1": {
					Key:  "key1",
					Type: entity.VariableTypeString,
				},
			}
			variables := map[string]any{"key1": "world"}

			result, err := renderTextContent("unknown", template, variableDefs, variables)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "unknown template type")
			So(result, ShouldEqual, "")
		})

		Convey("When template has multiple variables", func() {
			template := "{{greeting}} {{name}}!"
			variableDefs := map[string]*entity.VariableDef{
				"greeting": {
					Key:  "greeting",
					Type: entity.VariableTypeString,
				},
				"name": {
					Key:  "name",
					Type: entity.VariableTypeString,
				},
			}
			variables := map[string]any{
				"greeting": "Hello",
				"name":     "world",
			}

			result, err := renderTextContent(entity.TemplateTypeNormal, template, variableDefs, variables)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "Hello world!")
		})

		Convey("When template has non-string variable", func() {
			template := "Count: {{count}}"
			variableDefs := map[string]*entity.VariableDef{
				"count": {
					Key:  "count",
					Type: entity.VariableTypeString,
				},
			}
			variables := map[string]any{
				"count": 42,
			}

			result, err := renderTextContent(entity.TemplateTypeNormal, template, variableDefs, variables)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "Count: 42")
		})
	})
}

func TestDoGetPrompt(t *testing.T) {
	ctx := context.Background()
	httpClient := &httpclient.Client{}
	traceProvider := &trace.Provider{}
	options := Options{
		WorkspaceID:                "workspace1",
		PromptCacheMaxCount:        100,
		PromptCacheRefreshInterval: time.Minute,
		PromptTrace:                true,
	}
	provider := NewPromptProvider(httpClient, traceProvider, options)

	Convey("Test doGetPrompt method", t, func() {
		Convey("When prompt is cached", func() {
			// Mock cache Get method
			cachedPrompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
			}
			Mock((*PromptCache).Get).Return(cachedPrompt, true).Build()
			defer UnPatchAll()

			param := GetPromptParam{
				PromptKey: "key1",
				Version:   "1.0",
			}
			options := GetPromptOptions{}

			prompt, err := provider.doGetPrompt(ctx, param, options)
			So(err, ShouldBeNil)
			So(prompt, ShouldNotBeNil)
			So(prompt.WorkspaceID, ShouldEqual, "workspace1")
			So(prompt.PromptKey, ShouldEqual, "key1")
			So(prompt.Version, ShouldEqual, "1.0")
		})

		Convey("When prompt is not cached but found on server", func() {
			// Mock cache Get method
			Mock((*PromptCache).Get).Return(nil, false).Build()

			// Mock MPullPrompt method
			promptResult := &PromptResult{
				Query: PromptQuery{
					PromptKey: "key1",
					Version:   "1.0",
				},
				Prompt: &Prompt{
					WorkspaceID: "workspace1",
					PromptKey:   "key1",
					Version:     "1.0",
				},
			}
			Mock((*OpenAPIClient).MPullPrompt).Return([]*PromptResult{promptResult}, nil).Build()

			// Mock cache Set method
			Mock((*PromptCache).Set).Return().Build()

			defer UnPatchAll()

			param := GetPromptParam{
				PromptKey: "key1",
				Version:   "1.0",
			}
			options := GetPromptOptions{}

			prompt, err := provider.doGetPrompt(ctx, param, options)
			So(err, ShouldBeNil)
			So(prompt, ShouldNotBeNil)
			So(prompt.WorkspaceID, ShouldEqual, "workspace1")
			So(prompt.PromptKey, ShouldEqual, "key1")
			So(prompt.Version, ShouldEqual, "1.0")
		})

		Convey("When MPullPrompt returns error", func() {
			// Mock cache Get method
			Mock((*PromptCache).Get).Return(nil, false).Build()

			// Mock MPullPrompt method to return error
			Mock((*OpenAPIClient).MPullPrompt).Return(nil, errors.New("API error")).Build()

			defer UnPatchAll()

			param := GetPromptParam{
				PromptKey: "key1",
				Version:   "1.0",
			}
			options := GetPromptOptions{}

			prompt, err := provider.doGetPrompt(ctx, param, options)
			So(err, ShouldNotBeNil)
			So(prompt, ShouldBeNil)
			So(err.Error(), ShouldEqual, "API error")
		})

		Convey("When MPullPrompt returns empty results", func() {
			// Mock cache Get method
			Mock((*PromptCache).Get).Return(nil, false).Build()

			// Mock MPullPrompt method to return empty results
			Mock((*OpenAPIClient).MPullPrompt).Return([]*PromptResult{}, nil).Build()

			defer UnPatchAll()

			param := GetPromptParam{
				PromptKey: "key1",
				Version:   "1.0",
			}
			options := GetPromptOptions{}

			prompt, err := provider.doGetPrompt(ctx, param, options)
			So(err, ShouldBeNil)
			So(prompt, ShouldBeNil)
		})
	})
}

func TestDoPromptFormat(t *testing.T) {
	ctx := context.Background()
	httpClient := &httpclient.Client{}
	traceProvider := &trace.Provider{}
	options := Options{
		WorkspaceID:                "workspace1",
		PromptCacheMaxCount:        100,
		PromptCacheRefreshInterval: time.Minute,
		PromptTrace:                true,
	}
	provider := NewPromptProvider(httpClient, traceProvider, options)

	Convey("Test doPromptFormat method", t, func() {
		Convey("When prompt template is nil", func() {
			prompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
			}
			variables := map[string]any{}

			messages, err := provider.doPromptFormat(ctx, prompt, variables)
			So(err, ShouldBeNil)
			So(messages, ShouldBeNil)
		})

		Convey("When prompt template has no messages", func() {
			prompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
				PromptTemplate: &entity.PromptTemplate{
					TemplateType: entity.TemplateTypeNormal,
				},
			}
			variables := map[string]any{}

			messages, err := provider.doPromptFormat(ctx, prompt, variables)
			So(err, ShouldBeNil)
			So(messages, ShouldBeNil)
		})

		Convey("When variable validation fails", func() {
			content := "Hello {{.key1}}"
			prompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
				PromptTemplate: &entity.PromptTemplate{
					TemplateType: entity.TemplateTypeNormal,
					Messages: []*entity.Message{
						{
							Role:    entity.RoleSystem,
							Content: &content,
						},
					},
					VariableDefs: []*entity.VariableDef{
						{
							Key:  "key1",
							Type: entity.VariableTypeString,
						},
					},
				},
			}
			variables := map[string]any{"key1": 123} // Not a string

			messages, err := provider.doPromptFormat(ctx, prompt, variables)
			So(err, ShouldNotBeNil)
			So(messages, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "type of variable 'key1' should be string")
		})

		Convey("When formatting normal messages fails", func() {
			content := "Hello {{.key1}}"
			prompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
				PromptTemplate: &entity.PromptTemplate{
					TemplateType: "unknown", // Unknown template type will cause an error
					Messages: []*entity.Message{
						{
							Role:    entity.RoleSystem,
							Content: &content,
						},
					},
					VariableDefs: []*entity.VariableDef{
						{
							Key:  "key1",
							Type: entity.VariableTypeString,
						},
					},
				},
			}
			variables := map[string]any{"key1": "world"}

			messages, err := provider.doPromptFormat(ctx, prompt, variables)
			So(err, ShouldNotBeNil)
			So(messages, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "unknown template type")
		})

		Convey("When formatting placeholder messages fails", func() {
			systemContent := "System prompt"
			placeholderContent := "placeholder_var"
			prompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
				PromptTemplate: &entity.PromptTemplate{
					TemplateType: entity.TemplateTypeNormal,
					Messages: []*entity.Message{
						{
							Role:    entity.RoleSystem,
							Content: &systemContent,
						},
						{
							Role:    entity.RolePlaceholder,
							Content: &placeholderContent,
						},
					},
					VariableDefs: []*entity.VariableDef{
						{
							Key:  "placeholder_var",
							Type: entity.VariableTypePlaceholder,
						},
					},
				},
			}
			variables := map[string]any{
				"placeholder_var": "not a message", // Invalid type for placeholder
			}

			messages, err := provider.doPromptFormat(ctx, prompt, variables)
			So(err, ShouldNotBeNil)
			So(messages, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "type of variable 'placeholder_var' should be Message like object")
		})

		Convey("When formatting succeeds", func() {
			systemContent := "Hello {{key1}}"
			prompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
				PromptTemplate: &entity.PromptTemplate{
					TemplateType: entity.TemplateTypeNormal,
					Messages: []*entity.Message{
						{
							Role:    entity.RoleSystem,
							Content: &systemContent,
						},
					},
					VariableDefs: []*entity.VariableDef{
						{
							Key:  "key1",
							Type: entity.VariableTypeString,
						},
					},
				},
			}
			variables := map[string]any{"key1": "world"}

			messages, err := provider.doPromptFormat(ctx, prompt, variables)
			So(err, ShouldBeNil)
			So(messages, ShouldNotBeNil)
			So(len(messages), ShouldEqual, 1)
			So(messages[0].Role, ShouldEqual, entity.RoleSystem)
			So(*messages[0].Content, ShouldEqual, "Hello world")
		})

		Convey("When formatting succeeds with placeholder", func() {
			systemContent := "System prompt"
			placeholderContent := "placeholder_var"
			prompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
				PromptTemplate: &entity.PromptTemplate{
					TemplateType: entity.TemplateTypeNormal,
					Messages: []*entity.Message{
						{
							Role:    entity.RoleSystem,
							Content: &systemContent,
						},
						{
							Role:    entity.RolePlaceholder,
							Content: &placeholderContent,
						},
					},
					VariableDefs: []*entity.VariableDef{
						{
							Key:  "placeholder_var",
							Type: entity.VariableTypePlaceholder,
						},
					},
				},
			}

			userContent := "User message"
			variables := map[string]any{
				"placeholder_var": []*entity.Message{
					{
						Role:    entity.RoleUser,
						Content: &userContent,
					},
				},
			}

			messages, err := provider.doPromptFormat(ctx, prompt, variables)
			So(err, ShouldBeNil)
			So(messages, ShouldNotBeNil)
			So(len(messages), ShouldEqual, 2)
			So(messages[0].Role, ShouldEqual, entity.RoleSystem)
			So(*messages[0].Content, ShouldEqual, "System prompt")
			So(messages[1].Role, ShouldEqual, entity.RoleUser)
			So(*messages[1].Content, ShouldEqual, "User message")
		})
	})
}
func TestValidateVariableValuesType_ExtendedTypes(t *testing.T) {
	Convey("Test validateVariableValuesType for extended types", t, func() {
		Convey("When variable type is boolean", func() {
			Convey("Valid bool value", func() {
				variableDefs := []*entity.VariableDef{
					{
						Key:  "key1",
						Type: entity.VariableTypeBoolean,
					},
				}
				err := validateVariableValuesType(variableDefs, map[string]any{"key1": true})
				So(err, ShouldBeNil)
			})

			Convey("Invalid non-bool value", func() {
				variableDefs := []*entity.VariableDef{
					{
						Key:  "key1",
						Type: entity.VariableTypeBoolean,
					},
				}
				err := validateVariableValuesType(variableDefs, map[string]any{"key1": "not a bool"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "type of variable 'key1' should be bool")
			})
		})

		Convey("When variable type is integer", func() {
			Convey("Valid int value", func() {
				variableDefs := []*entity.VariableDef{
					{
						Key:  "key1",
						Type: entity.VariableTypeInteger,
					},
				}
				err := validateVariableValuesType(variableDefs, map[string]any{"key1": 42})
				So(err, ShouldBeNil)
			})

			Convey("Valid int64 value", func() {
				variableDefs := []*entity.VariableDef{
					{
						Key:  "key1",
						Type: entity.VariableTypeInteger,
					},
				}
				err := validateVariableValuesType(variableDefs, map[string]any{"key1": int64(42)})
				So(err, ShouldBeNil)
			})

			Convey("Valid int32 value", func() {
				variableDefs := []*entity.VariableDef{
					{
						Key:  "key1",
						Type: entity.VariableTypeInteger,
					},
				}
				err := validateVariableValuesType(variableDefs, map[string]any{"key1": int32(42)})
				So(err, ShouldBeNil)
			})

			Convey("Invalid non-integer value", func() {
				variableDefs := []*entity.VariableDef{
					{
						Key:  "key1",
						Type: entity.VariableTypeInteger,
					},
				}
				err := validateVariableValuesType(variableDefs, map[string]any{"key1": "not an int"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "type of variable 'key1' should be int or int64 or int32")
			})
		})

		Convey("When variable type is float", func() {
			Convey("Valid float64 value", func() {
				variableDefs := []*entity.VariableDef{
					{
						Key:  "key1",
						Type: entity.VariableTypeFloat,
					},
				}
				err := validateVariableValuesType(variableDefs, map[string]any{"key1": 3.14})
				So(err, ShouldBeNil)
			})

			Convey("Valid float32 value", func() {
				variableDefs := []*entity.VariableDef{
					{
						Key:  "key1",
						Type: entity.VariableTypeFloat,
					},
				}
				err := validateVariableValuesType(variableDefs, map[string]any{"key1": float32(3.14)})
				So(err, ShouldBeNil)
			})

			Convey("Invalid non-float value", func() {
				variableDefs := []*entity.VariableDef{
					{
						Key:  "key1",
						Type: entity.VariableTypeFloat,
					},
				}
				err := validateVariableValuesType(variableDefs, map[string]any{"key1": "not a float"})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "type of variable 'key1' should be float64 or float32")
			})
		})

		Convey("When variable type is array string", func() {
			Convey("Valid []string value", func() {
				variableDefs := []*entity.VariableDef{
					{
						Key:  "key1",
						Type: entity.VariableTypeArrayString,
					},
				}
				err := validateVariableValuesType(variableDefs, map[string]any{"key1": []string{"a", "b", "c"}})
				So(err, ShouldBeNil)
			})

			Convey("Invalid non-[]string value", func() {
				variableDefs := []*entity.VariableDef{
					{
						Key:  "key1",
						Type: entity.VariableTypeArrayString,
					},
				}
				err := validateVariableValuesType(variableDefs, map[string]any{"key1": []int{1, 2, 3}})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "type of variable 'key1' should be []string")
			})
		})

		Convey("When variable type is array boolean", func() {
			Convey("Valid []bool value", func() {
				variableDefs := []*entity.VariableDef{
					{
						Key:  "key1",
						Type: entity.VariableTypeArrayBoolean,
					},
				}
				err := validateVariableValuesType(variableDefs, map[string]any{"key1": []bool{true, false, true}})
				So(err, ShouldBeNil)
			})

			Convey("Invalid non-[]bool value", func() {
				variableDefs := []*entity.VariableDef{
					{
						Key:  "key1",
						Type: entity.VariableTypeArrayBoolean,
					},
				}
				err := validateVariableValuesType(variableDefs, map[string]any{"key1": []string{"true", "false"}})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "type of variable 'key1' should be []bool")
			})
		})

		Convey("When variable type is array integer", func() {
			Convey("Valid []int value", func() {
				variableDefs := []*entity.VariableDef{
					{
						Key:  "key1",
						Type: entity.VariableTypeArrayInteger,
					},
				}
				err := validateVariableValuesType(variableDefs, map[string]any{"key1": []int{1, 2, 3}})
				So(err, ShouldBeNil)
			})

			Convey("Valid []int64 value", func() {
				variableDefs := []*entity.VariableDef{
					{
						Key:  "key1",
						Type: entity.VariableTypeArrayInteger,
					},
				}
				err := validateVariableValuesType(variableDefs, map[string]any{"key1": []int64{1, 2, 3}})
				So(err, ShouldBeNil)
			})

			Convey("Valid []int32 value", func() {
				variableDefs := []*entity.VariableDef{
					{
						Key:  "key1",
						Type: entity.VariableTypeArrayInteger,
					},
				}
				err := validateVariableValuesType(variableDefs, map[string]any{"key1": []int32{1, 2, 3}})
				So(err, ShouldBeNil)
			})

			Convey("Invalid non-[]int value", func() {
				variableDefs := []*entity.VariableDef{
					{
						Key:  "key1",
						Type: entity.VariableTypeArrayInteger,
					},
				}
				err := validateVariableValuesType(variableDefs, map[string]any{"key1": []string{"1", "2", "3"}})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "type of variable 'key1' should be []int or []int64 or []int32")
			})
		})

		Convey("When variable type is array float", func() {
			Convey("Valid []float64 value", func() {
				variableDefs := []*entity.VariableDef{
					{
						Key:  "key1",
						Type: entity.VariableTypeArrayFloat,
					},
				}
				err := validateVariableValuesType(variableDefs, map[string]any{"key1": []float64{1.1, 2.2, 3.3}})
				So(err, ShouldBeNil)
			})

			Convey("Valid []float32 value", func() {
				variableDefs := []*entity.VariableDef{
					{
						Key:  "key1",
						Type: entity.VariableTypeArrayFloat,
					},
				}
				err := validateVariableValuesType(variableDefs, map[string]any{"key1": []float32{1.1, 2.2, 3.3}})
				So(err, ShouldBeNil)
			})

			Convey("Invalid non-[]float value", func() {
				variableDefs := []*entity.VariableDef{
					{
						Key:  "key1",
						Type: entity.VariableTypeArrayFloat,
					},
				}
				err := validateVariableValuesType(variableDefs, map[string]any{"key1": []string{"1.1", "2.2", "3.3"}})
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "type of variable 'key1' should be []float64 or []float32")
			})
		})

		Convey("When mixing different variable types", func() {
			variableDefs := []*entity.VariableDef{
				{
					Key:  "str_var",
					Type: entity.VariableTypeString,
				},
				{
					Key:  "bool_var",
					Type: entity.VariableTypeBoolean,
				},
				{
					Key:  "int_var",
					Type: entity.VariableTypeInteger,
				},
				{
					Key:  "float_var",
					Type: entity.VariableTypeFloat,
				},
				{
					Key:  "arr_str_var",
					Type: entity.VariableTypeArrayString,
				},
			}
			variables := map[string]any{
				"str_var":     "hello",
				"bool_var":    true,
				"int_var":     42,
				"float_var":   3.14,
				"arr_str_var": []string{"a", "b"},
			}
			err := validateVariableValuesType(variableDefs, variables)
			So(err, ShouldBeNil)
		})
	})
}

func TestRenderTextContent_Jinja2(t *testing.T) {
	Convey("Test renderTextContent function with Jinja2 template type", t, func() {
		Convey("When template type is Jinja2", func() {
			Convey("Basic Jinja2 template rendering", func() {
				template := "Hello {{ name }}"
				variableDefs := map[string]*entity.VariableDef{
					"name": {
						Key:  "name",
						Type: entity.VariableTypeString,
					},
				}
				variables := map[string]any{"name": "world"}

				result, err := renderTextContent(entity.TemplateTypeJinja2, template, variableDefs, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "Hello world")
			})

			Convey("Jinja2 with conditional logic", func() {
				template := "{% if active %}User is active{% else %}User is inactive{% endif %}"
				variableDefs := map[string]*entity.VariableDef{
					"active": {
						Key:  "active",
						Type: entity.VariableTypeBoolean,
					},
				}
				variables := map[string]any{"active": true}

				result, err := renderTextContent(entity.TemplateTypeJinja2, template, variableDefs, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "User is active")
			})

			Convey("Jinja2 with loop", func() {
				template := "Items: {% for item in items %}{{ item }}{% if not loop.last %}, {% endif %}{% endfor %}"
				variableDefs := map[string]*entity.VariableDef{
					"items": {
						Key:  "items",
						Type: entity.VariableTypeArrayString,
					},
				}
				variables := map[string]any{"items": []string{"apple", "banana", "cherry"}}

				result, err := renderTextContent(entity.TemplateTypeJinja2, template, variableDefs, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "Items: apple, banana, cherry")
			})

			Convey("Jinja2 template error handling", func() {
				template := "Hello {{ undefined_variable.property }}" // Runtime error
				variableDefs := map[string]*entity.VariableDef{
					"name": {
						Key:  "name",
						Type: entity.VariableTypeString,
					},
				}
				variables := map[string]any{"name": "world"}

				result, err := renderTextContent(entity.TemplateTypeJinja2, template, variableDefs, variables)
				So(err, ShouldNotBeNil)
				So(result, ShouldEqual, "")
				So(err.Error(), ShouldContainSubstring, "template render error")
			})
		})

		Convey("Comparison between Normal and Jinja2 templates", func() {
			Convey("Normal template behavior", func() {
				template := "Hello {{name}}"
				variableDefs := map[string]*entity.VariableDef{
					"name": {
						Key:  "name",
						Type: entity.VariableTypeString,
					},
				}
				variables := map[string]any{"name": "world"}

				result, err := renderTextContent(entity.TemplateTypeNormal, template, variableDefs, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "Hello world")
			})

			Convey("Jinja2 template behavior with same input", func() {
				template := "Hello {{ name }}"
				variableDefs := map[string]*entity.VariableDef{
					"name": {
						Key:  "name",
						Type: entity.VariableTypeString,
					},
				}
				variables := map[string]any{"name": "world"}

				result, err := renderTextContent(entity.TemplateTypeJinja2, template, variableDefs, variables)
				So(err, ShouldBeNil)
				So(result, ShouldEqual, "Hello world")
			})

			Convey("Different behavior with complex syntax", func() {
				// Normal template doesn't support conditional logic
				template := "{% if active %}Active{% else %}Inactive{% endif %}"
				variableDefs := map[string]*entity.VariableDef{
					"active": {
						Key:  "active",
						Type: entity.VariableTypeBoolean,
					},
				}
				variables := map[string]any{"active": true}

				// Normal template treats this as literal text
				normalResult, err := renderTextContent(entity.TemplateTypeNormal, template, variableDefs, variables)
				So(err, ShouldBeNil)
				So(normalResult, ShouldEqual, template) // Should remain unchanged

				// Jinja2 template processes the conditional
				jinja2Result, err := renderTextContent(entity.TemplateTypeJinja2, template, variableDefs, variables)
				So(err, ShouldBeNil)
				So(jinja2Result, ShouldEqual, "Active")
			})
		})
	})
}

func TestPromptFormat_Jinja2Integration(t *testing.T) {
	ctx := context.Background()
	httpClient := &httpclient.Client{}
	traceProvider := &trace.Provider{}
	options := Options{
		WorkspaceID:                "workspace1",
		PromptCacheMaxCount:        100,
		PromptCacheRefreshInterval: time.Minute,
		PromptTrace:                false,
	}
	provider := NewPromptProvider(httpClient, traceProvider, options)

	Convey("Test PromptFormat method with Jinja2 integration", t, func() {
		Convey("Basic Jinja2 template formatting", func() {
			content := "Hello {{ name }}"
			prompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
				PromptTemplate: &entity.PromptTemplate{
					TemplateType: entity.TemplateTypeJinja2,
					Messages: []*entity.Message{
						{
							Role:    entity.RoleSystem,
							Content: &content,
						},
					},
					VariableDefs: []*entity.VariableDef{
						{
							Key:  "name",
							Type: entity.VariableTypeString,
						},
					},
				},
			}
			variables := map[string]any{"name": "world"}

			messages, err := provider.PromptFormat(ctx, prompt, variables, PromptFormatOptions{})
			So(err, ShouldBeNil)
			So(messages, ShouldNotBeNil)
			So(len(messages), ShouldEqual, 1)
			So(*messages[0].Content, ShouldEqual, "Hello world")
		})

		Convey("Jinja2 with multiple variable types", func() {
			systemContent := "User {{ name }} ({{ age }} years old) is {% if active %}active{% else %}inactive{% endif %}"
			userContent := "My scores: {% for score in scores %}{{ score }}{% if not loop.last %}, {% endif %}{% endfor %}"
			prompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
				PromptTemplate: &entity.PromptTemplate{
					TemplateType: entity.TemplateTypeJinja2,
					Messages: []*entity.Message{
						{
							Role:    entity.RoleSystem,
							Content: &systemContent,
						},
						{
							Role:    entity.RoleUser,
							Content: &userContent,
						},
					},
					VariableDefs: []*entity.VariableDef{
						{
							Key:  "name",
							Type: entity.VariableTypeString,
						},
						{
							Key:  "age",
							Type: entity.VariableTypeInteger,
						},
						{
							Key:  "active",
							Type: entity.VariableTypeBoolean,
						},
						{
							Key:  "scores",
							Type: entity.VariableTypeArrayInteger,
						},
					},
				},
			}
			variables := map[string]any{
				"name":   "Alice",
				"age":    30,
				"active": true,
				"scores": []int{85, 92, 78},
			}

			messages, err := provider.PromptFormat(ctx, prompt, variables, PromptFormatOptions{})
			So(err, ShouldBeNil)
			So(messages, ShouldNotBeNil)
			So(len(messages), ShouldEqual, 2)
			So(*messages[0].Content, ShouldEqual, "User Alice (30 years old) is active")
			So(*messages[1].Content, ShouldEqual, "My scores: 85, 92, 78")
		})

		Convey("Jinja2 with complex logic", func() {
			content := `{% if user.role == "admin" %}
Welcome, Administrator {{ user.name }}!
Your permissions: {% for perm in user.permissions %}{{ perm }}{% if not loop.last %}, {% endif %}{% endfor %}
{% elif user.role == "user" %}
Hello, {{ user.name }}!
{% else %}
Access denied.
{% endif %}`
			prompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
				PromptTemplate: &entity.PromptTemplate{
					TemplateType: entity.TemplateTypeJinja2,
					Messages: []*entity.Message{
						{
							Role:    entity.RoleSystem,
							Content: &content,
						},
					},
					VariableDefs: []*entity.VariableDef{
						{
							Key:  "user",
							Type: entity.VariableTypeObject,
						},
					},
				},
			}
			variables := map[string]any{
				"user": map[string]any{
					"role":        "admin",
					"name":        "Alice",
					"permissions": []string{"read", "write", "delete"},
				},
			}

			messages, err := provider.PromptFormat(ctx, prompt, variables, PromptFormatOptions{})
			So(err, ShouldBeNil)
			So(messages, ShouldNotBeNil)
			So(len(messages), ShouldEqual, 1)
			So(*messages[0].Content, ShouldContainSubstring, "Welcome, Administrator Alice!")
			So(*messages[0].Content, ShouldContainSubstring, "read, write, delete")
		})

		Convey("Jinja2 with placeholder messages", func() {
			systemContent := "System: {{ system_msg }}"
			placeholderContent := "conversation_history"
			prompt := &entity.Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
				PromptTemplate: &entity.PromptTemplate{
					TemplateType: entity.TemplateTypeJinja2,
					Messages: []*entity.Message{
						{
							Role:    entity.RoleSystem,
							Content: &systemContent,
						},
						{
							Role:    entity.RolePlaceholder,
							Content: &placeholderContent,
						},
					},
					VariableDefs: []*entity.VariableDef{
						{
							Key:  "system_msg",
							Type: entity.VariableTypeString,
						},
						{
							Key:  "conversation_history",
							Type: entity.VariableTypePlaceholder,
						},
					},
				},
			}

			userContent1 := "First user message"
			userContent2 := "Second user message"
			variables := map[string]any{
				"system_msg": "You are a helpful assistant",
				"conversation_history": []*entity.Message{
					{
						Role:    entity.RoleUser,
						Content: &userContent1,
					},
					{
						Role:    entity.RoleUser,
						Content: &userContent2,
					},
				},
			}

			messages, err := provider.PromptFormat(ctx, prompt, variables, PromptFormatOptions{})
			So(err, ShouldBeNil)
			So(messages, ShouldNotBeNil)
			So(len(messages), ShouldEqual, 3)
			So(*messages[0].Content, ShouldEqual, "System: You are a helpful assistant")
			So(*messages[1].Content, ShouldEqual, "First user message")
			So(*messages[2].Content, ShouldEqual, "Second user message")
		})

		Convey("Error handling in Jinja2 integration", func() {
			Convey("Invalid Jinja2 template syntax", func() {
				content := "Hello {{ undefined_variable.property }}" // Runtime error
				prompt := &entity.Prompt{
					WorkspaceID: "workspace1",
					PromptKey:   "key1",
					Version:     "1.0",
					PromptTemplate: &entity.PromptTemplate{
						TemplateType: entity.TemplateTypeJinja2,
						Messages: []*entity.Message{
							{
								Role:    entity.RoleSystem,
								Content: &content,
							},
						},
						VariableDefs: []*entity.VariableDef{
							{
								Key:  "name",
								Type: entity.VariableTypeString,
							},
						},
					},
				}
				variables := map[string]any{"name": "world"}

				messages, err := provider.PromptFormat(ctx, prompt, variables, PromptFormatOptions{})
				So(err, ShouldNotBeNil)
				So(messages, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "template render error")
			})

			Convey("Variable type validation failure with Jinja2", func() {
				content := "Hello {{ name }}"
				prompt := &entity.Prompt{
					WorkspaceID: "workspace1",
					PromptKey:   "key1",
					Version:     "1.0",
					PromptTemplate: &entity.PromptTemplate{
						TemplateType: entity.TemplateTypeJinja2,
						Messages: []*entity.Message{
							{
								Role:    entity.RoleSystem,
								Content: &content,
							},
						},
						VariableDefs: []*entity.VariableDef{
							{
								Key:  "name",
								Type: entity.VariableTypeString,
							},
						},
					},
				}
				variables := map[string]any{"name": 123} // Should be string

				messages, err := provider.PromptFormat(ctx, prompt, variables, PromptFormatOptions{})
				So(err, ShouldNotBeNil)
				So(messages, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "type of variable 'name' should be string")
			})
		})

		Convey("Performance and edge cases", func() {
			Convey("Large template with many variables", func() {
				content := "Users: {% for user in users %}{{ user.name }} ({{ user.age }}){% if not loop.last %}, {% endif %}{% endfor %}"
				prompt := &entity.Prompt{
					WorkspaceID: "workspace1",
					PromptKey:   "key1",
					Version:     "1.0",
					PromptTemplate: &entity.PromptTemplate{
						TemplateType: entity.TemplateTypeJinja2,
						Messages: []*entity.Message{
							{
								Role:    entity.RoleSystem,
								Content: &content,
							},
						},
						VariableDefs: []*entity.VariableDef{
							{
								Key:  "users",
								Type: entity.VariableTypeArrayObject,
							},
						},
					},
				}

				// Create a large array of users
				users := make([]map[string]any, 100)
				for i := 0; i < 100; i++ {
					users[i] = map[string]any{
						"name": fmt.Sprintf("User%d", i+1),
						"age":  20 + i%50,
					}
				}
				variables := map[string]any{"users": users}

				messages, err := provider.PromptFormat(ctx, prompt, variables, PromptFormatOptions{})
				So(err, ShouldBeNil)
				So(messages, ShouldNotBeNil)
				So(len(messages), ShouldEqual, 1)
				So(*messages[0].Content, ShouldContainSubstring, "User1 (20)")
				So(*messages[0].Content, ShouldContainSubstring, "User100 (69)")
			})

			Convey("Empty variables with Jinja2", func() {
				content := "Hello {% if name %}{{ name }}{% else %}Guest{% endif %}"
				prompt := &entity.Prompt{
					WorkspaceID: "workspace1",
					PromptKey:   "key1",
					Version:     "1.0",
					PromptTemplate: &entity.PromptTemplate{
						TemplateType: entity.TemplateTypeJinja2,
						Messages: []*entity.Message{
							{
								Role:    entity.RoleSystem,
								Content: &content,
							},
						},
						VariableDefs: []*entity.VariableDef{
							{
								Key:  "name",
								Type: entity.VariableTypeString,
							},
						},
					},
				}
				variables := map[string]any{} // No name provided

				messages, err := provider.PromptFormat(ctx, prompt, variables, PromptFormatOptions{})
				So(err, ShouldBeNil)
				So(messages, ShouldNotBeNil)
				So(len(messages), ShouldEqual, 1)
				So(*messages[0].Content, ShouldEqual, "Hello Guest")
			})

			Convey("Special characters in Jinja2 template", func() {
				content := `{{ message|replace("'", "''") }}`
				prompt := &entity.Prompt{
					WorkspaceID: "workspace1",
					PromptKey:   "key1",
					Version:     "1.0",
					PromptTemplate: &entity.PromptTemplate{
						TemplateType: entity.TemplateTypeJinja2,
						Messages: []*entity.Message{
							{
								Role:    entity.RoleSystem,
								Content: &content,
							},
						},
						VariableDefs: []*entity.VariableDef{
							{
								Key:  "message",
								Type: entity.VariableTypeString,
							},
						},
					},
				}
				variables := map[string]any{"message": "It's a test"}

				messages, err := provider.PromptFormat(ctx, prompt, variables, PromptFormatOptions{})
				So(err, ShouldBeNil)
				So(messages, ShouldNotBeNil)
				So(len(messages), ShouldEqual, 1)
				So(*messages[0].Content, ShouldEqual, "It''s a test")
			})
		})
	})
}
