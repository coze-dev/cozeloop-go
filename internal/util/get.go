// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package util

import (
	"fmt"
	"math"
	"time"

	"github.com/coze-dev/cozeloop-go/internal/consts"
	"github.com/coze-dev/cozeloop-go/internal/idgen"
)

func Gen16CharID() string {
	rand := idgen.GetMultipleDeltaIdGenerator().GenId()
	return fmt.Sprintf("%016x", rand&math.MaxInt64)
}

func Gen32CharID() string {
	high := uint64(time.Now().Unix()) + idgen.GetMultipleDeltaIdGenerator().GenId()
	high = high & math.MaxInt64
	low := idgen.GetMultipleDeltaIdGenerator().GenId() & math.MaxInt64
	return fmt.Sprintf("%016x%016x", high, low)
}

func GetTagValueSizeLimit(tagKey string) int {
	t, ok := consts.TagValueSizeLimit[tagKey]
	if ok {
		return t
	}

	return consts.MaxBytesOfOneTagValueDefault
}

func GetTagKeySizeLimit() int {
	return consts.MaxBytesOfOneTagKeyDefault
}
