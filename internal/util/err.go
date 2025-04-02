// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package util

import (
	"errors"

	"github.com/coze-dev/cozeloop-go/internal/consts"
)

func GetErrorCode(err error) int {
	if err == nil {
		return 0
	}
	remoteServiceError := &consts.RemoteServiceError{}
	if ok := errors.As(err, &remoteServiceError); ok {
		return remoteServiceError.ErrCode
	}
	return -1
}
