// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package httpclient

import (
	"context"
	"time"

	"github.com/coze-dev/cozeloop-go/internal/consts"
	"github.com/coze-dev/cozeloop-go/internal/logger"
	"github.com/coze-dev/cozeloop-go/internal/util"
	"golang.org/x/sync/singleflight"
)

type Auth interface {
	Token(ctx context.Context) (string, error)
}

var (
	_ Auth = &tokenAuthImpl{}
	_ Auth = &jwtOAuthImpl{}
)

// tokenAuthImpl implements the Auth interface with fixed access token.
type tokenAuthImpl struct {
	accessToken string
}

// NewTokenAuth creates a new token authentication instance.
func NewTokenAuth(accessToken string) Auth {
	return &tokenAuthImpl{
		accessToken: accessToken,
	}
}

// Token returns the access token.
func (r *tokenAuthImpl) Token(ctx context.Context) (string, error) {
	return r.accessToken, nil
}

func NewJWTAuth(client *JWTOAuthClient, opt *GetJWTAccessTokenReq) Auth {
	ttl := consts.DefaultOAuthRefreshTTL
	if opt == nil {
		return &jwtOAuthImpl{
			TTL:    ttl,
			client: client,
		}
	}
	if opt.TTL > consts.OAuthRefreshAdvanceTime {
		ttl = opt.TTL
	}
	return &jwtOAuthImpl{
		TTL:         ttl,
		Scope:       opt.Scope,
		SessionName: opt.SessionName,
		client:      client,
		accountID:   opt.AccountID,
	}
}

type jwtOAuthImpl struct {
	TTL         time.Duration
	SessionName *string
	Scope       *Scope
	client      *JWTOAuthClient
	accessToken *string
	expireIn    int64
	accountID   *int64
	group       singleflight.Group
}

func (r *jwtOAuthImpl) needRefresh() bool {
	beforeSecond := consts.OAuthRefreshAdvanceTime
	return r.accessToken == nil || time.Now().Add(beforeSecond).Unix() > r.expireIn
}

func (r *jwtOAuthImpl) Token(ctx context.Context) (string, error) {
	if !r.needRefresh() {
		return util.PtrValue(r.accessToken), nil
	}
	logger.CtxDebugf(ctx, "jwt token need refresh")
	val, err, _ := r.group.Do("jwt_token", func() (interface{}, error) {
		logger.CtxDebugf(ctx, "get jwt token")
		resp, err := r.client.GetAccessToken(ctx, &GetJWTAccessTokenReq{
			TTL:         r.TTL,
			SessionName: r.SessionName,
			Scope:       r.Scope,
			AccountID:   r.accountID,
		})
		if err != nil {
			return "", err
		}
		r.accessToken = util.Ptr(resp.AccessToken)
		r.expireIn = resp.ExpiresIn
		return resp.AccessToken, nil
	})
	if err != nil {
		return "", err
	}
	return val.(string), nil
}
