// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package loop

import (
	"context"

	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/logger"
	"github.com/coze-dev/cozeloop-go/internal/trace"
)

var defaultNoopSpan = trace.DefaultNoopSpan

// noopClient a noop client
type noopClient struct {
	newClientError error
}

func (c *noopClient) GetWorkspaceID() string {
	logger.CtxWarnf(context.Background(), "Noop client not supported. %v", c.newClientError)
	return ""
}

func (c *noopClient) Close(ctx context.Context) {
	logger.CtxWarnf(context.Background(), "Noop client not supported. %v", c.newClientError)
}

func (c *noopClient) GetPrompt(ctx context.Context, param GetPromptParam, options ...GetPromptOption) (*entity.Prompt, error) {
	logger.CtxWarnf(context.Background(), "Noop client not supported. %v", c.newClientError)
	return nil, c.newClientError
}

func (c *noopClient) PromptFormat(ctx context.Context, prompt *entity.Prompt, variables map[string]any, options ...PromptFormatOption) (messages []*entity.Message, err error) {
	logger.CtxWarnf(context.Background(), "Noop client not supported. %v", c.newClientError)
	return nil, c.newClientError
}

func (c *noopClient) StartSpan(ctx context.Context, name, spanType string, opts ...StartSpanOption) (context.Context, Span) {
	logger.CtxWarnf(context.Background(), "Noop client not supported. %v", c.newClientError)
	return ctx, defaultNoopSpan
}

func (c *noopClient) GetSpanFromContext(ctx context.Context) Span {
	logger.CtxWarnf(context.Background(), "Noop client not supported. %v", c.newClientError)
	return defaultNoopSpan
}

func (c *noopClient) GetSpanFromHeader(ctx context.Context, header map[string]string) SpanContext {
	logger.CtxWarnf(context.Background(), "Noop client not supported. %v", c.newClientError)
	return defaultNoopSpan
}

func (c *noopClient) Flush(ctx context.Context) {
	logger.CtxWarnf(context.Background(), "Noop client not supported. %v", c.newClientError)
}
