// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package prompt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/consts"
	"github.com/coze-dev/cozeloop-go/internal/httpclient"
	"github.com/coze-dev/cozeloop-go/internal/stream"
)

// ExecuteSSEParser implements SSEParser for ExecuteResult
type ExecuteSSEParser struct {
	logID string
}

// NewExecuteSSEParser creates a new ExecuteSSEParser
func NewExecuteSSEParser(logID string) *ExecuteSSEParser {
	return &ExecuteSSEParser{
		logID: logID,
	}
}

// Parse parses SSE event into ExecuteResult
func (p *ExecuteSSEParser) Parse(sse *stream.ServerSentEvent) (entity.ExecuteResult, error) {
	// Skip empty data
	if sse.Data == "" {
		return entity.ExecuteResult{}, nil
	}

	// Parse streaming response
	var executeStreamingData ExecuteStreamingData
	if err := json.Unmarshal([]byte(sse.Data), &executeStreamingData); err != nil {
		return entity.ExecuteResult{}, fmt.Errorf("failed to unmarshal streaming response: %w", err)
	}

	// Convert to ExecuteResult
	result := entity.ExecuteResult{}
	result.Message = toModelMessage(executeStreamingData.Message)
	result.FinishReason = executeStreamingData.FinishReason
	result.Usage = toModelTokenUsage(executeStreamingData.Usage)

	return result, nil
}

// HandleError checks if the SSE event contains an error
func (p *ExecuteSSEParser) HandleError(sse *stream.ServerSentEvent) error {
	// Check if event field contains "error" (case-insensitive)
	if sse.Event != "" && bytes.Contains([]byte(strings.ToLower(sse.Event)), []byte("error")) {
		// This is an error event, parse the data field for error information
		data := sse.Data
		if data == "" {
			// Event indicates error but no data, return generic error
			return consts.NewRemoteServiceError(http.StatusOK, -1, "Error event received without data", p.logID)
		}

		// Try to parse as error response
		var errResp httpclient.BaseResponse
		if err := json.Unmarshal([]byte(data), &errResp); err == nil {
			return consts.NewRemoteServiceError(http.StatusOK, errResp.Code, errResp.Msg, p.logID)
		}

		// If no structured error found, return raw data as error message
		return consts.NewRemoteServiceError(http.StatusOK, -1, data, p.logID)
	}

	// Event field doesn't contain "error", this is not an error event
	return nil
}

// ExecuteStreamReader wraps BaseStreamReader for ExecuteResult
type ExecuteStreamReader struct {
	*stream.BaseStreamReader[entity.ExecuteResult]
}

// NewExecuteStreamReader creates a new ExecuteStreamReader
func NewExecuteStreamReader(ctx context.Context, resp *http.Response) (*ExecuteStreamReader, error) {
	// 从响应头中获取logID
	logID := resp.Header.Get(consts.LogIDHeader)

	parser := NewExecuteSSEParser(logID)
	baseReader := stream.NewBaseStreamReader[entity.ExecuteResult](ctx, resp, parser)

	return &ExecuteStreamReader{
		BaseStreamReader: baseReader,
	}, nil
}
