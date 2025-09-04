// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package stream

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/consts"
)

type streamReader[T, R any] struct {
	ctx            context.Context
	processor      *Processor[R]
	response       *http.Response
	recvMiddleware func(chunk R, err error) (T, error)
}

// NewStreamReader 创建新的泛型流式读取器
func NewStreamReader[T, R any](ctx context.Context, resp *http.Response, recvMiddleware func(chunk R, err error) (T, error)) (entity.StreamReader[T], error) {
	if recvMiddleware == nil {
		return nil, fmt.Errorf("recv middleware is nil")
	}
	reader := bufio.NewReader(resp.Body)
	errAccumulator := NewErrorAccumulator()
	logID := resp.Header.Get(consts.LogIDHeader)

	processor := NewProcessor[R](logID, reader, errAccumulator, json.Unmarshal)

	return &streamReader[T, R]{
		ctx:            ctx,
		processor:      processor,
		response:       resp,
		recvMiddleware: recvMiddleware,
	}, nil
}

// Recv 接收下一个流式响应
func (r *streamReader[T, R]) Recv() (T, error) {
	select {
	case <-r.ctx.Done():
		r.response.Body.Close()
		return *new(T), r.ctx.Err()
	default:
		response, err := r.processor.ProcessLines()
		if err != nil {
			r.response.Body.Close()
		}
		return r.recvMiddleware(response, err)
	}
}
