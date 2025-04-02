// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package httpclient

import (
	"net/http"
)

// HTTPClient an interface for making HTTP requests
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}
