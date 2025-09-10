// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math/rand"
	"strconv"
	"sync"
	"unicode/utf8"

	"github.com/coze-dev/cozeloop-go/internal/consts"
)

func RmDupStrSlice(slice []string) []string {
	uniMap := make(map[string]bool)
	var res []string
	for _, str := range slice {
		if !uniMap[str] {
			uniMap[str] = true
			res = append(res, str)
		}
	}
	return res
}

var bufferPool = sync.Pool{New: func() interface{} {
	return new(bytes.Buffer)
}}

func GetStringBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

func RecycleStringBuffer(buffer *bytes.Buffer) {
	buffer.Reset()
	bufferPool.Put(buffer)
}

func MapToStringString(mp map[string]string) string {
	if len(mp) == 0 {
		return ""
	}
	buffer := GetStringBuffer()

	count := len(mp)
	for key, value := range mp {
		buffer.WriteString(key)
		buffer.WriteString(consts.Equal)
		buffer.WriteString(value)
		if count > 1 {
			buffer.WriteString(consts.Comma)
			count--
		}
	}
	s := buffer.String()
	RecycleStringBuffer(buffer)
	return s
}

func PtrValue[T any](s *T) T {
	if s != nil {
		return *s
	}
	var empty T
	return empty
}

func Ptr[T any](s T) *T {
	return &s
}

func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return bytesToHex(bytes), nil
}

func bytesToHex(bytes []byte) string {
	hex := make([]byte, len(bytes)*2)
	for i, b := range bytes {
		hex[i*2] = hexChar(b >> 4)
		hex[i*2+1] = hexChar(b & 0xF)
	}
	return string(hex)
}

func hexChar(b byte) byte {
	if b < 10 {
		return '0' + b
	}
	return 'a' + (b - 10)
}

func GetValueOfInt(value interface{}) int64 {
	switch value.(type) {
	case int:
		return int64(value.(int))
	case int32:
		return int64(value.(int32))
	case int64:
		return value.(int64)
	case string:
		strValue, _ := value.(string)
		i64, err := strconv.ParseInt(strValue, 10, 64)
		if err != nil {
			return 0
		}
		return i64
	default:
		return 0
	}
}

// TruncateStringByChar Truncate the string to the first n characters
func TruncateStringByChar(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}

	result := make([]byte, 0, len(s))
	for i, r := range s {
		if i >= n {
			break
		}
		result = append(result, string(r)...)
	}

	return string(result)
}

func TruncateStringByByte(valueStr string, limit int) (string, bool) {
	if len(valueStr) <= limit {
		return valueStr, false
	}

	return valueStr[:limit], true
}

func ToJSON(param interface{}) string {
	if param == nil {
		return ""
	}
	if paramStr, ok := param.(string); ok {
		return paramStr
	}
	byteRes, err := json.Marshal(param)
	if err != nil {
		return ""
	}
	return string(byteRes)
}

func Stringify(value interface{}) string {
	switch tv := value.(type) {
	case nil:
		return ""
	case bool:
		if tv {
			return "true"
		} else {
			return "false"
		}
	case string:
		return tv
	case []byte:
		return string(tv)
	case fmt.Stringer:
		defer func() { recover() }()
		return tv.String()
	case error:
		return tv.Error()
	case int:
		return strconv.Itoa(tv)
	case int16:
		return strconv.FormatInt(int64(tv), 10)
	case int32:
		return strconv.FormatInt(int64(tv), 10)
	case int64:
		return strconv.FormatInt(int64(tv), 10)
	case uint:
		return strconv.FormatUint(uint64(tv), 10)
	case uint16:
		return strconv.FormatUint(uint64(tv), 10)
	case uint32:
		return strconv.FormatUint(uint64(tv), 10)
	case uint64:
		return strconv.FormatUint(uint64(tv), 10)
	case float32:
		return strconv.FormatFloat(float64(tv), 'f', 3, 32)
	case float64:
		return strconv.FormatFloat(float64(tv), 'f', 3, 32)
	default:
		return fmt.Sprint(value)
	}
}
