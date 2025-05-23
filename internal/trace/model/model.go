// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package model

type ObjectStorage struct {
	InputTosKey  string        `json:"input_tos_key,omitempty"`  // The key for reporting long input data
	OutputTosKey string        `json:"output_tos_key,omitempty"` // The key for reporting long output data
	Attachments  []*Attachment // attachments in input or output
}
type Attachment struct {
	Field  string `json:"field,omitempty"`
	Name   string `json:"name,omitempty"`
	Type   string `json:"type,omitempty"` // text, image, file
	TosKey string `json:"tos_key,omitempty"`
}
