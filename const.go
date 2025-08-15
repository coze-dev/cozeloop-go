// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package cozeloop

import (
	"github.com/coze-dev/cozeloop-go/internal/consts"
	"github.com/coze-dev/cozeloop-go/internal/trace"
)

const (
	// environment keys for loop client
	EnvApiBaseURL          = "COZELOOP_API_BASE_URL"
	EnvWorkspaceID         = "COZELOOP_WORKSPACE_ID"
	EnvApiToken            = "COZELOOP_API_TOKEN"
	EnvJwtOAuthClientID    = "COZELOOP_JWT_OAUTH_CLIENT_ID"
	EnvJwtOAuthPrivateKey  = "COZELOOP_JWT_OAUTH_PRIVATE_KEY"
	EnvJwtOAuthPublicKeyID = "COZELOOP_JWT_OAUTH_PUBLIC_KEY_ID"

	// ComBaseURL = consts.ComBaseURL
	CnBaseURL = consts.CnBaseURL
)

// SpanFinishEvent finish inner event
type SpanFinishEvent consts.SpanFinishEvent

const (
	SpanFinishEventSpanQueueEntryRate = SpanFinishEvent(consts.SpanFinishEventSpanQueueEntryRate)
	SpanFinishEventFileQueueEntryRate = SpanFinishEvent(consts.SpanFinishEventFileQueueEntryRate)
	SpanFinishEventFlushSpanRate      = SpanFinishEvent(consts.SpanFinishEventFlushSpanRate)
	SpanFinishEventFlushFileRate      = SpanFinishEvent(consts.SpanFinishEventFlushFileRate)
)

type FinishEventInfo consts.FinishEventInfo

type TagTruncateConf trace.TagTruncateConf

type APIBasePath struct {
	TraceSpanUploadPath string
	TraceFileUploadPath string
}

type TraceQueueConf trace.QueueConf
