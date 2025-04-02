// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package logger

import (
	"context"
	"fmt"
	"log"
	"os"
)

var defaultLogger = func() Logger {
	return stdLogger{log: log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)}
}()
var defaultLogLevel = LogLevelWarn

// Logger Interface for logging
type Logger interface {
	CtxDebugf(ctx context.Context, format string, v ...interface{})
	CtxInfof(ctx context.Context, format string, v ...interface{})
	CtxWarnf(ctx context.Context, format string, v ...interface{})
	CtxErrorf(ctx context.Context, format string, v ...interface{})
	CtxFatalf(ctx context.Context, format string, v ...interface{})
}

type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

var logLevelStr = []string{
	"[Debug] [cozeloop] ",
	"[Info] [cozeloop] ",
	"[Warn] [cozeloop] ",
	"[Error] [cozeloop] ",
	"[Fatal] [cozeloop] ",
}

func (lv LogLevel) toString() string {
	if lv >= LogLevelDebug && lv <= LogLevelFatal {
		return logLevelStr[lv]
	}
	return fmt.Sprintf("[?%d] [cozeloop] ", lv)
}

type LevelLogger interface {
	Logger
	SetLevel(level LogLevel)
}

func SetLogger(l Logger) {
	defaultLogger = l
}

func GetLogger() Logger {
	return defaultLogger
}

func SetLogLevel(level LogLevel) {
	defaultLogLevel = level
}

func GetLogLevel() LogLevel {
	return defaultLogLevel
}

func CtxDebugf(ctx context.Context, format string, v ...interface{}) {
	if GetLogLevel() <= LogLevelDebug {
		GetLogger().CtxDebugf(ctx, format, v...)
	}
}

func CtxInfof(ctx context.Context, format string, v ...interface{}) {
	if GetLogLevel() <= LogLevelInfo {
		GetLogger().CtxInfof(ctx, format, v...)
	}
}

func CtxWarnf(ctx context.Context, format string, v ...interface{}) {
	if GetLogLevel() <= LogLevelWarn {
		GetLogger().CtxWarnf(ctx, format, v...)
	}
}

func CtxErrorf(ctx context.Context, format string, v ...interface{}) {
	if GetLogLevel() <= LogLevelError {
		GetLogger().CtxErrorf(ctx, format, v...)
	}
}

func CtxFatalf(ctx context.Context, format string, v ...interface{}) {
	if GetLogLevel() <= LogLevelFatal {
		GetLogger().CtxFatalf(ctx, format, v...)
	}
}
