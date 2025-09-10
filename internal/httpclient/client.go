// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/coze-dev/cozeloop-go/internal/consts"
	"github.com/coze-dev/cozeloop-go/internal/logger"
)

type Client struct {
	baseURL       string
	httpClient    HTTPClient
	auth          Auth
	timeout       time.Duration
	uploadTimeout time.Duration
}

type ClientOptions struct {
	Timeout       time.Duration
	UploadTimeout time.Duration
}

func NewClient(baseURL string, httpClient HTTPClient, auth Auth, options *ClientOptions) *Client {
	c := &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
		auth:       auth,
	}
	if options != nil {
		c.timeout = options.Timeout
		c.uploadTimeout = options.UploadTimeout
	}
	return c
}

func (c *Client) GetWithRetry(ctx context.Context, path string, params map[string]string, resp OpenAPIResponse, retryTimes int) error {
	return defaultBackoff.Retry(ctx, func() error {
		return c.Get(ctx, path, params, resp)
	}, retryTimes)
}

func (c *Client) Get(ctx context.Context, path string, params map[string]string, resp OpenAPIResponse) error {
	var cancel context.CancelFunc
	if c.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	url := c.baseURL + path
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return consts.ErrInternal.Wrap(err)
	}

	q := request.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	request.URL.RawQuery = q.Encode()

	if err := c.setHeaders(ctx, request, map[string]string{"Content-Type": "application/json"}); err != nil {
		return err
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		logger.CtxErrorf(ctx, "http client Get failed, url: %v, err: %v", url, err)
		return consts.ErrRemoteService.Wrap(err)
	}

	return parseResponse(ctx, url, response, resp)
}

func (c *Client) PostWithRetry(ctx context.Context, path string, body any, resp OpenAPIResponse, retryTimes int) error {
	return defaultBackoff.Retry(ctx, func() error {
		return c.Post(ctx, path, body, resp)
	}, retryTimes)
}

func (c *Client) Post(ctx context.Context, path string, body any, resp OpenAPIResponse) error {
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok && c.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return consts.ErrInternal.Wrap(err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.baseURL + path
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bodyReader)
	if err != nil {
		return consts.ErrInternal.Wrap(err)
	}

	if err := c.setHeaders(ctx, request, map[string]string{"Content-Type": "application/json"}); err != nil {
		return err
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		logger.CtxErrorf(ctx, "http client Post failed, url: %v, err: %v", url, err)
		return consts.ErrRemoteService.Wrap(err)
	}

	return parseResponse(ctx, url, response, resp)
}

func (c *Client) PostStream(ctx context.Context, path string, body any) (*http.Response, error) {
	if _, ok := ctx.Deadline(); !ok && c.timeout > 0 {
		ctx, _ = context.WithTimeout(ctx, c.timeout)
	}

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.baseURL + path
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	headers := map[string]string{"Content-Type": "application/json"}
	if err := c.setHeaders(ctx, request, headers); err != nil {
		return nil, err
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		logger.CtxErrorf(ctx, "http client PostStream failed, url: %v, err: %v", url, err)
		return nil, consts.ErrRemoteService.Wrap(err)
	}
	if response.StatusCode != http.StatusOK {
		logger.CtxErrorf(ctx, "http client PostStream failed, url: %v, status code: %v", url, response.StatusCode)
		// 非200不会返回流式，而是直接返回错误信息
		return nil, parseResponse(ctx, url, response, &BaseResponse{})
	}

	return response, nil
}

func (c *Client) UploadFile(ctx context.Context, path string, fileName string, reader io.Reader, form map[string]string, resp OpenAPIResponse) error {
	var cancel context.CancelFunc
	if c.uploadTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, c.uploadTimeout)
		defer cancel()
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return consts.ErrInternal.Wrap(fmt.Errorf("create form file: %w", err))
	}

	if _, err = io.Copy(part, reader); err != nil {
		return consts.ErrInternal.Wrap(fmt.Errorf("copy file content: %w", err))
	}

	for key, value := range form {
		if err := writer.WriteField(key, value); err != nil {
			return consts.ErrInternal.Wrap(fmt.Errorf("write field %s: %w", key, err))
		}
	}

	if err := writer.Close(); err != nil {
		return consts.ErrInternal.Wrap(fmt.Errorf("close multipart writer: %w", err))
	}

	url := c.baseURL + path
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return consts.ErrInternal.Wrap(fmt.Errorf("create request: %w", err))
	}

	if err := c.setHeaders(ctx, request, map[string]string{"Content-Type": writer.FormDataContentType()}); err != nil {
		return err
	}

	response, err := c.httpClient.Do(request)
	logger.CtxDebugf(ctx, "http client upload file, url: %v, content type:%s, response: %#v",
		url, request.Header.Get("Content-Type"), response)
	if err != nil {
		logger.CtxErrorf(ctx, "http client UploadFile failed, url: %v, err: %v", url, err)
		return consts.ErrRemoteService.Wrap(err)
	}

	return parseResponse(ctx, url, response, resp)
}

func (c *Client) setHeaders(ctx context.Context, request *http.Request, headers map[string]string) error {
	for k, v := range headers {
		request.Header.Set(k, v)
	}

	if err := setAuthorizationHeader(ctx, request, c.auth); err != nil {
		return err
	}
	setUserAgent(request)

	if env := os.Getenv("x_tt_env"); env != "" {
		request.Header.Set("x-tt-env", env)
	}
	if env := os.Getenv("x_use_ppe"); env != "" {
		request.Header.Set("x-use-ppe", "1")
	}

	return nil
}

func setAuthorizationHeader(ctx context.Context, request *http.Request, auth Auth) error {
	token, err := auth.Token(ctx)
	if err != nil {
		return err
	}
	request.Header.Set(consts.AuthorizeHeader, fmt.Sprintf("Bearer %s", token))
	return nil
}

func parseResponse(ctx context.Context, url string, response *http.Response, resp OpenAPIResponse) error {
	if response == nil {
		return nil
	}
	defer response.Body.Close()

	logID := response.Header.Get(consts.LogIDHeader)
	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return consts.ErrInternal.Wrap(err)
	}

	if err := checkOAuthError(logID, respBody, response.StatusCode); err != nil {
		logger.CtxErrorf(ctx, "OAuth failed, %v", err)
		return consts.ErrRemoteService.Wrap(err)
	}

	if err = json.Unmarshal(respBody, resp); err != nil {
		logger.CtxErrorf(ctx, "call remote service failed, status code: %v, response: %v", response.StatusCode, string(respBody))
		return consts.ErrRemoteService.Wrap(consts.NewRemoteServiceError(
			response.StatusCode, -1, "", logID))
	}
	resp.SetLogID(logID)
	if resp.GetCode() != 0 {
		err := consts.ErrRemoteService.Wrap(consts.NewRemoteServiceError(
			response.StatusCode, resp.GetCode(), resp.GetMsg(), logID))
		logger.CtxErrorf(ctx, "call remote service failed, %v", err)
		return err
	}

	logger.CtxDebugf(ctx, "call remote service success, url: %v, response: %v, logID: %s",
		url, string(respBody), logID)
	return nil
}

func checkOAuthError(logID string, resp []byte, statusCode int) error {
	if statusCode != http.StatusOK {
		// oauth error has special format
		errorInfo := consts.AuthErrorFormat{}
		err := json.Unmarshal(resp, &errorInfo)
		if err != nil {
			// not oauth error, return nil
			return nil
		}

		if errorInfo.ErrorCode != "" {
			return consts.NewAuthError(&errorInfo, statusCode, logID)
		}
	}
	return nil
}
