// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package loop

import (
	"github.com/coze-dev/cozeloop-go/internal/logger"
)

// Logger interface for logging
type Logger = logger.Logger

// LogLevel log level
type LogLevel = logger.LogLevel

const (
	LogLevelDebug LogLevel = logger.LogLevelDebug
	LogLevelInfo           = logger.LogLevelInfo
	LogLevelWarn           = logger.LogLevelWarn
	LogLevelError          = logger.LogLevelError
	LogLevelFatal          = logger.LogLevelFatal
)

// SetLogger set default logger. By default, the logger is set to stderr.
// Note that this method is not thread-safe. Should be called before any other method.
func SetLogger(l Logger) {
	logger.SetLogger(l)
}

// SetLogLevel set log level. By default, the log level is set to Info.
// Note that this method is not thread-safe. Should be called before any other method.
func SetLogLevel(level LogLevel) {
	logger.SetLogLevel(level)
}

// GetLogger get default logger. By default, the logger is set to stderr.
func GetLogger() Logger {
	return logger.GetLogger()
}
