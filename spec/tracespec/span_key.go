// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package tracespec

// Tags for model-type span.
const (
	CallOptions       = "call_options"       // Used to identify option for model, like temperature, etc. Recommend use ModelCallOption struct.
	Stream            = "stream"             // Used to identify whether it is a streaming output.
	ReasoningTokens   = "reasoning_tokens"   // The token usage during the reasoning process.
	ReasoningDuration = "reasoning_duration" // The duration during the reasoning process. The unit is microseconds.
)

// Tags for retriever-type span
const (
	RetrieverProvider = "retriever_provider" // Data retrieval providers, such as Elasticsearch (ES), VikingDB, etc.
	VikingDBName      = "vikingdb_name"      // When using VikingDB to provide retrieval capabilities, db name.
	VikingDBRegion    = "vikingdb_region"    // When using VikingDB to provide retrieval capabilities, db region.
	ESName            = "es_name"            // When using ES to provide retrieval capabilities, es name.
	ESIndex           = "es_index"           // When using ES to provide retrieval capabilities, es index.
	ESCluster         = "es_cluster"         // When using ES to provide retrieval capabilities, es cluster.
)

// Tags for prompt-type span.
const (
	PromptProvider = "prompt_provider" // Prompt providers, such as CozeLoop, Langsmith, etc.
	PromptKey      = "prompt_key"
	PromptVersion  = "prompt_version"
)

// Internal experimental field.
// Not recommended for use unless you know what you're doing. Instead, use the corresponding Set method.
const (
	SpanType = "span_type"
	Input    = "input"
	Output   = "output"
	Error    = "error"
	Runtime_ = "runtime"

	ModelProvider       = "model_provider"
	ModelName           = "model_name"
	InputTokens         = "input_tokens"
	OutputTokens        = "output_tokens"
	Tokens              = "tokens"
	ModelPlatform       = "model_platform"
	ModelIdentification = "model_identification"
	TokenUsageBackup    = "token_usage_backup"
	LatencyFirstResp    = "latency_first_resp"

	CallType = "call_type"
)
