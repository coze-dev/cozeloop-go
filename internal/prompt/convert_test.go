// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package prompt

import (
	"testing"

	"github.com/coze-dev/cozeloop-go/entity"
	. "github.com/smartystreets/goconvey/convey"
)

func TestToModelPrompt(t *testing.T) {
	Convey("Test toModelPrompt", t, func() {
		Convey("When input is nil", func() {
			result := toModelPrompt(nil)
			So(result, ShouldBeNil)
		})

		Convey("When input is complete", func() {
			content := "test content"
			description := "test description"
			parameters := `{"type":"object"}`
			temperature := 0.7
			maxTokens := int32(100)
			input := &Prompt{
				WorkspaceID: "workspace1",
				PromptKey:   "key1",
				Version:     "1.0",
				PromptTemplate: &PromptTemplate{
					TemplateType: TemplateTypeNormal,
					Messages: []*Message{
						{
							Role:    RoleSystem,
							Content: &content,
						},
					},
					VariableDefs: []*VariableDef{
						{
							Key:  "var1",
							Desc: "desc1",
							Type: VariableTypeString,
						},
					},
				},
				Tools: []*Tool{
					{
						Type: ToolTypeFunction,
						Function: &Function{
							Name:        "func1",
							Description: &description,
							Parameters:  &parameters,
						},
					},
				},
				ToolCallConfig: &ToolCallConfig{
					ToolChoice: ToolChoiceTypeAuto,
				},
				LLMConfig: &LLMConfig{
					Temperature: &temperature,
					MaxTokens:   &maxTokens,
				},
			}

			result := toModelPrompt(input)
			So(result, ShouldNotBeNil)
			So(result.WorkspaceID, ShouldEqual, "workspace1")
			So(result.PromptKey, ShouldEqual, "key1")
			So(result.Version, ShouldEqual, "1.0")

			// Check PromptTemplate
			So(result.PromptTemplate, ShouldNotBeNil)
			So(result.PromptTemplate.TemplateType, ShouldEqual, entity.TemplateTypeNormal)
			So(len(result.PromptTemplate.Messages), ShouldEqual, 1)
			So(result.PromptTemplate.Messages[0].Role, ShouldEqual, entity.RoleSystem)
			So(*result.PromptTemplate.Messages[0].Content, ShouldEqual, content)

			// Check Tools
			So(len(result.Tools), ShouldEqual, 1)
			So(result.Tools[0].Type, ShouldEqual, entity.ToolTypeFunction)
			So(result.Tools[0].Function.Name, ShouldEqual, "func1")

			// Check Configs
			So(result.ToolCallConfig.ToolChoice, ShouldEqual, entity.ToolChoiceTypeAuto)
			So(*result.LLMConfig.Temperature, ShouldEqual, temperature)
			So(*result.LLMConfig.MaxTokens, ShouldEqual, maxTokens)
		})
	})
}

func TestToModelRole(t *testing.T) {
	Convey("Test toModelRole", t, func() {
		Convey("Test all role types", func() {
			cases := []struct {
				input    Role
				expected entity.Role
			}{
				{RoleSystem, entity.RoleSystem},
				{RoleUser, entity.RoleUser},
				{RoleAssistant, entity.RoleAssistant},
				{RoleTool, entity.RoleTool},
				{RolePlaceholder, entity.RolePlaceholder},
				{"unknown", entity.RoleUser}, // default case
			}

			for _, c := range cases {
				result := toModelRole(c.input)
				So(result, ShouldEqual, c.expected)
			}
		})
	})
}

func TestToSpanPromptInput(t *testing.T) {
	Convey("Test toSpanPromptInput", t, func() {
		Convey("When input is complete", func() {
			content := "test content"
			messages := []*entity.Message{
				{
					Role:    entity.RoleSystem,
					Content: &content,
				},
			}
			arguments := map[string]any{
				"key1": "value1",
				"key2": 123,
			}

			result := toSpanPromptInput(messages, arguments)
			So(result, ShouldNotBeNil)
			So(len(result.Templates), ShouldEqual, 1)
			So(result.Templates[0].Role, ShouldEqual, "system")
			So(result.Templates[0].Content, ShouldEqual, content)
			So(len(result.Arguments), ShouldEqual, 2)
		})

		Convey("When messages contain nil", func() {
			messages := []*entity.Message{nil}
			result := toSpanPromptInput(messages, nil)
			So(result, ShouldNotBeNil)
			So(len(result.Templates), ShouldEqual, 1)
			So(result.Templates[0], ShouldBeNil)
		})
	})
}

func TestToModelToolType(t *testing.T) {
	Convey("Test toModelToolType", t, func() {
		Convey("Test all tool types", func() {
			cases := []struct {
				input    ToolType
				expected entity.ToolType
			}{
				{ToolTypeFunction, entity.ToolTypeFunction},
				{"unknown", entity.ToolTypeFunction}, // default case
			}

			for _, c := range cases {
				result := toModelToolType(c.input)
				So(result, ShouldEqual, c.expected)
			}
		})
	})
}

func TestToModelVariableType(t *testing.T) {
	Convey("Test toModelVariableType", t, func() {
		Convey("Test all variable types", func() {
			cases := []struct {
				input    VariableType
				expected entity.VariableType
			}{
				{VariableTypeString, entity.VariableTypeString},
				{VariableTypePlaceholder, entity.VariableTypePlaceholder},
				{"unknown", entity.VariableTypeString}, // default case
			}

			for _, c := range cases {
				result := toModelVariableType(c.input)
				So(result, ShouldEqual, c.expected)
			}
		})
	})
}

func TestToModelToolChoiceType(t *testing.T) {
	Convey("Test toModelToolChoiceType", t, func() {
		Convey("Test all tool choice types", func() {
			cases := []struct {
				input    ToolChoiceType
				expected entity.ToolChoiceType
			}{
				{ToolChoiceTypeAuto, entity.ToolChoiceTypeAuto},
				{ToolChoiceTypeNone, entity.ToolChoiceTypeNone},
				{"unknown", entity.ToolChoiceTypeAuto}, // default case
			}

			for _, c := range cases {
				result := toModelToolChoiceType(c.input)
				So(result, ShouldEqual, c.expected)
			}
		})
	})
}

func TestToSpanMessage(t *testing.T) {
	Convey("Test toSpanMessage", t, func() {
		Convey("When input is nil", func() {
			result := toSpanMessage(nil)
			So(result, ShouldBeNil)
		})

		Convey("When input is complete", func() {
			content := "test content"
			input := &entity.Message{
				Role:    entity.RoleSystem,
				Content: &content,
			}

			result := toSpanMessage(input)
			So(result, ShouldNotBeNil)
			So(result.Role, ShouldEqual, string(entity.RoleSystem))
			So(result.Content, ShouldEqual, content)
		})

		Convey("When content is nil", func() {
			input := &entity.Message{
				Role:    entity.RoleSystem,
				Content: nil,
			}

			result := toSpanMessage(input)
			So(result, ShouldNotBeNil)
			So(result.Role, ShouldEqual, string(entity.RoleSystem))
			So(result.Content, ShouldEqual, "")
		})
	})
}
