// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/coze-dev/cozeloop-go"
)

// A simple example to init loop client by oauth jwt.
//
// First, you should access https://www.coze.cn/open/oauth/apps and create a new application.
// The specific process can be referred to the document: todo
// You should keep your publicKeyID and privateKey safe to prevent data leakage.
func main() {
	// Set the following environment variables first.
	// COZELOOP_WORKSPACE_ID=your workspace id
	// COZELOOP_JWT_OAUTH_CLIENT_ID=your client id
	// COZELOOP_JWT_OAUTH_PRIVATE_KEY=your private key
	// COZELOOP_JWT_OAUTH_PUBLIC_KEY_ID=your public key id

	// If you needn't any specific configs, you can call any functions without new a LoopClient.
	ctx, span := cozeloop.StartSpan(context.Background(), "first_span", "custom")
	span.Finish(ctx)
	// Remember to close the client when program exits. If client is not closed, traces may be lost.
	cozeloop.Close(ctx)

	// Or you can call NewClient to init a LoopClient if you want to make more custom configs.
	// useNewClient()
}

func useNewClient() {
	ctx := context.Background()
	// IMPORTANT: The client is thread-safe. You should NewClient only once in your program.
	client, err := cozeloop.NewClient(
		// You can also set your oauth info instead of environment variables.
		// loop.WithJWTOAuthClientID("your client id"),
		// loop.WithJWTOAuthPrivateKey("your private key"),
		// loop.WithJWTOAuthPublicKeyID("your public key id"),

		// You can set the workspace ID instead of environment variables.
		// loop.WithWorkspaceID("your workspace id"),

		// You can set the API base URL. Generally, there's no need to use it.
		// loop.WithAPIBaseURL("https://api.coze.cn"),

		// The SDK will communicate with the Loop server. You can set the read timeout for requests.
		// Default value is 3 seconds.
		cozeloop.WithTimeout(time.Second*3),
		// The SDK will upload images or large text to file storage server when necessary.
		// You can set the upload timeout for requests.
		// Default value is 30 seconds.
		cozeloop.WithUploadTimeout(time.Second*30),
		// Or you can set your own http client and make more custom configs.
		cozeloop.WithHTTPClient(http.DefaultClient),
		// If your trace input or output is more than 1M, and UltraLargeTraceReport is false,
		// input or output will be cut off.
		// If UltraLargeTraceReport is true, input or output will be uploaded to file storage server separately.
		// Default value is false.
		cozeloop.WithUltraLargeTraceReport(false),
		// The SDK will cache the prompts locally. You can set the max count of prompts.
		// Default value is 100.
		cozeloop.WithPromptCacheMaxCount(100),
		// The SDK will refresh the local prompts cache periodically. You can set the refresh interval.
		// Default value is 1 minute.
		cozeloop.WithPromptCacheRefreshInterval(time.Minute*1),
		// Set whether to report a trace span when get or format prompt.
		// Default value is false.
		cozeloop.WithPromptTrace(false),
	)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	// Then you can call the functions in the client.
	ctx, span := client.StartSpan(ctx, "first_span", "custom")
	span.Finish(ctx)

	// Remember to close the client when program exits. If client is not closed, traces may be lost.
	client.Close(ctx)
}
