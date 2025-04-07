// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

import (
	"context"
	"time"

	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/spec/tracespec"
)

var DefaultNoopSpan = &noopSpan{}

type noopSpan struct{}

// implement of commonSpanSetter
func (n noopSpan) SetInput(ctx context.Context, input interface{})                       {}
func (n noopSpan) SetOutput(ctx context.Context, output interface{})                     {}
func (n noopSpan) SetError(ctx context.Context, err error)                               {}
func (n noopSpan) SetStatusCode(ctx context.Context, code int)                           {}
func (n noopSpan) SetUserID(ctx context.Context, userID string)                          {}
func (n noopSpan) SetUserIDBaggage(ctx context.Context, userID string)                   {}
func (n noopSpan) SetMessageID(ctx context.Context, messageID string)                    {}
func (n noopSpan) SetMessageIDBaggage(ctx context.Context, messageID string)             {}
func (n noopSpan) SetThreadID(ctx context.Context, threadID string)                      {}
func (n noopSpan) SetThreadIDBaggage(ctx context.Context, threadID string)               {}
func (n noopSpan) SetPrompt(ctx context.Context, prompt entity.Prompt)                   {}
func (n noopSpan) SetModelProvider(ctx context.Context, modelProvider string)            {}
func (n noopSpan) SetModelName(ctx context.Context, modelName string)                    {}
func (n noopSpan) SetModelCallOptions(ctx context.Context, modelCallOptions interface{}) {}
func (n noopSpan) SetInputTokens(ctx context.Context, inputTokens int)                   {}
func (n noopSpan) SetOutputTokens(ctx context.Context, outputTokens int)                 {}
func (n noopSpan) SetStartTimeFirstResp(ctx context.Context, startTimeFirstResp int64)   {}
func (n noopSpan) SetRuntime(ctx context.Context, runtime tracespec.Runtime)             {}

// implement of Span
func (n noopSpan) SetTags(ctx context.Context, tagKVs map[string]interface{})     {}
func (n noopSpan) SetBaggage(ctx context.Context, baggageItems map[string]string) {}
func (n noopSpan) GetBaggage() map[string]string                                  { return nil }
func (n noopSpan) Finish(ctx context.Context)                                     {}
func (n noopSpan) GetTraceID() string                                             { return "" }
func (n noopSpan) GetSpanID() string                                              { return "" }
func (n noopSpan) GetStartTime() time.Time                                        { return time.Time{} }
func (n noopSpan) ToHeader() (map[string]string, error)                           { return nil, nil }
