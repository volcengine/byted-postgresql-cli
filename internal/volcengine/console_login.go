// Copyright (c) 2026 Beijing Volcano Engine Technology Co., Ltd.
// SPDX-License-Identifier: MIT
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Browser-based Volcengine Console Login (OAuth 2.0 + PKCE). This is the
// primary way end users authenticate the CLI: `login` opens the Volcengine
// signin page, exchanges the authorization code for temporary STS credentials,
// and caches them locally under ~/.volcengine/login/cache. Control-plane calls
// then resolve those cached STS credentials (see ResolveConfig -> the
// ModeConsoleLogin branch) and refresh them automatically before expiry.
//
// The on-disk config store (~/.volcengine/config.json) and login cache layout
// are shared with the sibling Volcengine CLIs, so a login performed by any of
// them is reused here.
package volcengine

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"golang.org/x/term"
)

const (
	// DefaultConsoleLoginRegion is the region assumed when the user does not
	// pick one during login.
	DefaultConsoleLoginRegion = "cn-beijing"
	// DefaultConsoleEndpoint is the Volcengine signin service base URL.
	DefaultConsoleEndpoint = "https://signin.volcengine.com"
	// ScopeAllAll is the OAuth scope requested for full console access.
	ScopeAllAll = "Console:All:All"

	consoleAuthorizePath       = "/authorize/oauth/authorize"
	consoleTokenPath           = "/authorize/oauth/token"
	consoleClientIDSameDevice  = "trn:signin:::devtools/same-device"
	consoleClientIDCrossDevice = "trn:signin:::devtools/cross-device"
	loginCacheDirectoryEnv     = "VOLCENGINE_LOGIN_CACHE_DIRECTORY"

	remoteCodeFilePollInterval = 1 * time.Second
	remoteCodeFilePollTimeout  = 10 * time.Minute
)

// ConsoleLoginParams configures a single `login` invocation.
type ConsoleLoginParams struct {
	Provider       Provider
	Profile        string
	Region         string
	Remote         bool
	EndpointURL    string
	SkipRegion     bool
	NoOpen         bool
	CredentialFile string
	// AssumeYes skips the interactive confirmation prompt when a login would
	// replace an existing login_session on the profile.
	AssumeYes bool
}

// ConsoleLogoutParams configures a single `logout` invocation.
type ConsoleLogoutParams struct {
	Provider Provider
	Profile  string
	All      bool
}

// LoginTokenCache is the on-disk cached OAuth token bundle for one
// login_session.
type LoginTokenCache struct {
	LoginSession string          `json:"login_session"`
	AccessToken  json.RawMessage `json:"access_token"`
	RefreshToken string          `json:"refresh_token,omitempty"`
	IDToken      string          `json:"id_token,omitempty"`
	Scope        string          `json:"scope"`
	ClientID     string          `json:"client_id"`
	EndpointURL  string          `json:"endpoint_url,omitempty"`
	IssuedAt     string          `json:"issued_at"`
	ExpiresIn    int             `json:"expires_in"`
	TokenType    string          `json:"token_type"`
}

// STSCredentials is the temporary AK/SK + session token minted by Console
// Login and embedded (JSON-encoded) in the OAuth access_token.
type STSCredentials struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	SessionToken    string `json:"session_token"`
}

// ConsoleTokenResponse is the OAuth token endpoint response.
type ConsoleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token"`
}

type consoleTokenRequest struct {
	GrantType    string
	Code         string
	RedirectURI  string
	ClientID     string
	Scope        string
	CodeVerifier string
	RefreshToken string
}

// ConsoleOAuthErrorResponse is the OAuth error payload from the signin service.
type ConsoleOAuthErrorResponse struct {
	State            string `json:"state,omitempty"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	ErrorURI         string `json:"error_uri,omitempty"`
}

// ConsoleOAuthAPIError wraps a non-2xx OAuth response so the caller can see the
// signin service's own message.
type ConsoleOAuthAPIError struct {
	StatusCode int
	Response   ConsoleOAuthErrorResponse
	RawBody    string
	RequestID  string
}

func (e *ConsoleOAuthAPIError) Error() string {
	if e == nil {
		return ""
	}
	var parts []string
	if e.Response.Error != "" {
		parts = append(parts, e.Response.Error)
	}
	if e.Response.ErrorDescription != "" {
		parts = append(parts, e.Response.ErrorDescription)
	}
	msg := strings.Join(parts, ": ")
	if msg == "" {
		msg = e.RawBody
	}
	if msg == "" {
		msg = "unknown error"
	}
	suffix := fmt.Sprintf("[status %d", e.StatusCode)
	if e.RequestID != "" {
		suffix += ", requestId: " + e.RequestID
	}
	suffix += "]"
	return fmt.Sprintf("console oauth request failed: %s %s", msg, suffix)
}

// RunConsoleLogin drives the interactive browser login flow (or credential-file
// import) and persists a console-login profile plus its cached STS token.
func RunConsoleLogin(ctx context.Context, params ConsoleLoginParams, input io.Reader, output io.Writer) error {
	if params.Profile == "" {
		params.Profile = "default"
	}
	if input == nil {
		input = os.Stdin
	}
	if output == nil {
		output = os.Stdout
	}
	params.CredentialFile = strings.TrimSpace(params.CredentialFile)
	if params.CredentialFile != "" && params.Remote {
		return errors.New("--credential-file cannot be combined with --remote")
	}
	if params.SkipRegion && strings.TrimSpace(params.Region) != "" {
		return errors.New("--skip-region cannot be combined with --region")
	}
	if params.SkipRegion {
		params.Region = ""
	} else {
		region, err := resolveConsoleLoginRegion(input, output, params.Region)
		if err != nil {
			return fmt.Errorf("resolving login region: %w", err)
		}
		params.Region = region
	}
	if params.Provider == "" {
		params.Provider = ProviderVolcengine
	}
	if err := ValidateProviderRegion(params.Provider, params.Region); err != nil {
		return fmt.Errorf("resolving login region: %w", err)
	}
	spec, err := ProviderSpecFor(params.Provider)
	if err != nil {
		return err
	}
	if params.EndpointURL == "" {
		params.EndpointURL = spec.ConsoleEndpoint
	}
	fmt.Fprintf(output, "Selected provider: %s\nRegion: %s\nEndpoint: %s\nConsole: %s\n",
		params.Provider, params.Region, spec.AIDAPEndpoint(params.Region), params.EndpointURL)

	var cache *LoginTokenCache
	if params.CredentialFile != "" {
		cache, err = importConsoleLoginCredentialFile(ctx, params, input, output)
	} else {
		cache, err = runConsoleOAuthLogin(ctx, params, input, output)
	}
	if err != nil {
		return err
	}
	fmt.Fprintln(output, "\nSuccessfully logged in!")
	fmt.Fprintf(output, "Credentials cached for profile: %s\n", params.Profile)
	issuedAt, _ := time.Parse(time.RFC3339, cache.IssuedAt)
	expiresAt := issuedAt.Add(time.Duration(cache.ExpiresIn) * time.Second)
	fmt.Fprintf(output, "STS credentials expire at: %s\n", expiresAt.Local().Format("2006-01-02 15:04:05"))
	return nil
}

func runConsoleOAuthLogin(ctx context.Context, params ConsoleLoginParams, input io.Reader, output io.Writer) (*LoginTokenCache, error) {
	clientID := consoleClientIDSameDevice
	if params.Remote {
		clientID = consoleClientIDCrossDevice
	}
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return nil, err
	}
	codeChallenge := generateCodeChallenge(codeVerifier)
	state, err := generateState()
	if err != nil {
		return nil, err
	}
	oauthClient := newConsoleOAuthClient(params.EndpointURL)
	var authCode, redirectURI string
	if params.Remote {
		authCode, redirectURI, err = remoteAuthorize(ctx, input, output, oauthClient, clientID, codeChallenge, state, params.EndpointURL)
	} else {
		authCode, redirectURI, err = localAuthorize(ctx, output, oauthClient, clientID, codeChallenge, state, !params.NoOpen)
	}
	if err != nil {
		return nil, err
	}
	return completeConsoleLogin(ctx, completeConsoleLoginParams{
		OAuthClient:  oauthClient,
		AuthCode:     authCode,
		RedirectURI:  redirectURI,
		ClientID:     clientID,
		CodeVerifier: codeVerifier,
		Profile:      params.Profile,
		Provider:     params.Provider,
		Region:       params.Region,
		EndpointURL:  params.EndpointURL,
		Input:        input,
		Output:       output,
		Confirm:      true,
		AssumeYes:    params.AssumeYes,
	})
}

type completeConsoleLoginParams struct {
	OAuthClient  *consoleOAuthClient
	AuthCode     string
	RedirectURI  string
	ClientID     string
	CodeVerifier string
	Profile      string
	Provider     Provider
	Region       string
	EndpointURL  string
	Input        io.Reader
	Output       io.Writer
	Confirm      bool
	AssumeYes    bool
}

func completeConsoleLogin(ctx context.Context, params completeConsoleLoginParams) (*LoginTokenCache, error) {
	if params.Provider == "" {
		params.Provider = ProviderVolcengine
	}
	output := params.Output
	if output == nil {
		output = io.Discard
	}
	tokenResp, err := params.OAuthClient.exchangeToken(ctx, &consoleTokenRequest{
		GrantType:    "authorization_code",
		Code:         params.AuthCode,
		RedirectURI:  params.RedirectURI,
		ClientID:     params.ClientID,
		Scope:        ScopeAllAll,
		CodeVerifier: params.CodeVerifier,
	})
	if err != nil {
		return nil, fmt.Errorf("exchanging authorization code for token: %w", err)
	}
	if _, err := ParseSTSCredentials(tokenResp.AccessToken); err != nil {
		return nil, fmt.Errorf("parsing STS credentials: %w", err)
	}
	loginSession, err := extractLoginSession(tokenResp.IDToken)
	if err != nil {
		return nil, fmt.Errorf("extracting login session from id_token: %w", err)
	}
	cfg, err := LoadFileConfigFor(params.Provider)
	if err != nil {
		return nil, err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]*Profile{}
	}
	profile := cfg.Profiles[params.Profile]
	if profile == nil {
		profile = &Profile{Name: params.Profile}
	}
	oldLoginSession := profile.LoginSession
	if params.Confirm && profile.LoginSession != "" && profile.LoginSession != loginSession {
		confirmed, err := confirmLoginSessionReplacement(params.Input, output, params.Profile, profile.LoginSession, loginSession, params.AssumeYes)
		if err != nil {
			return nil, err
		}
		if !confirmed {
			return nil, errors.New("login canceled: existing login_session was not replaced")
		}
	}
	cache := &LoginTokenCache{
		LoginSession: loginSession,
		AccessToken:  json.RawMessage(tokenResp.AccessToken),
		RefreshToken: tokenResp.RefreshToken,
		IDToken:      tokenResp.IDToken,
		Scope:        ScopeAllAll,
		ClientID:     params.ClientID,
		EndpointURL:  params.EndpointURL,
		IssuedAt:     time.Now().UTC().Format(time.RFC3339),
		ExpiresIn:    tokenResp.ExpiresIn,
		TokenType:    tokenResp.TokenType,
	}
	if err := writeLoginCacheFor(params.Provider, cache); err != nil {
		return nil, fmt.Errorf("writing login cache: %w", err)
	}
	if oldLoginSession != "" && oldLoginSession != loginSession {
		_ = removeLoginCacheFor(params.Provider, oldLoginSession)
	}
	cfg.Profiles[params.Profile] = &Profile{
		Name:         params.Profile,
		Mode:         ModeConsoleLogin,
		Region:       params.Region,
		LoginSession: loginSession,
		Provider:     params.Provider,
	}
	cfg.Current = params.Profile
	if err := SaveFileConfigFor(params.Provider, cfg); err != nil {
		return nil, err
	}
	return cache, nil
}

func importConsoleLoginCredentialFile(ctx context.Context, params ConsoleLoginParams, input io.Reader, output io.Writer) (*LoginTokenCache, error) {
	if params.Provider == "" {
		params.Provider = ProviderVolcengine
	}
	data, err := os.ReadFile(params.CredentialFile)
	if err != nil {
		return nil, fmt.Errorf("reading credential file: %w", err)
	}
	var cache LoginTokenCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("parsing credential file: %w", err)
	}
	cache.LoginSession = strings.TrimSpace(cache.LoginSession)
	cache.ClientID = strings.TrimSpace(cache.ClientID)
	cache.Scope = strings.TrimSpace(cache.Scope)
	cache.EndpointURL = strings.TrimSpace(cache.EndpointURL)
	cache.IssuedAt = strings.TrimSpace(cache.IssuedAt)
	if cache.LoginSession == "" {
		return nil, errors.New("credential file missing login_session")
	}
	if _, err := parseCachedSTSCredentials(&cache); err != nil {
		return nil, fmt.Errorf("invalid credential file access_token: %w", err)
	}
	issuedAt, err := time.Parse(time.RFC3339, cache.IssuedAt)
	if err != nil {
		return nil, fmt.Errorf("invalid credential file issued_at %q: %w", cache.IssuedAt, err)
	}
	if cache.ExpiresIn <= 0 {
		return nil, errors.New("credential file expires_in must be greater than 0")
	}
	expiration := issuedAt.Add(time.Duration(cache.ExpiresIn) * time.Second)
	if !time.Now().UTC().Before(expiration.Add(-60 * time.Second)) {
		if cache.RefreshToken == "" {
			return nil, errors.New("credential file session is expired or near expiry and has no refresh_token")
		}
		if err := refreshLoginCache(ctx, params.Provider, &cache); err != nil {
			return nil, fmt.Errorf("refreshing imported session: %w", err)
		}
	}

	cfg, err := LoadFileConfigFor(params.Provider)
	if err != nil {
		return nil, err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]*Profile{}
	}
	profile := cfg.Profiles[params.Profile]
	oldLoginSession := ""
	if profile != nil {
		oldLoginSession = profile.LoginSession
	}
	if oldLoginSession != "" && oldLoginSession != cache.LoginSession {
		confirmed, err := confirmLoginSessionReplacement(input, output, params.Profile, oldLoginSession, cache.LoginSession, params.AssumeYes)
		if err != nil {
			return nil, err
		}
		if !confirmed {
			return nil, errors.New("login canceled: existing login_session was not replaced")
		}
	}
	if err := writeLoginCacheFor(params.Provider, &cache); err != nil {
		return nil, fmt.Errorf("writing imported login cache: %w", err)
	}
	cfg.Profiles[params.Profile] = &Profile{
		Name:         params.Profile,
		Mode:         ModeConsoleLogin,
		Region:       params.Region,
		LoginSession: cache.LoginSession,
		Provider:     params.Provider,
	}
	cfg.Current = params.Profile
	if err := SaveFileConfigFor(params.Provider, cfg); err != nil {
		return nil, err
	}
	if oldLoginSession != "" && oldLoginSession != cache.LoginSession {
		_ = removeLoginCacheFor(params.Provider, oldLoginSession)
	}
	return &cache, nil
}

// RunConsoleLogout removes cached Console Login credentials and deletes
// console-login profiles. It never touches AK/SK profiles.
func RunConsoleLogout(params ConsoleLogoutParams, output io.Writer) error {
	if output == nil {
		output = os.Stdout
	}
	if params.Provider == "" {
		params.Provider = ProviderVolcengine
	}
	if params.All {
		return logoutAllConsoleProfiles(params.Provider, output)
	}
	profileName := params.Profile
	if profileName == "" {
		profileName = "default"
	}
	cfg, err := LoadFileConfigFor(params.Provider)
	if err != nil {
		return err
	}
	profile, ok := cfg.Profiles[profileName]
	if !ok || profile == nil {
		return fmt.Errorf("profile %q not found in configuration", profileName)
	}
	if profile.Mode != ModeConsoleLogin {
		return fmt.Errorf("profile %q is using %q mode, not %q mode. Only console-login profiles can be logged out with this command", profileName, profile.Mode, ModeConsoleLogin)
	}
	if profile.LoginSession == "" {
		fmt.Fprintf(output, "Profile %q does not have an active login session. Nothing to do.\n", profileName)
		return nil
	}
	if err := removeLoginCacheFor(params.Provider, profile.LoginSession); err != nil {
		return fmt.Errorf("removing cached token for profile %q: %w", profileName, err)
	}
	delete(cfg.Profiles, profileName)
	normalizeCurrentProfileAfterDelete(&cfg, profileName)
	if err := SaveFileConfigFor(params.Provider, cfg); err != nil {
		return err
	}
	fmt.Fprintf(output, "Successfully logged out and deleted console-login profile %q.\n", profileName)
	printPostLogoutHint(output)
	return nil
}

func logoutAllConsoleProfiles(provider Provider, output io.Writer) error {
	cfg, err := LoadFileConfigFor(provider)
	if err != nil {
		return err
	}
	deletedCount := 0
	var firstErr error
	for name, profile := range cfg.Profiles {
		if profile == nil || profile.Mode != ModeConsoleLogin || profile.LoginSession == "" {
			continue
		}
		if err := removeLoginCacheFor(provider, profile.LoginSession); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove cache for profile %q: %v\n", name, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		delete(cfg.Profiles, name)
		normalizeCurrentProfileAfterDelete(&cfg, name)
		deletedCount++
		fmt.Fprintf(output, "  Logged out and deleted profile %q\n", name)
	}
	if err := SaveFileConfigFor(provider, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update config after logout: %v\n", err)
	} else if deletedCount > 0 {
		fmt.Fprintf(output, "\nSuccessfully logged out %d console-login profile(s).\n", deletedCount)
		printPostLogoutHint(output)
	} else {
		fmt.Fprintln(output, "No console-login profiles with active sessions found. Nothing to do.")
	}
	return firstErr
}

func normalizeCurrentProfileAfterDelete(cfg *FileConfig, deletedProfile string) {
	if cfg == nil || cfg.Current != deletedProfile {
		return
	}
	cfg.Current = "default"
	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) > 0 {
		cfg.Current = names[0]
	}
}

// EnsureValidLoginToken returns fresh STS credentials for a console-login
// profile, transparently refreshing them (and rewriting the cache) when they
// are within 60s of expiry.
func EnsureValidLoginToken(provider Provider, cfg FileConfig, profileName string) (*STSCredentials, error) {
	profile, ok := cfg.Profiles[profileName]
	if !ok || profile == nil {
		return nil, fmt.Errorf("profile %q not found in config", profileName)
	}
	if profile.LoginSession == "" {
		return nil, fmt.Errorf("profile %q does not have a login_session; run `byted-postgresql-cli login --profile %s` first", profileName, profileName)
	}
	cache, err := readLoginCacheFor(provider, profile.LoginSession)
	if err != nil {
		return nil, errors.New("no active session. Please run `byted-postgresql-cli login` first")
	}
	creds, err := parseCachedSTSCredentials(cache)
	if err != nil {
		return nil, fmt.Errorf("parsing cached STS credentials: %w", err)
	}
	issuedAt, err := time.Parse(time.RFC3339, cache.IssuedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing issued_at %q: %w", cache.IssuedAt, err)
	}
	expiration := issuedAt.Add(time.Duration(cache.ExpiresIn) * time.Second)
	if time.Now().UTC().Before(expiration.Add(-60 * time.Second)) {
		return creds, nil
	}
	if cache.RefreshToken == "" {
		return nil, errors.New("no refresh token available. Session expired. Please run `byted-postgresql-cli login` to re-authenticate")
	}
	if err := refreshLoginCache(context.Background(), provider, cache); err != nil {
		return nil, fmt.Errorf("failed to refresh session token. Please run `byted-postgresql-cli login` to re-authenticate: %w", err)
	}
	newCreds, err := parseCachedSTSCredentials(cache)
	if err != nil {
		return nil, fmt.Errorf("parsing refreshed STS credentials: %w", err)
	}
	if err := writeLoginCacheFor(provider, cache); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update login cache: %v\n", err)
	}
	return newCreds, nil
}

func parseCachedSTSCredentials(cache *LoginTokenCache) (*STSCredentials, error) {
	if cache == nil {
		return nil, errors.New("login cache is nil")
	}
	accessTokenStr := string(cache.AccessToken)
	var quoted string
	if err := json.Unmarshal(cache.AccessToken, &quoted); err == nil {
		accessTokenStr = quoted
	}
	return ParseSTSCredentials(accessTokenStr)
}

func refreshLoginCache(ctx context.Context, provider Provider, cache *LoginTokenCache) error {
	if cache == nil {
		return errors.New("login cache is nil")
	}
	if strings.TrimSpace(cache.RefreshToken) == "" {
		return errors.New("no refresh token available")
	}
	if strings.TrimSpace(cache.ClientID) == "" {
		return errors.New("login cache missing client_id")
	}
	endpointURL := strings.TrimSpace(cache.EndpointURL)
	if endpointURL == "" {
		endpointURL = DefaultConsoleEndpoint
	}
	scope := strings.TrimSpace(cache.Scope)
	if scope == "" {
		scope = ScopeAllAll
	}
	tokenResp, err := newConsoleOAuthClient(endpointURL).exchangeToken(ctx, &consoleTokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: cache.RefreshToken,
		ClientID:     cache.ClientID,
		Scope:        scope,
	})
	if err != nil {
		return err
	}
	if _, err := ParseSTSCredentials(tokenResp.AccessToken); err != nil {
		return fmt.Errorf("parsing refreshed STS credentials: %w", err)
	}
	cache.AccessToken = json.RawMessage(tokenResp.AccessToken)
	if tokenResp.RefreshToken != "" {
		cache.RefreshToken = tokenResp.RefreshToken
	}
	if tokenResp.IDToken != "" {
		cache.IDToken = tokenResp.IDToken
	}
	cache.Scope = scope
	cache.EndpointURL = endpointURL
	cache.IssuedAt = time.Now().UTC().Format(time.RFC3339)
	cache.ExpiresIn = tokenResp.ExpiresIn
	cache.TokenType = tokenResp.TokenType
	return nil
}

// ParseSTSCredentials decodes the JSON-encoded STS credential bundle carried in
// the OAuth access_token.
func ParseSTSCredentials(accessToken string) (*STSCredentials, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("access_token is empty")
	}
	var creds STSCredentials
	if err := json.Unmarshal([]byte(accessToken), &creds); err != nil {
		return nil, fmt.Errorf("failed to parse STS credentials from access_token: %w", err)
	}
	if creds.AccessKeyID == "" {
		return nil, errors.New("parsed STS credentials missing access_key_id")
	}
	if creds.SecretAccessKey == "" {
		return nil, errors.New("parsed STS credentials missing secret_access_key")
	}
	if creds.SessionToken == "" {
		return nil, errors.New("parsed STS credentials missing session_token")
	}
	return &creds, nil
}

type consoleOAuthClient struct {
	endpointURL  string
	authorizeURL string
	tokenURL     string
	httpClient   *http.Client
}

func newConsoleOAuthClient(endpointURL string) *consoleOAuthClient {
	endpoint := strings.TrimRight(strings.TrimSpace(endpointURL), "/")
	if endpoint == "" {
		endpoint = DefaultConsoleEndpoint
	}
	return &consoleOAuthClient{
		endpointURL:  endpoint,
		authorizeURL: endpoint + consoleAuthorizePath,
		tokenURL:     endpoint + consoleTokenPath,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *consoleOAuthClient) buildAuthorizeURL(clientID, scope, codeChallenge, state, redirectURI string) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("scope", scope)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	q.Set("redirect_uri", redirectURI)
	return c.authorizeURL + "?" + q.Encode()
}

func (c *consoleOAuthClient) exchangeToken(ctx context.Context, req *consoleTokenRequest) (*ConsoleTokenResponse, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	if strings.TrimSpace(req.GrantType) == "" {
		return nil, errors.New("grant_type is required")
	}
	if strings.TrimSpace(req.ClientID) == "" {
		return nil, errors.New("client_id is required")
	}
	q := url.Values{}
	q.Set("grant_type", req.GrantType)
	q.Set("client_id", req.ClientID)
	if req.Scope != "" {
		q.Set("scope", req.Scope)
	}
	switch req.GrantType {
	case "authorization_code":
		if strings.TrimSpace(req.Code) == "" {
			return nil, errors.New("code is required for authorization_code grant")
		}
		if strings.TrimSpace(req.CodeVerifier) == "" {
			return nil, errors.New("code_verifier is required for authorization_code grant")
		}
		q.Set("code", req.Code)
		q.Set("code_verifier", req.CodeVerifier)
		if req.RedirectURI != "" {
			q.Set("redirect_uri", req.RedirectURI)
		}
	case "refresh_token":
		if strings.TrimSpace(req.RefreshToken) == "" {
			return nil, errors.New("refresh_token is required for refresh_token grant")
		}
		q.Set("refresh_token", req.RefreshToken)
	default:
		return nil, fmt.Errorf("unsupported grant_type: %s", req.GrantType)
	}
	var tokenResp ConsoleTokenResponse
	err := doConsoleRetry(ctx, 3, func() error {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(q.Encode()))
		if err != nil {
			return err
		}
		httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if customHeaders := os.Getenv("VOLCENGINE_LOGIN_HEADERS"); customHeaders != "" {
			for _, entry := range strings.Split(customHeaders, ";") {
				if idx := strings.Index(entry, "="); idx > 0 {
					httpReq.Header.Set(strings.TrimSpace(entry[:idx]), strings.TrimSpace(entry[idx+1:]))
				}
			}
		}
		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if resp.StatusCode/100 != 2 {
			apiErr := &ConsoleOAuthAPIError{
				StatusCode: resp.StatusCode,
				RequestID:  resp.Header.Get("X-Tt-Logid"),
				RawBody:    string(body),
			}
			var errResp ConsoleOAuthErrorResponse
			if json.Unmarshal(body, &errResp) == nil {
				apiErr.Response = errResp
			}
			return apiErr
		}
		if len(body) > 0 {
			if err := json.Unmarshal(body, &tokenResp); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if tokenResp.AccessToken == "" && tokenResp.TokenType == "" && tokenResp.RefreshToken == "" && tokenResp.ExpiresIn == 0 {
		return nil, errors.New("ExchangeToken succeeded but response was empty")
	}
	return &tokenResp, nil
}

func doConsoleRetry(ctx context.Context, maxAttempts int, fn func() error) error {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if ctx != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		var oauthErr *ConsoleOAuthAPIError
		retryable := errors.As(lastErr, &oauthErr) && (oauthErr.StatusCode == http.StatusTooManyRequests || oauthErr.StatusCode == http.StatusRequestTimeout || oauthErr.StatusCode/100 == 5)
		if !retryable || attempt == maxAttempts {
			return lastErr
		}
		time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
	}
	return lastErr
}

func localAuthorize(ctx context.Context, output io.Writer, oauthClient *consoleOAuthClient, clientID, codeChallenge, state string, openLoginPage bool) (string, string, error) {
	cbServer, err := newCallbackServer()
	if err != nil {
		return "", "", fmt.Errorf("starting callback server: %w", err)
	}
	cbServer.start()
	defer cbServer.shutdown()
	redirectURI := cbServer.redirectURI()
	authorizeURL := oauthClient.buildAuthorizeURL(clientID, ScopeAllAll, codeChallenge, state, redirectURI)
	if openLoginPage {
		fmt.Fprintln(output, "Attempting to automatically open the login page in your default browser.")
		fmt.Fprintln(output, "If the browser does not open, open the following URL:")
	} else {
		fmt.Fprintln(output, "Open the following URL in your browser to login:")
	}
	fmt.Fprintln(output, authorizeURL)
	if openLoginPage {
		_ = openBrowser(authorizeURL)
	}
	result, err := cbServer.waitForCallback(ctx, 10*time.Minute)
	if err != nil {
		return "", "", fmt.Errorf("waiting for authorization callback: %w", err)
	}
	if result.Error != "" {
		desc := result.Error
		if result.ErrorDescription != "" {
			desc = fmt.Sprintf("%s: %s", result.Error, result.ErrorDescription)
		}
		return "", "", fmt.Errorf("authorization failed: %s", desc)
	}
	if result.State != state {
		return "", "", fmt.Errorf("state mismatch: expected %s, got %s", state, result.State)
	}
	if result.Code == "" {
		return "", "", errors.New("authorization callback did not include an authorization code")
	}
	return result.Code, redirectURI, nil
}

func remoteAuthorize(ctx context.Context, input io.Reader, output io.Writer, oauthClient *consoleOAuthClient, clientID, codeChallenge, state, endpointURL string) (string, string, error) {
	redirectURI := strings.TrimRight(endpointURL, "/") + consoleAuthorizePath
	authorizeURL := oauthClient.buildAuthorizeURL(clientID, ScopeAllAll, codeChallenge, state, redirectURI)
	fmt.Fprintln(output, "Open the following URL in a browser on any device:")
	fmt.Fprintln(output)
	fmt.Fprintln(output, authorizeURL)
	fmt.Fprintln(output)

	// Without a TTY there is nobody to paste the code, so fall back to polling a
	// temporary file for the authorization response and clean it up afterwards.
	var pollFile string
	if !isInteractiveTerminal() {
		tmp, err := os.CreateTemp("", "byted-postgresql-login-*.txt")
		if err != nil {
			return "", "", fmt.Errorf("creating temporary authorization code file: %w", err)
		}
		pollFile = tmp.Name()
		_ = tmp.Close()
		defer os.Remove(pollFile)
	}

	if pollFile != "" {
		authCode, err := pollAuthCodeFromFile(ctx, output, pollFile, state)
		if err != nil {
			return "", "", err
		}
		return authCode, redirectURI, nil
	}

	fmt.Fprintln(output, "After completing login, enter the authorization code shown in the browser:")
	fmt.Fprint(output, "Authorization code: ")
	rawInput, err := bufio.NewReader(input).ReadString('\n')
	if err != nil {
		return "", "", err
	}
	rawInput = strings.TrimSpace(rawInput)
	if rawInput == "" {
		return "", "", errors.New("authorization code cannot be empty")
	}
	authCode, err := decodeRemoteAuthResponse(rawInput, state)
	if err != nil {
		return "", "", err
	}
	return authCode, redirectURI, nil
}

// pollAuthCodeFromFile waits for an authorization response to be written to
// codeFile, decoding it with decodeRemoteAuthResponse once available.
func pollAuthCodeFromFile(ctx context.Context, output io.Writer, codeFile, state string) (string, error) {
	fmt.Fprintf(output, "Waiting for authorization code — write it to: %s\n", codeFile)

	ticker := time.NewTicker(remoteCodeFilePollInterval)
	defer ticker.Stop()
	timeout := time.After(remoteCodeFilePollTimeout)

	var prev string
	var lastDecodeErr error
	for {
		raw, err := readTrimmedFile(codeFile)
		if err != nil {
			return "", fmt.Errorf("reading authorization code file %s: %w", codeFile, err)
		}
		if raw != "" {
			authCode, decodeErr := decodeRemoteAuthResponse(raw, state)
			if decodeErr == nil {
				return authCode, nil
			}
			if raw == prev {
				return "", fmt.Errorf("authorization code in %s is invalid: %w", codeFile, decodeErr)
			}
			prev, lastDecodeErr = raw, decodeErr
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			if lastDecodeErr != nil {
				return "", fmt.Errorf("timed out waiting for authorization code in %s: last decode error: %w", codeFile, lastDecodeErr)
			}
			return "", fmt.Errorf("timed out waiting for authorization code in %s", codeFile)
		case <-ticker.C:
		}
	}
}

func readTrimmedFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func decodeRemoteAuthResponse(raw, state string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("authorization code cannot be empty")
	}
	var decoded []byte
	var decodeErr error
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	} {
		if decoded, decodeErr = enc.DecodeString(raw); decodeErr == nil {
			break
		}
	}
	if decodeErr != nil {
		return "", fmt.Errorf("base64 decoding authorization response: %w", decodeErr)
	}
	params, err := url.ParseQuery(string(decoded))
	if err != nil {
		return "", fmt.Errorf("parsing decoded authorization response: %w", err)
	}
	authCode := params.Get("code")
	if authCode == "" {
		return "", errors.New("decoded authorization response does not contain a code parameter")
	}
	if respondedState := params.Get("state"); respondedState != state {
		return "", fmt.Errorf("state mismatch: expected %s, got %s", state, respondedState)
	}
	return authCode, nil
}

func resolveConsoleLoginRegion(input io.Reader, output io.Writer, commandRegion string) (string, error) {
	commandRegion = strings.TrimSpace(commandRegion)
	if commandRegion != "" {
		if err := ValidateRegion(commandRegion); err != nil {
			return "", err
		}
		return commandRegion, nil
	}
	if !isInteractiveTerminal() {
		fmt.Fprintf(output, "Using default region: %s\n", DefaultConsoleLoginRegion)
		return DefaultConsoleLoginRegion, nil
	}
	fmt.Fprintf(output, "Please enter region [%s]: ", DefaultConsoleLoginRegion)
	line, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return DefaultConsoleLoginRegion, nil
	}
	if err := ValidateRegion(line); err != nil {
		return "", err
	}
	return line, nil
}

func confirmLoginSessionReplacement(input io.Reader, output io.Writer, profileName, currentLoginSession, newLoginSession string, assumeYes bool) (bool, error) {
	if currentLoginSession == "" || currentLoginSession == newLoginSession {
		return true, nil
	}
	fmt.Fprintf(output, "Profile %q is currently using login_session %q.\n", profileName, currentLoginSession)
	fmt.Fprintf(output, "The new login would replace it with %q.\n", newLoginSession)
	if assumeYes {
		fmt.Fprintln(output, "Replace the existing login_session? [y/N]: y")
		return true, nil
	}
	if !isInteractiveTerminal() {
		return false, errors.New("confirmation required in non-interactive mode; re-run with --yes to confirm")
	}
	fmt.Fprint(output, "Replace the existing login_session? [y/N]: ")
	response, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	answer := strings.ToLower(strings.TrimSpace(response))
	return answer == "y" || answer == "yes", nil
}

func isInteractiveTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func loginCacheDir(provider Provider) (string, error) {
	if customCacheDir := os.Getenv(loginCacheDirectoryEnv); customCacheDir != "" {
		if err := os.MkdirAll(customCacheDir, 0700); err != nil {
			return "", err
		}
		return customCacheDir, nil
	}
	path, err := ConfigFilePath(provider)
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(filepath.Dir(path), "login", "cache")
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return "", err
	}
	return cacheDir, nil
}

func loginCacheFilePath(provider Provider, loginSession string) (string, error) {
	cacheDir, err := loginCacheDir(provider)
	if err != nil {
		return "", err
	}
	h := sha1.New()
	h.Write([]byte(loginSession))
	return filepath.Join(cacheDir, fmt.Sprintf("%x.json", h.Sum(nil))), nil
}

func writeLoginCacheFor(provider Provider, cache *LoginTokenCache) error {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	cachePath, err := loginCacheFilePath(provider, cache.LoginSession)
	if err != nil {
		return err
	}
	dir := filepath.Dir(cachePath)
	tmpFile, err := os.CreateTemp(dir, ".tmp-login-cache-*")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, 0600); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, cachePath)
}

func readLoginCacheFor(provider Provider, loginSession string) (*LoginTokenCache, error) {
	cachePath, err := loginCacheFilePath(provider, loginSession)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}
	var cache LoginTokenCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	return &cache, nil
}

func removeLoginCacheFor(provider Provider, loginSession string) error {
	cachePath, err := loginCacheFilePath(provider, loginSession)
	if err != nil {
		return err
	}
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func writeLoginCache(cache *LoginTokenCache) error {
	return writeLoginCacheFor(ProviderVolcengine, cache)
}

func readLoginCache(loginSession string) (*LoginTokenCache, error) {
	return readLoginCacheFor(ProviderVolcengine, loginSession)
}

func removeLoginCache(loginSession string) error {
	return removeLoginCacheFor(ProviderVolcengine, loginSession)
}

func extractLoginSession(idToken string) (string, error) {
	if idToken == "" {
		return "", errors.New("id_token is empty")
	}
	parts := strings.Split(idToken, ".")
	if len(parts) < 2 {
		return "", errors.New("id_token does not have a valid JWT structure")
	}
	payload := parts[1]
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}
	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return "", err
	}
	var claims struct {
		TRN string `json:"trn"`
	}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return "", err
	}
	if claims.TRN == "" {
		return "", errors.New("id_token JWT payload does not contain a trn claim")
	}
	return claims.TRN, nil
}

func generateCodeVerifier() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func generateState() (string, error) {
	var uuid [16]byte
	if _, err := rand.Read(uuid[:]); err != nil {
		return "", err
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16]), nil
}

func openBrowser(rawURL string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", rawURL).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL).Start()
	case "darwin":
		return exec.Command("open", rawURL).Start()
	default:
		return errors.New("cannot open browser automatically")
	}
}

type authorizationResult struct {
	Code             string
	State            string
	Error            string
	ErrorDescription string
}

type callbackServer struct {
	server   *http.Server
	listener net.Listener
	result   chan *authorizationResult
	port     int
}

func newCallbackServer() (*callbackServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	cs := &callbackServer{
		listener: listener,
		result:   make(chan *authorizationResult, 1),
		port:     listener.Addr().(*net.TCPAddr).Port,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/callback", cs.handleCallback)
	cs.server = &http.Server{Handler: mux}
	return cs, nil
}

func (s *callbackServer) redirectURI() string {
	return fmt.Sprintf("http://127.0.0.1:%d/oauth/callback", s.port)
}

func (s *callbackServer) start() {
	go func() {
		if err := s.server.Serve(s.listener); err != nil && err != http.ErrServerClosed {
			s.result <- &authorizationResult{Error: "server_error", ErrorDescription: err.Error()}
		}
	}()
}

func (s *callbackServer) waitForCallback(ctx context.Context, timeout time.Duration) (*authorizationResult, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case result := <-s.result:
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, fmt.Errorf("timed out waiting for OAuth callback after %v", timeout)
	}
}

func (s *callbackServer) shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.server.Shutdown(ctx)
}

func (s *callbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	query := r.URL.Query()
	oauthError := query.Get("error")
	if oauthError == "" {
		oauthError = query.Get("Error")
	}
	errorDescription := query.Get("error_description")
	if oauthError == "" {
		oauthError = errorDescription
		errorDescription = ""
	}
	result := &authorizationResult{
		Code:             query.Get("code"),
		State:            query.Get("state"),
		Error:            oauthError,
		ErrorDescription: errorDescription,
	}
	select {
	case s.result <- result:
	default:
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	message := "Authentication successful! You can close this page and return to the terminal."
	if oauthError != "" {
		message = "Authentication failed: " + oauthError
		if errorDescription != "" {
			message += ": " + errorDescription
		}
	}
	fmt.Fprintf(w, "<html><body><h2>%s</h2></body></html>", html.EscapeString(message))
}

func printPostLogoutHint(output io.Writer) {
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Note: Local cache has been removed for future CLI sessions.")
	fmt.Fprintln(output, "Already-running tools that loaded temporary STS credentials before logout")
	fmt.Fprintln(output, "may continue to use them until those credentials expire.")
}
