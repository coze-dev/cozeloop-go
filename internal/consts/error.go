// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package consts

import (
	"fmt"
)

var (
	ErrInvalidParam  = NewError("invalid param")
	ErrInternal      = NewError("internal error")
	ErrRemoteService = NewError("remote service error")
	ErrClientClosed  = NewError("client already closed")

	ErrAuthInfoRequired = NewError("api token or jwt oauth info is required")
	ErrParsePrivateKey  = NewError("failed to parse private key")
	ErrHeaderParent     = NewError("header traceparent is illegal")
)

type LoopError struct {
	Msg   string
	cause error
}

func NewError(msg string) *LoopError {
	return &LoopError{
		Msg: msg,
	}
}

func (e *LoopError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.cause)
	}
	return e.Msg
}

func (e *LoopError) Unwrap() error {
	return e.cause
}

func (e *LoopError) Wrap(err error) *LoopError {
	e.cause = err
	return e
}

type RemoteServiceError struct {
	HttpCode int
	ErrCode  int
	ErrMsg   string
	LogID    string
	cause    error
}

func NewRemoteServiceError(httpCode, errCode int, errMsg, logID string) *RemoteServiceError {
	return &RemoteServiceError{
		HttpCode: httpCode,
		ErrCode:  errCode,
		ErrMsg:   errMsg,
		LogID:    logID,
	}
}

func (e *RemoteServiceError) Error() string {
	base := fmt.Sprintf("%v [httpcode=%d code=%d logid=%s]",
		e.ErrMsg, e.HttpCode, e.ErrCode, e.LogID)
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", base, e.cause)
	}
	return base
}

func (e *RemoteServiceError) Unwrap() error {
	return e.cause
}

func (e *RemoteServiceError) Wrap(err error) *RemoteServiceError {
	e.cause = err
	return e
}

// authErrorFormat represents the error response from Coze API
type AuthErrorFormat struct {
	ErrorMessage string `json:"error_message"`
	ErrorCode    string `json:"error_code"`
	Error        string `json:"error"`
}

// AuthErrorCode represents authentication error codes
type AuthErrorCode string

const (
	/*
	 * The user has not completed authorization yet, please try again later
	 */
	AuthorizationPending AuthErrorCode = "authorization_pending"
	/*
	 * The request is too frequent, please try again later
	 */
	SlowDown AuthErrorCode = "slow_down"
	/*
	 * The user has denied the authorization
	 */
	AccessDenied AuthErrorCode = "access_denied"
	/*
	 * The token is expired
	 */
	ExpiredToken AuthErrorCode = "expired_token"
)

// String implements the Stringer interface
func (c AuthErrorCode) String() string {
	return string(c)
}

type AuthError struct {
	HttpCode     int
	Code         AuthErrorCode
	ErrorMessage string
	Param        string
	LogID        string
	cause        error
}

func NewAuthError(error *AuthErrorFormat, httpCode int, logID string) *AuthError {
	return &AuthError{
		HttpCode:     httpCode,
		ErrorMessage: error.ErrorMessage,
		Code:         AuthErrorCode(error.ErrorCode),
		Param:        error.Error,
		LogID:        logID,
	}
}

// LoopError implements the error interface
func (e *AuthError) Error() string {
	return fmt.Sprintf("%v [httpcode=%d code=%s param=%s logid=%s]",
		e.ErrorMessage,
		e.HttpCode,
		e.Code,
		e.Param,
		e.LogID)
}

// Unwrap returns the parent error
func (e *AuthError) Unwrap() error {
	return e.cause
}
