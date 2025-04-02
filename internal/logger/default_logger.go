// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package logger

import (
	"context"
	"fmt"
	"log"
)

type stdLogger struct {
	log *log.Logger
}

func (l stdLogger) CtxDebugf(ctx context.Context, format string, v ...interface{}) {
	l.ctxLogf(ctx, LogLevelDebug, format, v...)
}

func (l stdLogger) CtxInfof(ctx context.Context, format string, v ...interface{}) {
	l.ctxLogf(ctx, LogLevelInfo, format, v...)
}

func (l stdLogger) CtxWarnf(ctx context.Context, format string, v ...interface{}) {
	l.ctxLogf(ctx, LogLevelWarn, format, v...)
}

func (l stdLogger) CtxErrorf(ctx context.Context, format string, v ...interface{}) {
	l.ctxLogf(ctx, LogLevelError, format, v...)
}

func (l stdLogger) CtxFatalf(ctx context.Context, format string, v ...interface{}) {
	l.ctxLogf(ctx, LogLevelFatal, format, v...)
}

func (l stdLogger) ctxLogf(ctx context.Context, level LogLevel, format string, v ...interface{}) {
	msg := level.toString()
	msg += fmt.Sprintf(format, v...)
	_ = l.log.Output(4, msg)
}
