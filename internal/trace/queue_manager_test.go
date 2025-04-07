// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

import (
	"context"
	"testing"

	. "github.com/bytedance/mockey"
	"github.com/coze-dev/cozeloop-go/internal/httpclient"
	. "github.com/smartystreets/goconvey/convey"
)

func Test_GetBatchSpanProcessor(t *testing.T) {
	ctx := context.Background()
	httpClient := &httpclient.Client{}
	spanQM := NewBatchSpanProcessor(httpClient)

	PatchConvey("Test GetBatchSpanProcessor", t, func() {
		PatchConvey("Test with valid inputs", func() {
			spanQM.OnSpanEnd(ctx, &Span{})
			err := spanQM.ForceFlush(ctx)
			So(err, ShouldBeNil)
		})
	})
}
