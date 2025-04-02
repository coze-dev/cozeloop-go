// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package httpclient

type OpenAPIResponse interface {
	GetCode() int
	GetMsg() string
	GetLogID() string
	SetLogID(string)
}

type BaseResponse struct {
	Code  int    `json:"code"`
	Msg   string `json:"msg"`
	LogID string
}

func (b *BaseResponse) GetCode() int {
	return b.Code
}

func (b *BaseResponse) GetMsg() string {
	return b.Msg
}

func (b *BaseResponse) GetLogID() string {
	return b.LogID
}

func (b *BaseResponse) SetLogID(logID string) {
	b.LogID = logID
}
