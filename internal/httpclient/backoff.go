// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package httpclient

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/coze-dev/cozeloop-go/internal/consts"
)

var defaultBackoff = NewBackoff(defaultBaseDelay, defaultMaxDelay)

const (
	defaultBaseDelay   = 200 * time.Millisecond
	defaultMaxDelay    = 10 * time.Second
	defaultMaxAttempts = 3
)

type Backoff struct {
	baseDelay time.Duration
	maxDelay  time.Duration
}

func NewBackoff(baseDelay, maxDelay time.Duration) *Backoff {
	if baseDelay <= 0 {
		baseDelay = defaultBaseDelay
	}
	if maxDelay <= 0 || baseDelay > maxDelay {
		maxDelay = defaultMaxDelay
	}
	return &Backoff{
		baseDelay: baseDelay,
		maxDelay:  maxDelay,
	}
}

func (b *Backoff) Wait(ctx context.Context, currentRetryTimes int) error {
	delay := b.baseDelay * time.Duration(math.Pow(2, float64(currentRetryTimes)))
	if delay > b.maxDelay {
		delay = b.maxDelay
	}

	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *Backoff) Retry(ctx context.Context, f func() error, retryTimes int) error {
	var err error
	for i := 0; i < retryTimes; i++ {
		err = f()
		if err == nil {
			return err
		}
		// auth error needn't retry
		var authError *consts.AuthError
		if isAuthError := errors.As(err, &authError); isAuthError {
			return err
		}
		var remoteServiceError *consts.RemoteServiceError
		if isRemoteServiceError := errors.As(err, &remoteServiceError); isRemoteServiceError {
			// 3xx 4xx error needn't retry
			if remoteServiceError.HttpCode < 500 {
				return err
			}
		}

		if waitErr := b.Wait(ctx, i); waitErr != nil {
			return waitErr
		}
	}
	return err
}
