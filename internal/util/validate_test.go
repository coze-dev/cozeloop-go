// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package util

import "testing"

func TestIsValidMDNBase64(t *testing.T) {
	testStr := "data:image/png;base64,SGVsbG8sIFdvcmxkIQ=="
	t.Run("TestIsValidMDNBase64", func(t *testing.T) {
		if got := ParseValidMDNBase64(testStr); !got {
			t.Errorf("ParseValidMDNBase64() = %v", got)
		}
	})

}
