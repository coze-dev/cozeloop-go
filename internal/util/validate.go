// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package util

import (
	"encoding/base64"
	"net/url"
	"strings"
)

func IsValidURL(u string) bool {
	parsedURL, err := url.ParseRequestURI(u)
	if err != nil {
		return false
	}

	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return false
	}

	return true
}

// ParseValidMDNBase64 MDN: https://developer.mozilla.org/en-US/docs/Web/URI/Reference/Schemes/data#syntax
func ParseValidMDNBase64(mdnBase64 string) (string, bool) {
	ss := strings.Split(mdnBase64, ",")
	if len(ss) != 2 {
		return "", false
	}

	base64Data := ss[1]
	if len(base64Data) == 0 {
		return "", false
	}

	_, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", false
	}

	return base64Data, true
}

func IsValidHexStr(s string) bool {
	for _, c := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return false
		}
	}
	return true
}
