// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package stream

import (
	"context"
	"fmt"
	"net/http"
)

// SSEParser defines the interface for parsing SSE events into specific types
type SSEParser[T any] interface {
	Parse(sse *ServerSentEvent) (T, error)
	HandleError(sse *ServerSentEvent) error
}

// BaseStreamReader provides generic SSE stream reading capabilities
type BaseStreamReader[T any] struct {
	ctx      context.Context
	response *http.Response
	decoder  *SSEDecoder
	parser   SSEParser[T]
	closed   bool
	events   <-chan SSEEvent
}

// NewBaseStreamReader creates a new base stream reader
func NewBaseStreamReader[T any](ctx context.Context, resp *http.Response, parser SSEParser[T]) *BaseStreamReader[T] {
	decoder := NewSSEDecoder(resp.Body)
	events := decoder.Decode(ctx)

	return &BaseStreamReader[T]{
		ctx:      ctx,
		response: resp,
		decoder:  decoder,
		parser:   parser,
		closed:   false,
		events:   events,
	}
}

// Recv receives the next item from the stream
func (r *BaseStreamReader[T]) Recv() (T, error) {
	var zero T

	if r.closed {
		return zero, fmt.Errorf("stream reader is closed")
	}

	for {
		select {
		case <-r.ctx.Done():
			r.Close()
			return zero, r.ctx.Err()

		case sseEvent, ok := <-r.events:
			if !ok {
				// Channel closed, stream ended
				r.Close()
				return zero, fmt.Errorf("stream ended")
			}

			if sseEvent.Error != nil {
				r.Close()
				return zero, sseEvent.Error
			}

			if sseEvent.Event == nil {
				continue
			}

			// Check for error events first
			if err := r.parser.HandleError(sseEvent.Event); err != nil {
				r.Close()
				return zero, err
			}

			// Parse the event
			result, err := r.parser.Parse(sseEvent.Event)
			if err != nil {
				// Continue to next event for parsing errors
				continue
			}

			return result, nil
		}
	}
}

// Close closes the stream reader and releases resources
func (r *BaseStreamReader[T]) Close() error {
	if r.closed {
		return nil
	}

	r.closed = true
	if r.response != nil && r.response.Body != nil {
		return r.response.Body.Close()
	}

	return nil
}
