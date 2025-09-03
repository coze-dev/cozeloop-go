// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package entity

type StreamReader[T any] interface {
	Recv() (T, error)
}
