// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package cozeloop

import (
	"context"

	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/logger"
	"github.com/coze-dev/cozeloop-go/internal/trace"
)

var defaultNoopSpan = trace.DefaultNoopSpan

// NoopClient a noop client
type NoopClient struct {
	newClientError error
}

func (c *NoopClient) GetWorkspaceID() string {
	logger.CtxWarnf(context.Background(), "Noop client not supported. %v", c.newClientError)
	return ""
}

func (c *NoopClient) Close(ctx context.Context) {
	logger.CtxWarnf(context.Background(), "Noop client not supported. %v", c.newClientError)
}

func (c *NoopClient) GetPrompt(ctx context.Context, param GetPromptParam, options ...GetPromptOption) (*entity.Prompt, error) {
	logger.CtxWarnf(context.Background(), "Noop client not supported. %v", c.newClientError)
	return nil, c.newClientError
}

func (c *NoopClient) PromptFormat(ctx context.Context, prompt *entity.Prompt, variables map[string]any, options ...PromptFormatOption) (messages []*entity.Message, err error) {
	logger.CtxWarnf(context.Background(), "Noop client not supported. %v", c.newClientError)
	return nil, c.newClientError
}

func (c *NoopClient) StartSpan(ctx context.Context, name, spanType string, opts ...StartSpanOption) (context.Context, Span) {
	logger.CtxWarnf(context.Background(), "Noop client not supported. %v", c.newClientError)
	return ctx, defaultNoopSpan
}

func (c *NoopClient) GetSpanFromContext(ctx context.Context) Span {
	logger.CtxWarnf(context.Background(), "Noop client not supported. %v", c.newClientError)
	return defaultNoopSpan
}

func (c *NoopClient) GetSpanFromHeader(ctx context.Context, header map[string]string) SpanContext {
	logger.CtxWarnf(context.Background(), "Noop client not supported. %v", c.newClientError)
	return defaultNoopSpan
}

func (c *NoopClient) Flush(ctx context.Context) {
	logger.CtxWarnf(context.Background(), "Noop client not supported. %v", c.newClientError)
}
