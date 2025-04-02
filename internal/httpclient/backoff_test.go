// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package httpclient

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/coze-dev/cozeloop-go/internal/consts"
	. "github.com/smartystreets/goconvey/convey"
)

func Test_Retry(t *testing.T) {
	ctx := context.Background()
	f := func() error { return nil } // Mock function to be retried
	retryTimes := 3
	b := NewBackoff(1*time.Millisecond, 10*time.Second)

	PatchConvey("Test f returns nil on first attempt", t, func() {
		actualErr := b.Retry(ctx, f, retryTimes)
		So(actualErr, ShouldBeNil)
	})

	PatchConvey("Test f returns AuthError", t, func() {
		mock := Mock(f).Return(&consts.AuthError{}).Build()
		actualErr := b.Retry(ctx, f, retryTimes)
		So(actualErr, ShouldNotBeNil)
		So(actualErr, ShouldEqual, &consts.AuthError{})
		So(mock.Times(), ShouldEqual, 1)
	})

	PatchConvey("Test f returns RemoteServiceError with HTTP code < 500", t, func() {
		mock := Mock(f).Return(&consts.RemoteServiceError{HttpCode: 400}).Build()
		actualErr := b.Retry(ctx, f, retryTimes)
		So(actualErr, ShouldNotBeNil)
		So(actualErr, ShouldEqual, &consts.RemoteServiceError{HttpCode: 400})
		So(mock.Times(), ShouldEqual, 1)
	})

	PatchConvey("Test f returns RemoteServiceError with HTTP code >= 500", t, func() {
		mock := Mock(f).Return(&consts.RemoteServiceError{HttpCode: 500}).Build()
		actualErr := b.Retry(ctx, f, retryTimes)
		So(actualErr, ShouldNotBeNil)
		So(actualErr, ShouldEqual, &consts.RemoteServiceError{HttpCode: 500})
		So(mock.Times(), ShouldEqual, retryTimes)
	})

	PatchConvey("Test f returns generic error and Wait succeeds", t, func() {
		mock := Mock(f).Return(errors.New("generic error")).Build()
		actualErr := b.Retry(ctx, f, retryTimes)
		So(actualErr, ShouldNotBeNil)
		So(actualErr, ShouldEqual, errors.New("generic error"))
		So(mock.Times(), ShouldEqual, retryTimes)
	})
}
