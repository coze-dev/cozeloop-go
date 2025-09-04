// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package stream

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"

	"github.com/coze-dev/cozeloop-go/internal/consts"
	"github.com/coze-dev/cozeloop-go/internal/httpclient"
)

var (
	headerData    = []byte("data:")
	eventData     = []byte("event:")
	errorPrefixes = [][]byte{
		[]byte(`data:{"code":`),
		[]byte(`data:{"error":`),
	}
)

// ErrorAccumulator 错误累加器接口
type ErrorAccumulator interface {
	Write(p []byte) error
	Bytes() []byte
}

type Unmarshal func(data []byte, v interface{}) error

// DefaultErrorAccumulator 默认错误累加器实现
type DefaultErrorAccumulator struct {
	buffer bytes.Buffer
}

// NewErrorAccumulator 创建新的错误累加器
func NewErrorAccumulator() ErrorAccumulator {
	return &DefaultErrorAccumulator{}
}

func (e *DefaultErrorAccumulator) Write(p []byte) error {
	_, err := e.buffer.Write(p)
	return err
}

func (e *DefaultErrorAccumulator) Bytes() []byte {
	if e.buffer.Len() == 0 {
		return nil
	}
	return e.buffer.Bytes()
}

// Processor 内部流处理器
type Processor[T any] struct {
	logid          string
	reader         *bufio.Reader
	errAccumulator ErrorAccumulator
	unmarshal      Unmarshal
}

// NewProcessor 创建新的流处理器
func NewProcessor[T any](logid string, reader *bufio.Reader, errAccumulator ErrorAccumulator, unmarshaler Unmarshal) *Processor[T] {
	return &Processor[T]{
		logid:          logid,
		reader:         reader,
		errAccumulator: errAccumulator,
		unmarshal:      unmarshaler,
	}
}

// ProcessLines 处理流式数据行，这是内部方法
func (p *Processor[T]) ProcessLines() (T, error) {
	var (
		hasErrorPrefix bool
		zero           T
	)

	for {
		rawLine, readErr := p.reader.ReadBytes('\n')
		if readErr != nil || hasErrorPrefix {
			respErr := p.unmarshalError()
			if respErr != nil {
				return zero, consts.NewRemoteServiceError(http.StatusOK, respErr.Code, respErr.Msg, p.logid)
			}
			return zero, readErr
		}

		noSpaceLine := bytes.TrimSpace(rawLine)
		if p.hasError(noSpaceLine) {
			hasErrorPrefix = true
		}

		// 跳过空行
		if len(noSpaceLine) == 0 {
			continue
		}

		// 忽略 Event Data
		if bytes.HasPrefix(noSpaceLine, eventData) {
			continue
		}

		if !bytes.HasPrefix(noSpaceLine, headerData) || hasErrorPrefix {
			if hasErrorPrefix {
				noSpaceLine = bytes.TrimPrefix(noSpaceLine, headerData)
			}
			writeErr := p.errAccumulator.Write(noSpaceLine)
			if writeErr != nil {
				return zero, writeErr
			}
			continue
		}

		noPrefixLine := bytes.TrimPrefix(noSpaceLine, headerData)
		var response T
		unmarshalErr := p.unmarshal(noPrefixLine, &response)
		if unmarshalErr != nil {
			return zero, fmt.Errorf("failed to unmarshal streaming response: %w", unmarshalErr)
		}

		return response, nil
	}
}

func (p *Processor[T]) hasError(input []byte) bool {
	for _, prefix := range errorPrefixes {
		if bytes.HasPrefix(input, prefix) {
			return true
		}
	}
	return false
}

func (p *Processor[T]) unmarshalError() (errResp *httpclient.BaseResponse) {
	errBytes := p.errAccumulator.Bytes()
	if len(errBytes) == 0 {
		return
	}

	err := p.unmarshal(errBytes, &errResp)
	if err != nil {
		errResp = nil
	}

	return
}
