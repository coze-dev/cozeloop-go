// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package util

import (
	"context"
	"runtime"

	"github.com/coze-dev/cozeloop-go/internal/logger"
)

// GoSafe Safely start a goroutine, which will automatically recover from panics and print stack information.
func GoSafe(ctx context.Context, fn func()) {
	go func() {
		defer func() {
			if e := recover(); e != nil {
				const size = 64 << 10
				buf := make([]byte, size)
				buf = buf[:runtime.Stack(buf, false)]
				logger.CtxErrorf(ctx, "goroutine panic: %s: %s", e, buf)
			}
		}()
		fn()
	}()
}
