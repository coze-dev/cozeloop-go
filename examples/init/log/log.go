// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"

	"github.com/coze-dev/cozeloop-go"
)

// A simple example to change logger and log level.
func main() {
	// Set the following environment variables first.
	// COZELOOP_WORKSPACE_ID=your workspace id
	// COZELOOP_JWT_OAUTH_CLIENT_ID=your client id
	// COZELOOP_JWT_OAUTH_PRIVATE_KEY=your private key
	// COZELOOP_JWT_OAUTH_PUBLIC_KEY_ID=your public key id
	ctx := context.Background()

	// You can change log level. Default log level is LogLevelWarn.
	cozeloop.SetLogLevel(cozeloop.LogLevelDebug)
	// You can also use you own logger for loop sdk.
	cozeloop.SetLogger(&CustomLogger{})

	ctx, span := cozeloop.StartSpan(ctx, "your span name", "custom")
	span.Finish(ctx)

	// Remember to close the client when program exits. If client is not closed, traces may be lost.
	cozeloop.Close(ctx)
}

type CustomLogger struct {
}

func (l *CustomLogger) CtxDebugf(ctx context.Context, format string, v ...interface{}) {
	fmt.Printf("[Custom] [DEBUG] "+format+"\n", v...)
}

func (l *CustomLogger) CtxInfof(ctx context.Context, format string, v ...interface{}) {
	fmt.Printf("[Custom] [Info] "+format+"\n", v...)
}

func (l *CustomLogger) CtxWarnf(ctx context.Context, format string, v ...interface{}) {
	fmt.Printf("[Custom] [Warn] "+format+"\n", v...)
}

func (l *CustomLogger) CtxErrorf(ctx context.Context, format string, v ...interface{}) {
	fmt.Printf("[Custom] [Error] "+format+"\n", v...)
}

func (l *CustomLogger) CtxFatalf(ctx context.Context, format string, v ...interface{}) {
	fmt.Printf("[Custom] [Fatal] "+format+"\n", v...)
}
