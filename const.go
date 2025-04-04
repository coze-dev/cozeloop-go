// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package cozeloop

import (
	"time"

	"github.com/coze-dev/cozeloop-go/internal/consts"
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

	// default values for loop client
	DefaultPromptCacheMaxCount        = 100
	DefaultPromptCacheRefreshInterval = 10 * time.Minute
	DefaultHttpClientTimeout          = 300 * time.Second
)
