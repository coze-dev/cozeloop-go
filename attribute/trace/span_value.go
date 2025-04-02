// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

// SpanType tag builtin values
const (
	VPromptHubSpanType      = "prompt_hub"
	VPromptTemplateSpanType = "prompt"
	VModelSpanType          = "model"
	VRetrieverSpanType      = "retriever"
	VToolSpanType           = "tool"
)

const (
	VErrDefault = -1 // Default StatusCode for errors.
)

// Tag values for model messages.
const (
	VRoleUser      = "user"
	VRoleSystem    = "system"
	VRoleAssistant = "assistant"
	VRoleTool      = "tool"

	// VToolChoiceNone Reference: https://platform.openai.com/docs/api-reference/chat/create#chat-create-messages
	VToolChoiceNone     = "none"     // Means the model will not call any tool and instead generates a message.
	VToolChoiceAuto     = "auto"     // Means the model can pick between generating a message or calling one or more tools.
	VToolChoiceRequired = "required" // Means the model must call one or more tools.
	VToolChoiceFunction = "function" // Forces the model to call that tool.
)

// Tag values for runtime tags.
const (
	VLangGo         = "go"
	VLangPython     = "python"
	VLangTypeScript = "ts"

	VLibEino      = "eino"
	VLibLangChain = "langchain"

	VSceneCustom         = "custom"          // user custom, it has the same meaning as blank.
	VScenePromptHub      = "prompt_hub"      // get_prompt
	VScenePromptTemplate = "prompt_template" // prompt_template
)

// Tag values for prompt input.
const (
	VPromptArgSourceInput   = "input"
	VPromptArgSourcePartial = "partial"
)
