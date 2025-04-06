// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/coze-dev/cozeloop-go"
)

// A simple example about how to handle loop sdk errors.
func main() {
	// Set the following environment variables first.
	// COZELOOP_WORKSPACE_ID=your workspace id
	// COZELOOP_JWT_OAUTH_CLIENT_ID=your client id
	// COZELOOP_JWT_OAUTH_PRIVATE_KEY=your private key
	// COZELOOP_JWT_OAUTH_PUBLIC_KEY_ID=your public key id
	ctx := context.Background()

	// If you try to get an invalid prompt, you will get a LoopError.
	_, err := cozeloop.GetPrompt(ctx, cozeloop.GetPromptParam{PromptKey: "invalid key"})
	if err != nil {
		// Loop sdk will always return a LoopError like ErrXXX, you can find them in error.go.
		if errors.Is(err, cozeloop.ErrRemoteService) {
			fmt.Printf("Got a loop error.\n")
		}

		// You can use errors.As to unwrap LoopError to RemoteServiceError or AuthError
		var loopError *cozeloop.RemoteServiceError
		if errors.As(err, &loopError) {
			fmt.Printf("Got a RemoteServiceError, error_code: %v, log_id: %v.\n", loopError.ErrCode, loopError.LogID)
		}
	}

	// Considering that tracing is generally not in your main process, in order to simplify the business code,
	// all trace api will never return errors or throw panic. Therefore, the business does not need to handle exceptions.
	ctx, span := cozeloop.StartSpan(ctx, "invalid name", "invalie type")
	span.Finish(ctx)

	// Remember to close the client when program exits. If client is not closed, traces may be lost.
	cozeloop.Close(ctx)
}
