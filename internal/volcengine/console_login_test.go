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

package volcengine

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// makeIDToken builds an unsigned JWT whose payload carries the given trn claim,
// matching what extractLoginSession parses out.
func makeIDToken(t *testing.T, trn string) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	payloadJSON, err := json.Marshal(map[string]string{"trn": trn})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	return header + "." + payload + ".sig"
}

// stsAccessToken encodes STS credentials the way the signin service embeds them
// in the OAuth access_token.
func stsAccessToken(t *testing.T, ak, sk, token string) string {
	t.Helper()
	data, err := json.Marshal(STSCredentials{AccessKeyID: ak, SecretAccessKey: sk, SessionToken: token})
	if err != nil {
		t.Fatalf("marshal sts: %v", err)
	}
	return string(data)
}

// isolateConfig points both the config store and login cache at a temp dir so
// the tests never touch a developer's real ~/.volcengine.
func isolateConfig(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(loginCacheDirectoryEnv, filepath.Join(home, "login-cache"))
	// Clear any ambient credential/profile env so resolution uses our profile.
	for _, k := range []string{EnvAccessKeyID, EnvSecretAccessKey, EnvSessionToken, EnvProfile, EnvRegion, EnvLongRegion} {
		t.Setenv(k, "")
	}
	SetProfileOverride("")
	SetRegionOverride("")
}

// fakeSigninServer emulates the Volcengine signin OAuth token endpoint.
func fakeSigninServer(t *testing.T, idToken, accessToken string, refreshValid bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, consoleTokenPath) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		switch r.Form.Get("grant_type") {
		case "authorization_code":
			writeJSON(w, ConsoleTokenResponse{
				AccessToken:  accessToken,
				TokenType:    "Bearer",
				ExpiresIn:    900,
				RefreshToken: "refresh-1",
				Scope:        ScopeAllAll,
				IDToken:      idToken,
			})
		case "refresh_token":
			if !refreshValid {
				w.WriteHeader(http.StatusBadRequest)
				writeJSON(w, ConsoleOAuthErrorResponse{
					Error:            "invalid_request",
					ErrorDescription: "The request parameter refresh_token is invalid.",
				})
				return
			}
			writeJSON(w, ConsoleTokenResponse{
				AccessToken:  accessToken,
				TokenType:    "Bearer",
				ExpiresIn:    900,
				RefreshToken: "refresh-2",
				Scope:        ScopeAllAll,
				IDToken:      idToken,
			})
		default:
			http.Error(w, "unsupported grant", http.StatusBadRequest)
		}
	}))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// TestConsoleLoginRoundTrip drives a full authorization_code exchange against a
// fake signin server and verifies the profile + cache are written and that
// ResolveConfig then returns the STS credentials.
func TestConsoleLoginRoundTrip(t *testing.T) {
	isolateConfig(t)
	trn := "trn:iam::123:user/tester"
	idToken := makeIDToken(t, trn)
	accessToken := stsAccessToken(t, "AKSTS", "SKSTS", "session-xyz")
	srv := fakeSigninServer(t, idToken, accessToken, true)
	defer srv.Close()

	oauthClient := newConsoleOAuthClient(srv.URL)
	verifier, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("verifier: %v", err)
	}
	cache, err := completeConsoleLogin(context.Background(), completeConsoleLoginParams{
		OAuthClient:  oauthClient,
		AuthCode:     "auth-code",
		RedirectURI:  "http://127.0.0.1:1234/oauth/callback",
		ClientID:     consoleClientIDSameDevice,
		CodeVerifier: verifier,
		Profile:      "default",
		Region:       "cn-beijing",
		EndpointURL:  srv.URL,
		Output:       nil,
		Confirm:      false,
	})
	if err != nil {
		t.Fatalf("completeConsoleLogin: %v", err)
	}
	if cache.LoginSession != trn {
		t.Fatalf("login_session = %q, want %q", cache.LoginSession, trn)
	}

	// The profile must be persisted as console-login.
	cfg, err := LoadFileConfig()
	if err != nil {
		t.Fatalf("LoadFileConfig: %v", err)
	}
	profile := cfg.Profiles["default"]
	if profile == nil || profile.Mode != ModeConsoleLogin || profile.LoginSession != trn {
		t.Fatalf("persisted profile = %+v, want console-login with login_session %q", profile, trn)
	}

	// ResolveConfig must surface the cached STS credentials.
	resolved, err := ResolveConfig(context.Background())
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if resolved.AccessKeyID != "AKSTS" || resolved.SecretAccessKey != "SKSTS" || resolved.SessionToken != "session-xyz" {
		t.Fatalf("ResolveConfig returned %+v, want STS creds", resolved)
	}
	if resolved.Region != "cn-beijing" {
		t.Fatalf("resolved region = %q, want cn-beijing", resolved.Region)
	}
}

// TestResolveConfigRefreshesExpiredConsoleLogin proves that a cached token
// within the 60s expiry window is transparently refreshed before use.
func TestResolveConfigRefreshesExpiredConsoleLogin(t *testing.T) {
	isolateConfig(t)
	trn := "trn:iam::123:user/tester"
	freshAccess := stsAccessToken(t, "AKNEW", "SKNEW", "session-new")
	srv := fakeSigninServer(t, makeIDToken(t, trn), freshAccess, true)
	defer srv.Close()

	// Seed an expired cache (issued long ago) with an old access token.
	oldAccess := stsAccessToken(t, "AKOLD", "SKOLD", "session-old")
	cache := &LoginTokenCache{
		LoginSession: trn,
		AccessToken:  json.RawMessage(oldAccess),
		RefreshToken: "refresh-1",
		ClientID:     consoleClientIDSameDevice,
		EndpointURL:  srv.URL,
		Scope:        ScopeAllAll,
		IssuedAt:     time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
		ExpiresIn:    900,
		TokenType:    "Bearer",
	}
	if err := writeLoginCache(cache); err != nil {
		t.Fatalf("writeLoginCache: %v", err)
	}
	seedConsoleLoginProfile(t, "default", trn, "cn-beijing")

	resolved, err := ResolveConfig(context.Background())
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if resolved.AccessKeyID != "AKNEW" || resolved.SessionToken != "session-new" {
		t.Fatalf("ResolveConfig returned stale creds %+v, want refreshed", resolved)
	}
	// The refreshed token must have been written back to the cache.
	updated, err := readLoginCache(trn)
	if err != nil {
		t.Fatalf("readLoginCache: %v", err)
	}
	if updated.RefreshToken != "refresh-2" {
		t.Fatalf("cache refresh_token = %q, want rotated refresh-2", updated.RefreshToken)
	}
}

// TestResolveConfigConsoleLoginExpiredRefreshFails proves the OAuth service's
// own error is passed through when the refresh token is rejected.
func TestResolveConfigConsoleLoginExpiredRefreshFails(t *testing.T) {
	isolateConfig(t)
	trn := "trn:iam::123:user/tester"
	srv := fakeSigninServer(t, makeIDToken(t, trn), stsAccessToken(t, "A", "B", "C"), false)
	defer srv.Close()

	cache := &LoginTokenCache{
		LoginSession: trn,
		AccessToken:  json.RawMessage(stsAccessToken(t, "AKOLD", "SKOLD", "session-old")),
		RefreshToken: "refresh-1",
		ClientID:     consoleClientIDSameDevice,
		EndpointURL:  srv.URL,
		Scope:        ScopeAllAll,
		IssuedAt:     time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
		ExpiresIn:    900,
		TokenType:    "Bearer",
	}
	if err := writeLoginCache(cache); err != nil {
		t.Fatalf("writeLoginCache: %v", err)
	}
	seedConsoleLoginProfile(t, "default", trn, "cn-beijing")

	_, err := ResolveConfig(context.Background())
	if err == nil {
		t.Fatal("ResolveConfig succeeded, want refresh failure")
	}
	if !strings.Contains(err.Error(), "refresh_token is invalid") {
		t.Fatalf("error = %v, want passed-through OAuth message", err)
	}
}

// TestRunConsoleLogoutRemovesProfileAndCache verifies logout deletes both the
// console-login profile and its cached token.
func TestRunConsoleLogoutRemovesProfileAndCache(t *testing.T) {
	isolateConfig(t)
	trn := "trn:iam::123:user/tester"
	cache := &LoginTokenCache{
		LoginSession: trn,
		AccessToken:  json.RawMessage(stsAccessToken(t, "AK", "SK", "ST")),
		ClientID:     consoleClientIDSameDevice,
		Scope:        ScopeAllAll,
		IssuedAt:     time.Now().UTC().Format(time.RFC3339),
		ExpiresIn:    900,
	}
	if err := writeLoginCache(cache); err != nil {
		t.Fatalf("writeLoginCache: %v", err)
	}
	seedConsoleLoginProfile(t, "default", trn, "cn-beijing")

	var out strings.Builder
	if err := RunConsoleLogout(ConsoleLogoutParams{Profile: "default"}, &out); err != nil {
		t.Fatalf("RunConsoleLogout: %v", err)
	}
	cfg, err := LoadFileConfig()
	if err != nil {
		t.Fatalf("LoadFileConfig: %v", err)
	}
	if _, ok := cfg.Profiles["default"]; ok {
		t.Fatalf("profile still present after logout: %+v", cfg.Profiles)
	}
	if _, err := readLoginCache(trn); err == nil {
		t.Fatal("login cache still present after logout")
	}
}

// TestRunConsoleLogoutRejectsAKProfile confirms logout refuses to delete AK/SK
// profiles.
func TestRunConsoleLogoutRejectsAKProfile(t *testing.T) {
	isolateConfig(t)
	cfg := FileConfig{Current: "default", Profiles: map[string]*Profile{
		"default": {Name: "default", Mode: ModeAK, AccessKey: "AK", SecretKey: "SK", Region: "cn-beijing"},
	}}
	if err := SaveFileConfig(cfg); err != nil {
		t.Fatalf("SaveFileConfig: %v", err)
	}
	err := RunConsoleLogout(ConsoleLogoutParams{Profile: "default"}, &strings.Builder{})
	if err == nil || !strings.Contains(err.Error(), "not") {
		t.Fatalf("RunConsoleLogout on AK profile err = %v, want refusal", err)
	}
}

func TestRunConsoleLogoutMissingProfileIsIdempotent(t *testing.T) {
	isolateConfig(t)
	var output strings.Builder
	if err := RunConsoleLogout(ConsoleLogoutParams{Profile: "default"}, &output); err != nil {
		t.Fatalf("RunConsoleLogout missing profile: %v", err)
	}
	if !strings.Contains(output.String(), "Nothing to do") {
		t.Fatalf("output = %q, want no-op message", output.String())
	}
}

func TestExtractLoginSession(t *testing.T) {
	trn := "trn:iam::999:user/example"
	got, err := extractLoginSession(makeIDToken(t, trn))
	if err != nil {
		t.Fatalf("extractLoginSession: %v", err)
	}
	if got != trn {
		t.Fatalf("extractLoginSession = %q, want %q", got, trn)
	}
}

func TestParseSTSCredentialsValidatesFields(t *testing.T) {
	if _, err := ParseSTSCredentials(""); err == nil {
		t.Fatal("empty access_token should error")
	}
	if _, err := ParseSTSCredentials(`{"access_key_id":"a"}`); err == nil {
		t.Fatal("missing secret should error")
	}
	creds, err := ParseSTSCredentials(`{"access_key_id":"a","secret_access_key":"b","session_token":"c"}`)
	if err != nil {
		t.Fatalf("valid creds errored: %v", err)
	}
	if creds.AccessKeyID != "a" || creds.SecretAccessKey != "b" || creds.SessionToken != "c" {
		t.Fatalf("parsed creds = %+v", creds)
	}
}

func TestDecodeRemoteAuthResponse(t *testing.T) {
	state := "state-123"
	raw := base64.StdEncoding.EncodeToString([]byte(url.Values{
		"code":  {"the-code"},
		"state": {state},
	}.Encode()))
	code, err := decodeRemoteAuthResponse(raw, state)
	if err != nil {
		t.Fatalf("decodeRemoteAuthResponse: %v", err)
	}
	if code != "the-code" {
		t.Fatalf("code = %q, want the-code", code)
	}
	// State mismatch must fail.
	if _, err := decodeRemoteAuthResponse(raw, "other-state"); err == nil {
		t.Fatal("state mismatch should error")
	}
}

func seedConsoleLoginProfile(t *testing.T, name, loginSession, region string) {
	t.Helper()
	cfg, err := LoadFileConfig()
	if err != nil {
		t.Fatalf("LoadFileConfig: %v", err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]*Profile{}
	}
	cfg.Profiles[name] = &Profile{
		Name:         name,
		Mode:         ModeConsoleLogin,
		Region:       region,
		LoginSession: loginSession,
	}
	cfg.Current = name
	if err := SaveFileConfig(cfg); err != nil {
		t.Fatalf("SaveFileConfig: %v", err)
	}
}
