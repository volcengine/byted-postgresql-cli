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

// Package volcengine wraps the Volcengine AIDAP OpenAPI SDK for the PostgreSQL
// engine. Volcengine's AIDAP platform hosts multiple engines (Supabase,
// PostgreSQL, veDB MySQL) behind one control-plane API; this package speaks the
// Action-based AIDAP gateway (projects / workspaces / branches), filtering every
// listing to EngineType=PostgreSQL and creating PostgreSQL_17 workspaces.
//
// Every exported method on Client corresponds to one AIDAP Action, invoked
// through the SDK's universal client so the request/response envelope
// ({"ResponseMetadata":..,"Result":..}) is signed with AK/SK and unwrapped into
// the locally-shaped, JSON-tagged structs this package defines. Errors from the
// gateway are passed through with fmt.Errorf("...: %w") so the caller prints the
// gateway's own message rather than a fabricated one.
package volcengine

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/volcengine/volcengine-go-sdk/service/aidap"
	sdk "github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
	"github.com/volcengine/volcengine-go-sdk/volcengine/endpoints"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

const managementAPIRequestTimeout = 60 * time.Second

// Client wraps the Volcengine AIDAP SDK. Every exported method corresponds to
// one AIDAP Action (see workspaces.go, branches.go). Actions are invoked through
// the strongly-typed aidap service client, which signs with AK/SK and decodes
// the gateway's Result envelope into the SDK's generated structs; mapXxx helpers
// then translate those into the locally-shaped, JSON-tagged structs this package
// exposes to the command layer.
type Client struct {
	cfg   Config
	aidap *aidap.AIDAP
}

// NewClient builds a client from a resolved AK/SK Config. It returns an error
// rather than panicking so callers on the command path can surface a friendly
// message.
func NewClient(cfg Config) (*Client, error) {
	creds := credentials.NewStaticCredentials(cfg.AccessKeyID, cfg.SecretAccessKey, cfg.SessionToken)
	region := cfg.Region
	if region == "" {
		region = DefaultRegion
	}
	sdkConfig := sdk.NewConfig().
		WithCredentials(creds).
		WithRegion(region).
		WithHTTPClient(&http.Client{Timeout: managementAPIRequestTimeout, Transport: newTransport()}).
		WithSimpleError(true)
	if cfg.Endpoint != "" {
		sdkConfig.WithEndpoint(cfg.Endpoint)
	} else {
		sdkConfig.WithEndpointResolver(endpoints.NewStandardEndpointResolver())
	}
	sess, err := session.NewSession(sdkConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build Volcengine AIDAP client: %w", err)
	}
	return &Client{cfg: cfg, aidap: aidap.New(sess)}, nil
}

// GatewayKnowsRegion reports whether the aidap service resolves for a region.
func GatewayKnowsRegion(region string) bool {
	return ValidateRegion(region) == nil
}

func newTransport() http.RoundTripper {
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialIPv4First,
		ForceAttemptHTTP2:     false,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: managementAPIRequestTimeout,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

func dialIPv4First(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("failed to split address %q: %w", address, err)
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		return nil, fmt.Errorf("invalid port %q: %w", port, err)
	}
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	addrs = sortIPv4First(addrs)
	dialer := net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	var lastErr error
	for _, addr := range addrs {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(addr.IP.String(), strconv.Itoa(portNumber)))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return dialer.DialContext(ctx, network, address)
}

func sortIPv4First(addrs []net.IPAddr) []net.IPAddr {
	sorted := append([]net.IPAddr(nil), addrs...)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].IP.To4() == nil && sorted[j].IP.To4() != nil {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	return sorted
}

// Pointer/value helpers bridge between the SDK's pointer-heavy request and
// response structs and the plain-value structs this package exposes.

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func intValue(value *int32) int {
	if value == nil {
		return 0
	}
	return int(*value)
}

func int64Value(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func floatValue(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func stringSlicePointers(values []string) []*string {
	result := make([]*string, len(values))
	for i := range values {
		result[i] = &values[i]
	}
	return result
}

func stringPointersValue(values []*string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != nil {
			result = append(result, *value)
		}
	}
	return result
}
