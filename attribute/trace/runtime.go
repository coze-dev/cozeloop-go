// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

type Runtime struct {
	Language string `json:"language"`          // from enum VLang in span_value.go
	Library  string `json:"library,omitempty"` // integration library, from enum VLib in span_value.go
	Scene    string `json:"scene,omitempty"`   // usage scene, from enum VScene in span_value.go

	// Dependency Versions.
	LibraryVersion string `json:"library_version,omitempty"`
	LoopSDKVersion string `json:"loop_sdk_version,omitempty"`
}
