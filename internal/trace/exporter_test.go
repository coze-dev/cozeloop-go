// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package trace

import (
	"context"
	"testing"

	"github.com/coze-dev/cozeloop-go/internal/httpclient"
	. "github.com/smartystreets/goconvey/convey"
)

func Test_ExportSpans(t *testing.T) {
	ctx := context.Background()
	spans := []*UploadSpan{&UploadSpan{}, &UploadSpan{}}

	PatchConvey("Test transferToUploadSpanAndFile failed", t, func() {
		Mock((*httpclient.Client).Post).Return(nil).Build()
		err := (&SpanExporter{}).ExportSpans(ctx, spans)
		So(err, ShouldBeNil)
	})
}
