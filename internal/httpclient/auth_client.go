// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package httpclient

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coze-dev/cozeloop-go/internal/consts"
	"github.com/coze-dev/cozeloop-go/internal/logger"
	"github.com/coze-dev/cozeloop-go/internal/util"
	"github.com/golang-jwt/jwt"
)

// Scope represents the OAuth scope
type Scope struct {
	AccountPermission   *ScopeAccountPermission   `json:"account_permission"`
	AttributeConstraint *ScopeAttributeConstraint `json:"attribute_constraint,omitempty"`
}

// ScopeAccountPermission represents the account permissions in the scope
type ScopeAccountPermission struct {
	PermissionList []string `json:"permission_list"`
}

// ScopeAttributeConstraint represents the attribute constraints in the scope
type ScopeAttributeConstraint struct {
	ConnectorBotChatAttribute *ScopeAttributeConstraintConnectorBotChatAttribute `json:"connector_bot_chat_attribute"`
}

// ScopeAttributeConstraintConnectorBotChatAttribute represents the bot chat attributes
type ScopeAttributeConstraintConnectorBotChatAttribute struct {
	BotIDList []string `json:"bot_id_list"`
}

// getAccessTokenReq represents the access token request
type getAccessTokenReq struct {
	ClientID        string        `json:"client_id"`
	Code            string        `json:"code,omitempty"`
	GrantType       string        `json:"grant_type"`
	RedirectURI     string        `json:"redirect_uri,omitempty"`
	RefreshToken    string        `json:"refresh_token,omitempty"`
	CodeVerifier    string        `json:"code_verifier,omitempty"`
	DeviceCode      string        `json:"device_code,omitempty"`
	DurationSeconds time.Duration `json:"duration_seconds,omitempty"`
	Scope           *Scope        `json:"scope,omitempty"`
	LogID           string        `json:"log_id,omitempty"`
	AccountID       *int64        `json:"account_id,omitempty"`
}

// GrantType represents the OAuth grant type
type GrantType string

const (
	GrantTypeAuthorizationCode GrantType = "authorization_code"
	GrantTypeDeviceCode        GrantType = "urn:ietf:params:oauth:grant-type:device_code"
	GrantTypeJWTCode           GrantType = "urn:ietf:params:oauth:grant-type:jwt-bearer"
	GrantTypeRefreshToken      GrantType = "refresh_token"
)

func (r GrantType) String() string {
	return string(r)
}

// urlOption represents URL option function type
type urlOption func(*url.Values)

// withCodeChallenge adds code_challenge parameter
func withCodeChallenge(challenge string) urlOption {
	return func(v *url.Values) {
		v.Set("code_challenge", challenge)
	}
}

// withCodeChallengeMethod adds code_challenge_method parameter
func withCodeChallengeMethod(method string) urlOption {
	return func(v *url.Values) {
		v.Set("code_challenge_method", method)
	}
}

// OAuthToken represents the OAuth token response
type OAuthToken struct {
	BaseResponse
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// OAuthClient represents the base OAuth core structure
type OAuthClient struct {
	httpClient   HTTPClient
	clientID     string
	clientSecret string
	baseURL      string
	wwwURL       string
	hostName     string
}

const (
	getTokenPath               = "/api/permission/oauth2/token"
	getAccountTokenPath        = "/api/permission/oauth2/account/%d/token"
	getDeviceCodePath          = "/api/permission/oauth2/device/code"
	getWorkspaceDeviceCodePath = "/api/permission/oauth2/workspace_id/%s/device/code"
)

type oauthOption struct {
	baseURL    string
	wwwURL     string
	httpClient HTTPClient
}

type OAuthClientOption func(*oauthOption)

// WithAuthBaseURL adds base URL
func WithAuthBaseURL(baseURL string) OAuthClientOption {
	return func(opt *oauthOption) {
		opt.baseURL = baseURL
	}
}

// WithAuthWWWURL adds base URL
func WithAuthWWWURL(wwwURL string) OAuthClientOption {
	return func(opt *oauthOption) {
		opt.wwwURL = wwwURL
	}
}

func WithAuthHttpClient(client HTTPClient) OAuthClientOption {
	return func(opt *oauthOption) {
		opt.httpClient = client
	}
}

// newOAuthClient creates a new OAuth core
func newOAuthClient(clientID, clientSecret string, opts ...OAuthClientOption) (*OAuthClient, error) {
	initSettings := &oauthOption{
		baseURL: consts.CnBaseURL,
	}

	for _, opt := range opts {
		opt(initSettings)
	}

	var hostName string
	if initSettings.baseURL != "" {
		parsedURL, err := url.Parse(initSettings.baseURL)
		if err != nil {
			return nil, consts.ErrInvalidParam.Wrap(fmt.Errorf("invalid api base url: %v", err))
		}
		hostName = parsedURL.Host
	} else {
		return nil, consts.ErrInvalidParam.Wrap(fmt.Errorf("invalid api base url"))
	}
	var httpClient HTTPClient
	if initSettings.httpClient != nil {
		httpClient = initSettings.httpClient
	} else {
		httpClient = http.DefaultClient
	}

	if initSettings.wwwURL == "" {
		initSettings.wwwURL = strings.Replace(initSettings.baseURL, "api.", "www.", 1)
	}

	return &OAuthClient{
		httpClient:   httpClient,
		clientID:     clientID,
		clientSecret: clientSecret,
		baseURL:      initSettings.baseURL,
		wwwURL:       initSettings.wwwURL,
		hostName:     hostName,
	}, nil
}

// getOAuthURL generates OAuth URL
func (c *OAuthClient) getOAuthURL(redirectURI, state string, opts ...urlOption) string {
	params := url.Values{}
	params.Set("response_type", "code")
	if c.clientID != "" {
		params.Set("client_id", c.clientID)
	}
	if redirectURI != "" {
		params.Set("redirect_uri", redirectURI)
	}
	if state != "" {
		params.Set("state", state)
	}

	for _, opt := range opts {
		opt(&params)
	}

	uri := c.wwwURL + "/api/permission/oauth2/authorize"
	return uri + "?" + params.Encode()
}

// getWorkspaceOAuthURL generates OAuth URL with workspace
func (c *OAuthClient) getWorkspaceOAuthURL(redirectURI, state, workspaceID string, opts ...urlOption) string {
	params := url.Values{}
	params.Set("response_type", "code")
	if c.clientID != "" {
		params.Set("client_id", c.clientID)
	}
	if redirectURI != "" {
		params.Set("redirect_uri", redirectURI)
	}
	if state != "" {
		params.Set("state", state)
	}

	for _, opt := range opts {
		opt(&params)
	}

	uri := fmt.Sprintf("%s/api/permission/oauth2/workspace_id/%s/authorize", c.wwwURL, workspaceID)
	return uri + "?" + params.Encode()
}

type getAccessTokenParams struct {
	Type         GrantType
	Code         string
	Secret       string
	RedirectURI  string
	RefreshToken string
	Request      *getAccessTokenReq
}

func (c *OAuthClient) getAccessToken(ctx context.Context, params getAccessTokenParams) (*OAuthToken, error) {
	// If Request is provided, use it directly
	result := &OAuthToken{}
	var req *getAccessTokenReq
	if params.Request != nil {
		req = params.Request
	} else {
		req = &getAccessTokenReq{
			ClientID:     c.clientID,
			GrantType:    params.Type.String(),
			Code:         params.Code,
			RefreshToken: params.RefreshToken,
			RedirectURI:  params.RedirectURI,
		}
	}

	path := getTokenPath
	if req.AccountID != nil && *req.AccountID > 0 {
		path = fmt.Sprintf(getAccountTokenPath, *req.AccountID)
	}
	header := map[string]string{
		"Content-Type":         "application/json",
		consts.AuthorizeHeader: fmt.Sprintf("Bearer %s", params.Secret),
	}

	if err := defaultBackoff.Retry(ctx, func() error {
		return c.doPost(ctx, path, req, result, header)
	}, 3); err != nil {
		logger.CtxErrorf(ctx, "get access token failed: %v", err)
		return nil, err
	}
	return result, nil
}

// refreshAccessToken is a convenience method that internally calls getAccessToken
func (c *OAuthClient) refreshAccessToken(ctx context.Context, refreshToken string) (*OAuthToken, error) {
	return c.getAccessToken(ctx, getAccessTokenParams{
		Type:         GrantTypeRefreshToken,
		RefreshToken: refreshToken,
	})
}

// refreshAccessToken is a convenience method that internally calls getAccessToken
func (c *OAuthClient) refreshAccessTokenWithClientSecret(ctx context.Context, refreshToken string) (*OAuthToken, error) {
	return c.getAccessToken(ctx, getAccessTokenParams{
		Secret:       c.clientSecret,
		Type:         GrantTypeRefreshToken,
		RefreshToken: refreshToken,
	})
}

func (c *OAuthClient) doPost(ctx context.Context, path string, body any, resp OpenAPIResponse, headers map[string]string) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.baseURL + path
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bodyReader)
	if err != nil {
		return consts.ErrInternal.Wrap(err)
	}
	for k, v := range headers {
		request.Header.Add(k, v)
	}
	setUserAgent(request)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return consts.ErrRemoteService.Wrap(err)
	}

	return parseResponse(ctx, url, response, resp)
}

// JWTOAuthClient represents the JWT OAuth core
type JWTOAuthClient struct {
	*OAuthClient
	ttl        time.Duration
	privateKey *rsa.PrivateKey
	publicKey  string
}

type NewJWTOAuthClientParam struct {
	ClientID      string
	PublicKey     string
	PrivateKeyPEM string
	TTL           time.Duration
}

// NewJWTOAuthClient creates a new JWT OAuth core
func NewJWTOAuthClient(param NewJWTOAuthClientParam, opts ...OAuthClientOption) (*JWTOAuthClient, error) {
	privateKey, err := parsePrivateKey(param.PrivateKeyPEM)
	if err != nil {
		return nil, consts.ErrParsePrivateKey.Wrap(err)
	}
	client, err := newOAuthClient(param.ClientID, "", opts...)
	if err != nil {
		return nil, err
	}
	ttl := param.TTL
	if ttl < consts.OAuthRefreshAdvanceTime {
		ttl = consts.DefaultOAuthRefreshTTL // Default 15 minutes
	}
	jwtClient := &JWTOAuthClient{
		OAuthClient: client,
		ttl:         ttl,
		privateKey:  privateKey,
		publicKey:   param.PublicKey,
	}

	return jwtClient, nil
}

// GetJWTAccessTokenReq represents options for getting JWT OAuth token
type GetJWTAccessTokenReq struct {
	TTL         time.Duration `json:"ttl,omitempty"`          // Token validity period (in seconds)
	Scope       *Scope        `json:"scope,omitempty"`        // Permission scope
	SessionName *string       `json:"session_name,omitempty"` // Session name
	AccountID   *int64        `json:"account_id,omitempty"`   // Account ID
}

// GetAccessToken gets the access token, using options pattern
func (c *JWTOAuthClient) GetAccessToken(ctx context.Context, opts *GetJWTAccessTokenReq) (*OAuthToken, error) {
	if opts == nil {
		opts = &GetJWTAccessTokenReq{}
	}

	ttl := c.ttl
	if opts.TTL > 0 {
		ttl = opts.TTL
	}

	jwtCode, err := c.generateJWT(ttl, opts.SessionName)
	if err != nil {
		return nil, err
	}

	req := getAccessTokenParams{
		Type:   GrantTypeJWTCode,
		Secret: jwtCode,
		Request: &getAccessTokenReq{
			ClientID:        c.clientID,
			GrantType:       string(GrantTypeJWTCode),
			DurationSeconds: ttl / time.Second,
			Scope:           opts.Scope,
			AccountID:       opts.AccountID,
		},
	}
	return c.getAccessToken(ctx, req)
}

func (c *JWTOAuthClient) generateJWT(ttl time.Duration, sessionName *string) (string, error) {
	now := time.Now()
	jti, err := util.GenerateRandomString(16)
	if err != nil {
		return "", err
	}

	// Build claims
	claims := jwt.MapClaims{
		"iss": c.clientID,
		"aud": c.hostName,
		"iat": now.Unix(),
		"exp": now.Add(ttl).Unix(),
		"jti": jti,
	}

	// If session_name is provided, add it to claims
	if sessionName != nil {
		claims["session_name"] = *sessionName
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	// Set header
	token.Header["kid"] = c.publicKey
	token.Header["typ"] = "JWT"
	token.Header["alg"] = "RS256"

	// Sign and get full token string
	tokenString, err := token.SignedString(c.privateKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// tool function
func parsePrivateKey(privateKeyPEM string) (*rsa.PrivateKey, error) {
	// Remove PEM header and footer and whitespace
	privateKeyPEM = strings.ReplaceAll(privateKeyPEM, "-----BEGIN PRIVATE KEY-----", "")
	privateKeyPEM = strings.ReplaceAll(privateKeyPEM, "-----END PRIVATE KEY-----", "")
	privateKeyPEM = strings.ReplaceAll(privateKeyPEM, "\\n", "\n")
	privateKeyPEM = strings.ReplaceAll(privateKeyPEM, "\n", "")
	privateKeyPEM = strings.ReplaceAll(privateKeyPEM, "\r", "")
	privateKeyPEM = strings.ReplaceAll(privateKeyPEM, " ", "")

	// Decode Base64
	block, err := base64.StdEncoding.DecodeString(privateKeyPEM)
	if err != nil {
		return nil, err
	}

	// Parse PKCS8 private key
	key, err := x509.ParsePKCS8PrivateKey(block)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not RSA")
	}

	return rsaKey, nil
}
