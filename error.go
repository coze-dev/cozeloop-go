// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package cozeloop

import (
	"github.com/coze-dev/cozeloop-go/internal/consts"
)

var (
	ErrInvalidParam  = consts.ErrInvalidParam
	ErrHeaderParent  = consts.ErrHeaderParent
	ErrRemoteService = consts.ErrRemoteService

	ErrAuthInfoRequired = consts.ErrAuthInfoRequired
	ErrParsePrivateKey  = consts.ErrParsePrivateKey
)

type (
	AuthError          = consts.AuthError
	RemoteServiceError = consts.RemoteServiceError
)
