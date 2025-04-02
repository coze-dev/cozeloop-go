// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

type RetrieverInput struct {
	Query string `json:"query,omitempty"`
}

type RetrieverOutput struct {
	Documents []*RetrieverDocument `json:"documents,omitempty"`
}

type RetrieverDocument struct {
	ID      string    `json:"id,omitempty"`
	Index   string    `json:"index,omitempty"`
	Content string    `json:"content"`
	Vector  []float64 `json:"vector,omitempty"`
	Score   float64   `json:"score"`
}

type RetrieverCallOption struct {
	TopK     int64    `json:"top_k,omitempty"`
	MinScore *float64 `json:"min_score,omitempty"`
	Filter   string   `json:"filter,omitempty"`
}
