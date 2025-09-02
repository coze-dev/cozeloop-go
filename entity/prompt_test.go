// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package entity

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/coze-dev/cozeloop-go/internal/util"
)

func TestPromptDeepCopy(t *testing.T) {
	Convey("Test Prompt DeepCopy method", t, func() {
		Convey("When input is nil", func() {
			var p *Prompt
			copied := p.DeepCopy()
			So(copied, ShouldBeNil)
		})

		Convey("When input is not nil", func() {
			// Create a test prompt with all fields populated
			temperature := 0.7
			maxTokens := int32(100)
			topK := int32(5)
			topP := 0.9
			freqPenalty := 0.5
			presPenalty := 0.2
			jsonMode := true
			description := "test description"
			parameters := `{"type":"object"}`
			content := "test content"

			p := &Prompt{
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
					Temperature:      &temperature,
					MaxTokens:        &maxTokens,
					TopK:             &topK,
					TopP:             &topP,
					FrequencyPenalty: &freqPenalty,
					PresencePenalty:  &presPenalty,
					JSONMode:         &jsonMode,
				},
			}

			copied := p.DeepCopy()

			// Check that all fields are properly copied
			So(copied, ShouldNotBeNil)
			So(copied.WorkspaceID, ShouldEqual, p.WorkspaceID)
			So(copied.PromptKey, ShouldEqual, p.PromptKey)
			So(copied.Version, ShouldEqual, p.Version)

			// Verify deep copy by modifying original and checking copied remains unchanged
			p.WorkspaceID = "new_workspace"
			So(copied.WorkspaceID, ShouldEqual, "workspace1")

			// Verify that child objects are correctly deep copied
			So(copied.PromptTemplate, ShouldNotBeNil)
			So(copied.PromptTemplate.TemplateType, ShouldEqual, p.PromptTemplate.TemplateType)

			// Verify messages are copied
			So(len(copied.PromptTemplate.Messages), ShouldEqual, 1)
			So(copied.PromptTemplate.Messages[0].Role, ShouldEqual, RoleSystem)
			So(*copied.PromptTemplate.Messages[0].Content, ShouldEqual, "test content")

			// Modify original message content and verify copied remains unchanged
			newContent := "modified content"
			*p.PromptTemplate.Messages[0].Content = newContent
			So(*copied.PromptTemplate.Messages[0].Content, ShouldEqual, "test content")

			// Verify that modifying original does not affect copied
			p.PromptTemplate.TemplateType = "modified"
			p.PromptTemplate.VariableDefs[0].Key = "modified_key"
			p.PromptTemplate.VariableDefs[0].Desc = "modified_desc"
			p.PromptTemplate.VariableDefs[0].Type = VariableTypePlaceholder

			So(copied.PromptTemplate.TemplateType, ShouldEqual, TemplateTypeNormal)
			So(copied.PromptTemplate.VariableDefs[0].Key, ShouldEqual, "var1")
			So(copied.PromptTemplate.VariableDefs[0].Desc, ShouldEqual, "desc1")
			So(copied.PromptTemplate.VariableDefs[0].Type, ShouldEqual, VariableTypeString)
		})
	})
}

func TestPromptTemplateDeepCopy(t *testing.T) {
	Convey("Test PromptTemplate DeepCopy method", t, func() {
		Convey("When input is nil", func() {
			var pt *PromptTemplate
			copied := pt.DeepCopy()
			So(copied, ShouldBeNil)
		})

		Convey("When input is not nil", func() {
			content := "test content"
			pt := &PromptTemplate{
				TemplateType: TemplateTypeNormal,
				Messages: []*Message{
					{
						Role:    RoleUser,
						Content: &content,
					},
				},
				VariableDefs: []*VariableDef{
					{
						Key:  "var1",
						Desc: "desc1",
						Type: VariableTypePlaceholder,
					},
				},
			}

			copied := pt.DeepCopy()

			// Check that fields are properly copied
			So(copied, ShouldNotBeNil)
			So(copied.TemplateType, ShouldEqual, pt.TemplateType)

			// Check messages are deep copied
			So(len(copied.Messages), ShouldEqual, 1)
			So(copied.Messages[0].Role, ShouldEqual, RoleUser)
			So(*copied.Messages[0].Content, ShouldEqual, content)

			// Check VariableDefs are deep copied
			So(len(copied.VariableDefs), ShouldEqual, 1)
			So(copied.VariableDefs[0].Key, ShouldEqual, "var1")
			So(copied.VariableDefs[0].Type, ShouldEqual, VariableTypePlaceholder)

			// Verify deep copy by modifying original
			pt.TemplateType = "changed"
			pt.VariableDefs[0].Key = "changed"
			*pt.Messages[0].Content = "changed"

			So(copied.TemplateType, ShouldEqual, TemplateTypeNormal)
			So(copied.VariableDefs[0].Key, ShouldEqual, "var1")
			So(*copied.Messages[0].Content, ShouldEqual, "test content")
		})
	})
}

func TestMessageDeepCopy(t *testing.T) {
	Convey("Test Message DeepCopy method", t, func() {
		Convey("When input is nil", func() {
			var m *Message
			copied := m.DeepCopy()
			So(copied, ShouldBeNil)
		})

		Convey("When input has content", func() {
			content := "test content"
			m := &Message{
				Role:    RoleAssistant,
				Content: &content,
			}

			copied := m.DeepCopy()

			So(copied, ShouldNotBeNil)
			So(copied.Role, ShouldEqual, RoleAssistant)
			So(*copied.Content, ShouldEqual, content)

			// Verify deep copy by modifying original
			*m.Content = "changed"
			m.Role = RoleSystem

			So(copied.Role, ShouldEqual, RoleAssistant)
			So(*copied.Content, ShouldEqual, "test content")
		})

		Convey("When input has nil content", func() {
			m := &Message{
				Role:    RoleTool,
				Content: nil,
			}

			copied := m.DeepCopy()

			So(copied, ShouldNotBeNil)
			So(copied.Role, ShouldEqual, RoleTool)
			So(copied.Content, ShouldBeNil)
		})

		Convey("Test message with Role and Parts", func() {
			url := "http://example.com/image.png"
			parts := []*ContentPart{
				{
					Type: "text",
					Text: util.Ptr("Hello"),
				},
				{
					Type:     "image",
					ImageURL: &url,
				},
			}
			msg := &Message{
				Role:  "user",
				Parts: parts,
			}
			copied := msg.DeepCopy()
			So(copied, ShouldNotBeNil)
			So(copied.Role, ShouldEqual, msg.Role)
			So(copied.Content, ShouldBeNil)
			So(copied.Parts, ShouldNotBeNil)
			So(len(copied.Parts), ShouldEqual, len(msg.Parts))
			for i, part := range copied.Parts {
				So(part.Type, ShouldEqual, msg.Parts[i].Type)
				if part.Text != nil {
					So(*part.Text, ShouldEqual, *msg.Parts[i].Text)
				}
				if part.ImageURL != nil {
					So(part.ImageURL, ShouldEqual, msg.Parts[i].ImageURL)
				}
			}
		})

		Convey("Test message with Role, Content, and Parts", func() {
			content := "Hello, World!"
			url := "http://example.com/image.png"
			parts := []*ContentPart{
				{
					Type: "text",
					Text: util.Ptr("Hello"),
				},
				{
					Type:     "image",
					ImageURL: &url,
				},
			}
			msg := &Message{
				Role:    "user",
				Content: &content,
				Parts:   parts,
			}
			copied := msg.DeepCopy()
			So(copied, ShouldNotBeNil)
			So(copied.Role, ShouldEqual, msg.Role)
			So(copied.Content, ShouldNotBeNil)
			So(*copied.Content, ShouldEqual, *msg.Content)
			So(copied.Parts, ShouldNotBeNil)
			So(len(copied.Parts), ShouldEqual, len(msg.Parts))
			for i, part := range copied.Parts {
				So(part.Type, ShouldEqual, msg.Parts[i].Type)
				if part.Text != nil {
					So(*part.Text, ShouldEqual, *msg.Parts[i].Text)
				}
				if part.ImageURL != nil {
					So(part.ImageURL, ShouldEqual, msg.Parts[i].ImageURL)
				}
			}
		})
	})
}

func TestVariableDefDeepCopy(t *testing.T) {
	Convey("Test VariableDef DeepCopy method", t, func() {
		Convey("When input is nil", func() {
			var v *VariableDef
			copied := v.DeepCopy()
			So(copied, ShouldBeNil)
		})

		Convey("When input is not nil", func() {
			v := &VariableDef{
				Key:  "key1",
				Desc: "description1",
				Type: VariableTypeString,
			}

			copied := v.DeepCopy()

			So(copied, ShouldNotBeNil)
			So(copied.Key, ShouldEqual, "key1")
			So(copied.Desc, ShouldEqual, "description1")
			So(copied.Type, ShouldEqual, VariableTypeString)

			// Verify deep copy by modifying original
			v.Key = "changed"
			v.Type = VariableTypePlaceholder

			So(copied.Key, ShouldEqual, "key1")
			So(copied.Type, ShouldEqual, VariableTypeString)
		})
	})
}

func TestToolDeepCopy(t *testing.T) {
	Convey("Test Tool DeepCopy method", t, func() {
		Convey("When input is nil", func() {
			var t *Tool
			copied := t.DeepCopy()
			So(copied, ShouldBeNil)
		})

		Convey("When input has function", func() {
			description := "function description"
			parameters := `{"type":"object"}`
			t := &Tool{
				Type: ToolTypeFunction,
				Function: &Function{
					Name:        "func1",
					Description: &description,
					Parameters:  &parameters,
				},
			}

			copied := t.DeepCopy()

			So(copied, ShouldNotBeNil)
			So(copied.Type, ShouldEqual, ToolTypeFunction)
			So(copied.Function, ShouldNotBeNil)
			So(copied.Function.Name, ShouldEqual, "func1")
			So(*copied.Function.Description, ShouldEqual, description)
			So(*copied.Function.Parameters, ShouldEqual, parameters)

			// Verify deep copy by modifying original
			t.Type = "modified_type"
			t.Function.Name = "modified_name"
			*t.Function.Description = "modified_description"
			*t.Function.Parameters = "modified_parameters"

			So(copied.Type, ShouldEqual, ToolTypeFunction)
			So(copied.Function.Name, ShouldEqual, "func1")
			So(*copied.Function.Description, ShouldEqual, "function description")
			So(*copied.Function.Parameters, ShouldEqual, `{"type":"object"}`)
		})

		Convey("When input has nil function", func() {
			t := &Tool{
				Type:     ToolTypeFunction,
				Function: nil,
			}

			copied := t.DeepCopy()

			So(copied, ShouldNotBeNil)
			So(copied.Type, ShouldEqual, ToolTypeFunction)
			So(copied.Function, ShouldBeNil)
		})
	})
}

func TestFunctionDeepCopy(t *testing.T) {
	Convey("Test Function DeepCopy method", t, func() {
		Convey("When input is nil", func() {
			var f *Function
			copied := f.DeepCopy()
			So(copied, ShouldBeNil)
		})

		Convey("When input has all fields", func() {
			description := "function description"
			parameters := `{"type":"object"}`
			f := &Function{
				Name:        "func1",
				Description: &description,
				Parameters:  &parameters,
			}

			copied := f.DeepCopy()

			So(copied, ShouldNotBeNil)
			So(copied.Name, ShouldEqual, "func1")
			So(*copied.Description, ShouldEqual, description)
			So(*copied.Parameters, ShouldEqual, parameters)

			// Verify deep copy by modifying original
			f.Name = "changed"
			*f.Description = "changed"

			So(copied.Name, ShouldEqual, "func1")
			So(*copied.Description, ShouldEqual, "function description")
		})

		Convey("When input has nil optional fields", func() {
			f := &Function{
				Name:        "func1",
				Description: nil,
				Parameters:  nil,
			}

			copied := f.DeepCopy()

			So(copied, ShouldNotBeNil)
			So(copied.Name, ShouldEqual, "func1")
			So(copied.Description, ShouldBeNil)
			So(copied.Parameters, ShouldBeNil)
		})
	})
}

func TestToolCallConfigDeepCopy(t *testing.T) {
	Convey("Test ToolCallConfig DeepCopy method", t, func() {
		Convey("When input is nil", func() {
			var tc *ToolCallConfig
			copied := tc.DeepCopy()
			So(copied, ShouldBeNil)
		})

		Convey("When input is not nil", func() {
			tc := &ToolCallConfig{
				ToolChoice: ToolChoiceTypeAuto,
			}

			copied := tc.DeepCopy()

			So(copied, ShouldNotBeNil)
			So(copied.ToolChoice, ShouldEqual, ToolChoiceTypeAuto)

			// Verify deep copy by modifying original
			tc.ToolChoice = ToolChoiceTypeNone

			So(copied.ToolChoice, ShouldEqual, ToolChoiceTypeAuto)
		})
	})
}

func TestLLMConfigDeepCopy(t *testing.T) {
	Convey("Test LLMConfig DeepCopy method", t, func() {
		Convey("When input is nil", func() {
			var mc *LLMConfig
			copied := mc.DeepCopy()
			So(copied, ShouldBeNil)
		})

		Convey("When input has all fields", func() {
			temperature := 0.7
			maxTokens := int32(100)
			topK := int32(5)
			topP := 0.9
			freqPenalty := 0.5
			presPenalty := 0.2
			jsonMode := true

			mc := &LLMConfig{
				Temperature:      &temperature,
				MaxTokens:        &maxTokens,
				TopK:             &topK,
				TopP:             &topP,
				FrequencyPenalty: &freqPenalty,
				PresencePenalty:  &presPenalty,
				JSONMode:         &jsonMode,
			}

			copied := mc.DeepCopy()

			So(copied, ShouldNotBeNil)
			So(*copied.Temperature, ShouldEqual, temperature)
			So(*copied.MaxTokens, ShouldEqual, maxTokens)
			So(*copied.TopK, ShouldEqual, topK)
			So(*copied.TopP, ShouldEqual, topP)
			So(*copied.FrequencyPenalty, ShouldEqual, freqPenalty)
			So(*copied.PresencePenalty, ShouldEqual, presPenalty)
			So(*copied.JSONMode, ShouldEqual, jsonMode)

			// Verify deep copy by modifying original
			newTemp := 0.1
			newMaxTokens := int32(200)
			*mc.Temperature = newTemp
			*mc.MaxTokens = newMaxTokens
			*mc.TopK = 10
			*mc.TopP = 0.1
			*mc.FrequencyPenalty = 0.1
			*mc.PresencePenalty = 0.1
			*mc.JSONMode = false

			So(*copied.Temperature, ShouldEqual, 0.7)
			So(*copied.MaxTokens, ShouldEqual, 100)
			So(*copied.TopK, ShouldEqual, 5)
			So(*copied.TopP, ShouldEqual, 0.9)
			So(*copied.FrequencyPenalty, ShouldEqual, 0.5)
			So(*copied.PresencePenalty, ShouldEqual, 0.2)
			So(*copied.JSONMode, ShouldEqual, true)
		})

		Convey("When input has nil fields", func() {
			mc := &LLMConfig{
				Temperature:      nil,
				MaxTokens:        nil,
				TopK:             nil,
				TopP:             nil,
				FrequencyPenalty: nil,
				PresencePenalty:  nil,
				JSONMode:         nil,
			}

			copied := mc.DeepCopy()

			So(copied, ShouldNotBeNil)
			So(copied.Temperature, ShouldBeNil)
			So(copied.MaxTokens, ShouldBeNil)
			So(copied.TopK, ShouldBeNil)
			So(copied.TopP, ShouldBeNil)
			So(copied.FrequencyPenalty, ShouldBeNil)
			So(copied.PresencePenalty, ShouldBeNil)
			So(copied.JSONMode, ShouldBeNil)
		})
	})
}

func TestHelperFunctions(t *testing.T) {
	Convey("Test deepCopyMessages function", t, func() {
		Convey("When input is nil", func() {
			result := deepCopyMessages(nil)
			So(result, ShouldBeNil)
		})

		Convey("When input is not nil", func() {
			content1 := "content1"
			content2 := "content2"
			messages := []*Message{
				{Role: RoleSystem, Content: &content1},
				{Role: RoleUser, Content: &content2},
			}

			copied := deepCopyMessages(messages)

			So(len(copied), ShouldEqual, 2)
			So(copied[0].Role, ShouldEqual, RoleSystem)
			So(*copied[0].Content, ShouldEqual, content1)
			So(copied[1].Role, ShouldEqual, RoleUser)
			So(*copied[1].Content, ShouldEqual, content2)

			// Verify deep copy by modifying original
			*messages[0].Content = "changed"

			So(*copied[0].Content, ShouldEqual, "content1")
		})
	})

	Convey("Test deepCopyVariableDefs function", t, func() {
		Convey("When input is nil", func() {
			result := deepCopyVariableDefs(nil)
			So(result, ShouldBeNil)
		})

		Convey("When input is not nil", func() {
			defs := []*VariableDef{
				{Key: "key1", Desc: "desc1", Type: VariableTypeString},
				{Key: "key2", Desc: "desc2", Type: VariableTypePlaceholder},
			}

			copied := deepCopyVariableDefs(defs)

			So(len(copied), ShouldEqual, 2)
			So(copied[0].Key, ShouldEqual, "key1")
			So(copied[0].Type, ShouldEqual, VariableTypeString)
			So(copied[1].Key, ShouldEqual, "key2")
			So(copied[1].Type, ShouldEqual, VariableTypePlaceholder)

			// Verify deep copy by modifying original
			defs[0].Key = "changed"

			So(copied[0].Key, ShouldEqual, "key1")
		})
	})

	Convey("Test deepCopyTools function", t, func() {
		Convey("When input is nil", func() {
			result := deepCopyTools(nil)
			So(result, ShouldBeNil)
		})

		Convey("When input is not nil", func() {
			desc := "description"
			tools := []*Tool{
				{Type: ToolTypeFunction, Function: &Function{Name: "func1", Description: &desc}},
				{Type: ToolTypeFunction, Function: nil},
			}

			copied := deepCopyTools(tools)

			So(len(copied), ShouldEqual, 2)
			So(copied[0].Type, ShouldEqual, ToolTypeFunction)
			So(copied[0].Function.Name, ShouldEqual, "func1")
			So(*copied[0].Function.Description, ShouldEqual, desc)
			So(copied[1].Type, ShouldEqual, ToolTypeFunction)
			So(copied[1].Function, ShouldBeNil)

			// Verify deep copy by modifying original
			tools[0].Type = "changed"
			tools[0].Function.Name = "changed"

			So(copied[0].Type, ShouldEqual, ToolTypeFunction)
			So(copied[0].Function.Name, ShouldEqual, "func1")
		})
	})
}
