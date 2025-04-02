// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package httpclient

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/coze-dev/cozeloop-go/internal/consts"
	. "github.com/smartystreets/goconvey/convey"
)

func Test_Get(t *testing.T) {
	ctx := context.Background()
	path := "/api/v1/data"
	params := map[string]string{"param1": "value1", "param2": "value2"}
	httpclient := &mockHttpClient{}
	auth := &mockAuthImpl{}
	client := NewClient("http://test", httpclient, auth, nil)
	resp := &BaseResponse{}

	PatchConvey("Test httpClient.Do failed", t, func() {
		Mock((*mockHttpClient).Do).Return(nil, errors.New("http client do failed")).Build()
		err := client.Get(ctx, path, params, resp)
		So(err, ShouldNotBeNil)
		So(errors.Is(err, consts.ErrRemoteService), ShouldBeTrue)
	})

	PatchConvey("Test return auth error", t, func() {
		Mock((*mockHttpClient).Do).Return(&http.Response{StatusCode: 403, Body: buildBody("{\"error_code\":\"invalid_request\"}")}, nil).Build()
		err := client.Get(ctx, path, params, resp)
		So(err, ShouldNotBeNil)
		authErr := &consts.AuthError{}
		So(errors.As(err, &authErr), ShouldBeTrue)
		So(authErr.Code, ShouldEqual, consts.AuthErrorCode("invalid_request"))
	})

	PatchConvey("Test return 5xx error", t, func() {
		Mock((*mockHttpClient).Do).Return(&http.Response{StatusCode: 500, Body: buildBody("{\"code\":4000}")}, nil).Build()
		err := client.Get(ctx, path, params, resp)
		So(err, ShouldNotBeNil)
		remoteServiceErr := &consts.RemoteServiceError{}
		So(errors.As(err, &remoteServiceErr), ShouldBeTrue)
		So(remoteServiceErr.ErrCode, ShouldEqual, 4000)
	})

	PatchConvey("Test Get success", t, func() {
		Mock((*mockHttpClient).Do).Return(&http.Response{StatusCode: 200, Body: buildBody("{\"code\":0}")}, nil).Build()
		err := client.Get(ctx, path, params, resp)
		So(err, ShouldBeNil)
		So(resp.Code, ShouldEqual, 0)
	})
}

func Test_Post(t *testing.T) {
	ctx := context.Background()
	path := "/api/v1/data"
	httpclient := &mockHttpClient{}
	auth := &mockAuthImpl{}
	client := NewClient("http://test", httpclient, auth, nil)
	resp := &BaseResponse{}

	PatchConvey("Test Post success", t, func() {
		Mock((*mockHttpClient).Do).Return(&http.Response{StatusCode: 200, Body: buildBody("{\"code\":0}")}, nil).Build()
		err := client.Post(ctx, path, "body", resp)
		So(err, ShouldBeNil)
		So(resp.Code, ShouldEqual, 0)
	})
}

func Test_UploadFile(t *testing.T) {
	ctx := context.Background()
	path := "/api/v1/upload"
	fileName := "test.txt"
	reader := bytes.NewReader([]byte("test content"))
	form := map[string]string{"key": "value"}
	httpclient := &mockHttpClient{}
	auth := &mockAuthImpl{}
	client := NewClient("http://test", httpclient, auth, nil)
	resp := &BaseResponse{}

	PatchConvey("Test UploadFile success", t, func() {
		Mock((*mockHttpClient).Do).Return(&http.Response{StatusCode: 200, Body: buildBody("{\"code\":0}")}, nil).Build()
		err := client.UploadFile(ctx, path, fileName, reader, form, resp)
		So(err, ShouldBeNil)
		So(resp.Code, ShouldEqual, 0)
	})
}

type mockHttpClient struct{}

func (c *mockHttpClient) Do(req *http.Request) (*http.Response, error) {
	return nil, nil
}

type mockAuthImpl struct{}

func (a *mockAuthImpl) Token(ctx context.Context) (string, error) {
	return "test", nil
}

func buildBody(body string) io.ReadCloser {
	return io.NopCloser(bytes.NewReader([]byte(body)))
}
