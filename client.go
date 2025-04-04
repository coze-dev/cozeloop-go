// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package cozeloop

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/consts"
	"github.com/coze-dev/cozeloop-go/internal/httpclient"
	"github.com/coze-dev/cozeloop-go/internal/logger"
	"github.com/coze-dev/cozeloop-go/internal/prompt"
	"github.com/coze-dev/cozeloop-go/internal/trace"
)

// Client interface of loop client.
// The client is thread-safe. **Do not** create multiple instances.
type Client interface {
	// PromptClient interface of prompt client
	PromptClient
	// TraceClient interface of trace client
	TraceClient

	// GetWorkSpaceID return workspace id
	GetWorkspaceID() string
	// Close close the client. Should be called before program exit.
	Close(ctx context.Context)
}

type Option func(o *options)

// HttpClient interface of HttpClient, can use http.DefaultClient
type HttpClient = httpclient.HTTPClient

type options struct {
	apiBaseURL    string
	workspaceID   string
	httpClient    HttpClient
	timeout       time.Duration
	uploadTimeout time.Duration

	apiToken            string
	jwtOAuthClientID    string
	jwtOAuthPrivateKey  string
	jwtOAuthPublicKeyID string

	ultraLargeReport bool

	promptCacheMaxCount        int
	promptCacheRefreshInterval time.Duration
	promptTrace                bool
}

func (o *options) MD5() string {
	h := md5.New()
	separator := "\t"
	h.Write([]byte(o.apiBaseURL + separator))
	h.Write([]byte(o.workspaceID + separator))
	h.Write([]byte(fmt.Sprintf("%p", o.httpClient) + separator))
	h.Write([]byte(o.timeout.String() + separator))
	h.Write([]byte(o.uploadTimeout.String() + separator))
	h.Write([]byte(o.apiToken + separator))
	h.Write([]byte(o.jwtOAuthClientID + separator))
	h.Write([]byte(o.jwtOAuthPrivateKey + separator))
	h.Write([]byte(o.jwtOAuthPublicKeyID + separator))
	h.Write([]byte(fmt.Sprintf("%v", o.ultraLargeReport) + separator))
	h.Write([]byte(fmt.Sprintf("%d", o.promptCacheMaxCount) + separator))
	h.Write([]byte(o.promptCacheRefreshInterval.String() + separator))
	h.Write([]byte(fmt.Sprintf("%v", o.promptTrace) + separator))
	return hex.EncodeToString(h.Sum(nil))
}

func defaultOptions() options {
	opts := options{
		apiBaseURL:                 CnBaseURL,
		httpClient:                 http.DefaultClient,
		timeout:                    consts.DefaultTimeout,
		uploadTimeout:              consts.DefaultUploadTimeout,
		ultraLargeReport:           false,
		promptCacheMaxCount:        consts.DefaultPromptCacheMaxCount,
		promptCacheRefreshInterval: consts.DefaultPromptCacheRefreshInterval,
		promptTrace:                false,
	}
	return opts
}

// NewClient creates a new loop client with the provided options.
// The client is thread-safe. **Do not** create multiple instances.
func NewClient(opts ...Option) (Client, error) {
	options := defaultOptions()
	buildOptionsFromEnv(&options)

	for _, opt := range opts {
		opt(&options)
	}

	options.apiBaseURL = strings.TrimRight(strings.TrimSpace(options.apiBaseURL), "/")

	if err := checkOptions(&options); err != nil {
		return nil, err
	}

	cacheKey := options.MD5()
	if cachedClient, ok := clientCache.Load(cacheKey); ok {
		logger.CtxWarnf(context.Background(), "You shouldn't creating a client with same options repeatedly, "+
			"return the cached client instead.")
		return cachedClient.(*loopClient), nil
	}

	auth, err := buildAuth(options)
	if err != nil {
		return nil, err
	}

	c := &loopClient{
		workspaceID: options.workspaceID,
	}
	httpClient := httpclient.NewClient(options.apiBaseURL, options.httpClient, auth,
		&httpclient.ClientOptions{
			Timeout:       options.timeout,
			UploadTimeout: options.uploadTimeout,
		})
	c.traceProvider = trace.NewTraceProvider(httpClient, trace.Options{
		WorkspaceID:      options.workspaceID,
		UltraLargeReport: options.ultraLargeReport,
	})
	c.promptProvider = prompt.NewPromptProvider(httpClient, c.traceProvider, prompt.Options{
		WorkspaceID:                options.workspaceID,
		PromptCacheMaxCount:        options.promptCacheMaxCount,
		PromptCacheRefreshInterval: options.promptCacheRefreshInterval,
		PromptTrace:                options.promptTrace,
	})

	clientCache.Store(cacheKey, c)
	return c, nil
}

// WithAPIToken set api token. You can get it from https://www.coze.cn/open/oauth/pats
// **APIToken is just used for testing.** You should use JWTOauth in production.
func WithAPIToken(apiToken string) Option {
	return func(p *options) {
		p.apiToken = apiToken
	}
}

// WithJWTOAuthClientID set jwt oauth client id. You can get it from https://www.coze.cn/open/oauth/apps
func WithJWTOAuthClientID(clientID string) Option {
	return func(p *options) {
		p.jwtOAuthClientID = clientID
	}
}

// WithJWTOAuthPrivateKey set jwt oauth private key. You can get it from https://www.coze.cn/open/oauth/apps
func WithJWTOAuthPrivateKey(privateKey string) Option {
	return func(p *options) {
		p.jwtOAuthPrivateKey = privateKey
	}
}

// WithJWTOAuthPublicKeyID set jwt oauth public key id. You can get it from https://www.coze.cn/open/oauth/apps
func WithJWTOAuthPublicKeyID(publicKeyID string) Option {
	return func(p *options) {
		p.jwtOAuthPublicKeyID = publicKeyID
	}
}

// WithAPIBaseURL set api base url. Generally, there's no need to use it. Default is http://api.coze.cn
func WithAPIBaseURL(apiBaseURL string) Option {
	return func(p *options) {
		p.apiBaseURL = apiBaseURL
	}
}

// WithWorkspaceID set workspace id.
func WithWorkspaceID(workspaceID string) Option {
	return func(p *options) {
		p.workspaceID = workspaceID
	}
}

// WithHTTPClient set http client. All http call inside SDK will use this HttpClient. Default is http.DefaultClient
func WithHTTPClient(client HttpClient) Option {
	return func(p *options) {
		p.httpClient = client
	}
}

// WithTimeout set timeout when communicating with loop server. Default is 3s
func WithTimeout(timeout time.Duration) Option {
	return func(p *options) {
		p.timeout = timeout
	}
}

// WithUploadTimeout set timeout when uploading images or files to loop server. Default is 30s
func WithUploadTimeout(timeout time.Duration) Option {
	return func(p *options) {
		p.uploadTimeout = timeout
	}
}

// WithUltraLargeTraceReport set whether to report ultra large trace report. Default is false
func WithUltraLargeTraceReport(enable bool) Option {
	return func(p *options) {
		p.ultraLargeReport = enable
	}
}

// WithPromptCacheMaxCount set prompt cache max count. Default is 100
func WithPromptCacheMaxCount(count int) Option {
	return func(p *options) {
		p.promptCacheMaxCount = count
	}
}

// WithPromptCacheRefreshInterval set prompt cache refresh interval. Default is 10 minute
func WithPromptCacheRefreshInterval(interval time.Duration) Option {
	return func(p *options) {
		p.promptCacheRefreshInterval = interval
	}
}

// WithPromptTrace set whether to report trace when get and format prompt. Default is false
func WithPromptTrace(enable bool) Option {
	return func(p *options) {
		p.promptTrace = enable
	}
}

// GetWorkspaceID return space id
func GetWorkspaceID() string {
	return getDefaultClient().GetWorkspaceID()
}

// Close close the client. Should be called before program exit.
func Close(ctx context.Context) {
	getDefaultClient().Close(ctx)
}

// GetPrompt get prompt by prompt key and version
func GetPrompt(ctx context.Context, param GetPromptParam, options ...GetPromptOption) (*entity.Prompt, error) {
	return getDefaultClient().GetPrompt(ctx, param, options...)
}

// PromptFormat format prompt with variables
func PromptFormat(ctx context.Context, prompt *entity.Prompt, variables map[string]any, options ...PromptFormatOption) (
	messages []*entity.Message, err error) {
	return getDefaultClient().PromptFormat(ctx, prompt, variables, options...)
}

// StartSpan Generate a span that automatically links to the previous span in the context.
// The start time of the span starts counting from the call of StartSpan.
// The generated span will be automatically written into the context.
// Subsequent spans that need to be chained should call StartSpan based on the new context.
func StartSpan(ctx context.Context, name, spanType string, opts ...StartSpanOption) (context.Context, Span) {
	return getDefaultClient().StartSpan(ctx, name, spanType, opts...)
}

// GetSpanFromContext Get the span from the context.
func GetSpanFromContext(ctx context.Context) Span {
	return getDefaultClient().GetSpanFromContext(ctx)
}

// GetSpanFromHeader Get the span from the header.
func GetSpanFromHeader(ctx context.Context, header map[string]string) SpanContext {
	return getDefaultClient().GetSpanFromHeader(ctx, header)
}

// Flush Force the reporting of spans in the queue.
func Flush(ctx context.Context) {
	getDefaultClient().Flush(ctx)
}

func buildOptionsFromEnv(opts *options) {
	if baseURL := os.Getenv(EnvApiBaseURL); baseURL != "" {
		opts.apiBaseURL = baseURL
	}
	if workspaceID := os.Getenv(EnvWorkspaceID); workspaceID != "" {
		opts.workspaceID = workspaceID
	}

	if apiToken := os.Getenv(EnvApiToken); apiToken != "" {
		opts.apiToken = apiToken
	}
	if jwtOAuthClientID := os.Getenv(EnvJwtOAuthClientID); jwtOAuthClientID != "" {
		opts.jwtOAuthClientID = jwtOAuthClientID
	}
	if jwtOAuthPrivateKey := os.Getenv(EnvJwtOAuthPrivateKey); jwtOAuthPrivateKey != "" {
		opts.jwtOAuthPrivateKey = jwtOAuthPrivateKey
	}
	if jwtOAuthPublicKeyID := os.Getenv(EnvJwtOAuthPublicKeyID); jwtOAuthPublicKeyID != "" {
		opts.jwtOAuthPublicKeyID = jwtOAuthPublicKeyID
	}
}

func checkOptions(opts *options) error {
	if opts.apiBaseURL == "" {
		return ErrInvalidParam.Wrap(errors.New("apiBaseURL is required"))
	}
	if opts.workspaceID == "" {
		return ErrInvalidParam.Wrap(errors.New("workspaceID is required"))
	}
	if opts.httpClient == nil {
		return ErrInvalidParam.Wrap(errors.New("httpClient is required"))
	}
	if opts.promptCacheMaxCount < 0 {
		opts.promptCacheMaxCount = consts.DefaultPromptCacheMaxCount
	}
	if opts.promptCacheRefreshInterval < 0 {
		opts.promptCacheRefreshInterval = consts.DefaultPromptCacheRefreshInterval
	}
	return nil
}

func buildAuth(opts options) (httpclient.Auth, error) {
	if opts.jwtOAuthClientID != "" && opts.jwtOAuthPrivateKey != "" && opts.jwtOAuthPublicKeyID != "" {
		oauthClient, err := httpclient.NewJWTOAuthClient(httpclient.NewJWTOAuthClientParam{
			ClientID:      opts.jwtOAuthClientID,
			PublicKey:     opts.jwtOAuthPublicKeyID,
			PrivateKeyPEM: opts.jwtOAuthPrivateKey,
		}, httpclient.WithAuthBaseURL(opts.apiBaseURL), httpclient.WithAuthHttpClient(opts.httpClient))
		if err != nil {
			return nil, err
		}
		return httpclient.NewJWTAuth(oauthClient, nil), nil
	}
	if opts.apiToken != "" {
		return httpclient.NewTokenAuth(opts.apiToken), nil
	}
	return nil, ErrAuthInfoRequired
}

func getDefaultClient() Client {
	once.Do(func() {
		var err error
		defaultClient, err = NewClient()
		if err != nil {
			defaultClient = &noopClient{newClientError: err}
		} else {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				sig := <-sigChan
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				logger.CtxInfof(ctx, "Received signal: %v, starting graceful shutdown...", sig)
				defaultClient.Close(ctx)
				defaultClient = &noopClient{newClientError: consts.ErrClientClosed}
				logger.CtxInfof(ctx, "Graceful shutdown finished.")
				os.Exit(0)
			}()
		}
	})
	return defaultClient
}

var (
	defaultClient Client
	once          sync.Once
	clientCache   sync.Map // client cache to avoid creating multiple clients with the same options
)

type loopClient struct {
	traceProvider  *trace.Provider
	promptProvider *prompt.Provider

	workspaceID string

	closed bool
}

func (c *loopClient) GetWorkspaceID() string {
	return c.workspaceID
}

func (c *loopClient) Close(ctx context.Context) {
	if c.closed {
		return
	}
	c.traceProvider.CloseTrace(ctx)
	c.closed = true
}

func (c *loopClient) GetPrompt(ctx context.Context, param GetPromptParam, options ...GetPromptOption) (*entity.Prompt, error) {
	if c.closed {
		return nil, consts.ErrClientClosed
	}
	config := prompt.GetPromptOptions{}
	for _, opt := range options {
		opt(&config)
	}
	return c.promptProvider.GetPrompt(ctx, param, config)
}

func (c *loopClient) PromptFormat(ctx context.Context, loopPrompt *entity.Prompt, variables map[string]any, options ...PromptFormatOption) (messages []*entity.Message, err error) {
	if c.closed {
		return nil, consts.ErrClientClosed
	}
	config := prompt.PromptFormatOptions{}
	for _, opt := range options {
		opt(&config)
	}
	return c.promptProvider.PromptFormat(ctx, loopPrompt, variables, config)
}

func (c *loopClient) StartSpan(ctx context.Context, name, spanType string, opts ...StartSpanOption) (context.Context, Span) {
	if c.closed {
		return nil, defaultNoopSpan
	}
	config := trace.StartSpanOptions{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(&config)
	}
	ctx, span, err := c.traceProvider.StartSpan(ctx, name, spanType, config)
	if err != nil {
		logger.CtxWarnf(ctx, "start span failed, return noop span. %v", err)
		return ctx, defaultNoopSpan
	}
	return ctx, span
}

func (c *loopClient) GetSpanFromContext(ctx context.Context) Span {
	if c.closed {
		return defaultNoopSpan
	}
	span := c.traceProvider.GetSpanFromContext(ctx)
	if span == nil {
		return defaultNoopSpan
	}
	return span
}

func (c *loopClient) GetSpanFromHeader(ctx context.Context, header map[string]string) SpanContext {
	if c.closed {
		return defaultNoopSpan
	}
	return c.traceProvider.GetSpanFromHeader(ctx, header)
}

func (c *loopClient) Flush(ctx context.Context) {
	if c.closed {
		return
	}
	c.traceProvider.Flush(ctx)
}
