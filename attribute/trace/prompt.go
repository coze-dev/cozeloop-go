// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

// PromptInput is the input of prompt span, for tag key: input
type PromptInput struct {
	Templates []*ModelMessage   `json:"templates"`
	Arguments []*PromptArgument `json:"arguments"`
}

type PromptArgument struct {
	Key    string `json:"key"`
	Value  any    `json:"value"`
	Source string `json:"source"` // from enum VPromptArgSource in span_value.go
}

// PromptOutput is the output of prompt span, for tag key: output
type PromptOutput struct {
	Prompts []*ModelMessage `json:"prompts"`
}
