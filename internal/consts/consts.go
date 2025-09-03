// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package consts

import (
	"time"
)

// default values for loop client
const (
	// ComBaseURL = "https://api.coze.com"
	CnBaseURL                         = "https://api.coze.cn"
	DefaultOAuthRefreshTTL            = 900 * time.Second
	OAuthRefreshAdvanceTime           = 60 * time.Second
	DefaultPromptCacheMaxCount        = 100
	DefaultPromptCacheRefreshInterval = 1 * time.Minute
	DefaultTimeout                    = 3 * time.Second
	DefaultUploadTimeout              = 30 * time.Second
)

const (
	LogIDHeader     = "x-tt-logid"
	AuthorizeHeader = "Authorization"
)

// Define various boundary size.
const (
	MaxTagKvCountInOneSpan = 50

	MaxBytesOfOneTagValueOfInputOutput = 1 * 1024 * 1024
	TextTruncateCharLength             = 1000

	MaxBytesOfOneTagValueDefault = 1024
	MaxBytesOfOneTagKeyDefault   = 1024
)

const (
	StatusCodeErrorDefault int = -1
)

const (
	GlobalTraceVersion = 0
)

const (
	Equal = "="
	Comma = ","
)

// On the basis of W3C, the "loop" prefix is added to avoid conflicts with other traces that use W3C.
const (
	TraceContextHeaderParent  = "X-Cozeloop-Traceparent"
	TraceContextHeaderBaggage = "X-Cozeloop-Tracestate"
)

const (
	TracePromptHubSpanName              = "PromptHub"
	TracePromptTemplateSpanName         = "PromptTemplate"
	TracePromptExecuteSpanName          = "PromptExecute"
	TracePromptExecuteStreamingSpanName = "PromptExecuteStreaming"
)

const (
	PromptNormalTemplateStartTag = "{{"
	PromptNormalTemplateEndTag   = "}}"
)
