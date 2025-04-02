// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package httpclient

import (
	"context"
	"testing"
	"time"

	"github.com/coze-dev/cozeloop-go/internal/consts"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTokenAuthImpl(t *testing.T) {
	ctx := context.Background()
	accessToken := "test_token"
	auth := NewTokenAuth(accessToken)

	Convey("Test TokenAuthImpl Token method", t, func() {
		token, err := auth.Token(ctx)
		So(err, ShouldBeNil)
		So(token, ShouldEqual, accessToken)
	})
}

func TestJWTAuthImpl(t *testing.T) {
	ctx := context.Background()
	client := &JWTOAuthClient{}
	opt := &GetJWTAccessTokenReq{
		TTL: consts.DefaultOAuthRefreshTTL,
	}
	auth := NewJWTAuth(client, opt)

	Convey("Test JWTAuthImpl Token method", t, func() {
		mockClient := Mock((*JWTOAuthClient).GetAccessToken).Return(&OAuthToken{
			AccessToken: "jwt_token",
			ExpiresIn:   time.Now().Add(1 * time.Hour).Unix(),
		}, nil).Build()
		token, err := auth.Token(ctx)
		So(err, ShouldBeNil)
		So(token, ShouldEqual, "jwt_token")
		So(mockClient.Times(), ShouldEqual, 1)

		token, err = auth.Token(ctx)
		So(err, ShouldBeNil)
		So(token, ShouldEqual, "jwt_token")
		So(mockClient.Times(), ShouldEqual, 1)

		auth.(*jwtOAuthImpl).expireIn = time.Now().Unix()
		token, err = auth.Token(ctx)
		So(err, ShouldBeNil)
		So(token, ShouldEqual, "jwt_token")
		So(mockClient.Times(), ShouldEqual, 2)
	})
}
