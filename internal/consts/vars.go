// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package consts

import (
	"reflect"

	"github.com/coze-dev/cozeloop-go/spec/tracespec"
)

// span

var (
	BaggageSpecialChars = []string{"=", ","}
)

var (
	TagValueSizeLimit = map[string]int{
		tracespec.Input:  MaxBytesOfOneTagValueOfInputOutput,
		tracespec.Output: MaxBytesOfOneTagValueOfInputOutput,
	}
)

var typeInt64 int64
var typeStr string
var typeInt int
var typeInt32 int32

// ReserveFieldTypes Define the allowed types for each reserved field.
var ReserveFieldTypes = map[string][]reflect.Type{
	UserID:                 {reflect.TypeOf(typeStr)},
	MessageID:              {reflect.TypeOf(typeStr)},
	ThreadID:               {reflect.TypeOf(typeStr)},
	tracespec.InputTokens:  {reflect.TypeOf(typeInt64), reflect.TypeOf(typeInt), reflect.TypeOf(typeInt32)},
	tracespec.OutputTokens: {reflect.TypeOf(typeInt64), reflect.TypeOf(typeInt), reflect.TypeOf(typeInt32)},
	tracespec.Tokens:       {reflect.TypeOf(typeInt64), reflect.TypeOf(typeInt), reflect.TypeOf(typeInt32)},
	StartTimeFirstResp:     {reflect.TypeOf(typeInt64), reflect.TypeOf(typeInt), reflect.TypeOf(typeInt32)},
	LatencyFirstResp:       {reflect.TypeOf(typeInt64), reflect.TypeOf(typeInt), reflect.TypeOf(typeInt32)},
}
