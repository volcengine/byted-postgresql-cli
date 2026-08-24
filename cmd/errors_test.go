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

package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/volcengine/byted-postgresql-cli/internal/api"
)

type sdkStatusError struct {
	status int
	body   string
}

func (e sdkStatusError) Error() string   { return e.body }
func (e sdkStatusError) StatusCode() int { return e.status }
func (e sdkStatusError) Body() []byte    { return []byte(e.body) }

func TestFormatErrorForForbiddenResource(t *testing.T) {
	err := &api.APIError{
		Status:  403,
		Message: "psm not found: aidap.project.internal",
		Path:    "/projects/does-not-exist",
	}
	got := FormatError(err)
	for _, want := range []string{
		"HTTP 403",
		"workspaces list",
		"status",
		"project administrator",
		"Request path: /projects/does-not-exist",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("FormatError() = %q, want substring %q", got, want)
		}
	}
	if strings.Contains(got, "psm not found") {
		t.Fatalf("FormatError() leaked backend detail: %q", got)
	}
}

func TestFormatErrorForNotFoundResource(t *testing.T) {
	got := FormatError(sdkStatusError{status: 404, body: `{"message":"internal detail"}`})
	if !strings.Contains(got, "HTTP 404") || !strings.Contains(got, "Retry with the confirmed resource ID") {
		t.Fatalf("FormatError() = %q, want actionable 404 message", got)
	}
	if strings.Contains(got, "internal detail") {
		t.Fatalf("FormatError() leaked backend detail: %q", got)
	}
}

func TestFormatErrorForDeletionProtection(t *testing.T) {
	got := FormatError(sdkStatusError{
		status: 400,
		body:   `{"code":"OperationDenied_DeletionProtection"}`,
	})
	for _, want := range []string{
		"deletion protection",
		"workspaces deletion-protection",
		"--disable",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("FormatError() = %q, want substring %q", got, want)
		}
	}
	if strings.Contains(got, "OperationDenied_DeletionProtection") {
		t.Fatalf("FormatError() leaked deletion protection API code: %q", got)
	}
}

func TestFormatErrorForUpdateRegistryFailure(t *testing.T) {
	got := FormatError(errors.New("fetch latest release: status 404 Not Found"))
	if !strings.Contains(got, "public npm registry") || !strings.Contains(got, "@byted-postgresql/cli") ||
		!strings.Contains(got, "npm view") {
		t.Fatalf("FormatError() = %q, want update registry guidance", got)
	}
	if strings.Contains(got, "workspaces list") {
		t.Fatalf("FormatError() gave workspace guidance for update failure: %q", got)
	}
}

func TestFormatErrorPreservesStatusErrorDetail(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "message",
			body: `{"code":"InvalidParameter","message":"database already exists"}`,
			want: "database already exists",
		},
		{
			name: "nested detail",
			body: `{"Result":{"Error":{"detail":"role does not exist"}}}`,
			want: "role does not exist",
		},
		{
			name: "raw body",
			body: "invalid request payload",
			want: "invalid request payload",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := FormatError(sdkStatusError{status: 400, body: test.body})
			if !strings.Contains(got, "HTTP 400") || !strings.Contains(got, test.want) {
				t.Fatalf("FormatError() = %q, want HTTP 400 and %q", got, test.want)
			}
		})
	}
}

func TestFormatErrorPreservesOrdinaryErrors(t *testing.T) {
	err := errors.New("validation failed")
	if got := FormatError(err); got != err.Error() {
		t.Fatalf("FormatError() = %q, want %q", got, err.Error())
	}
}

func TestFormatErrorRecognizesGeneratedSDKStatusText(t *testing.T) {
	err := errors.New("failed to describe workspaces: 403 Forbidden")
	got := FormatError(err)
	for _, want := range []string{
		"HTTP 403",
		"workspaces list",
		"status",
		"project administrator",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("FormatError() = %q, want substring %q", got, want)
		}
	}
}

type bodyError struct {
	body []byte
}

func (e bodyError) Error() string { return "unexpected status 400 Bad Request" }
func (e bodyError) Body() []byte  { return e.body }

func TestFormatErrorPreservesSDKStyleErrorBody(t *testing.T) {
	got := FormatError(bodyError{
		body: []byte(`{"message":"time must be RFC3339"}`),
	})
	if !strings.Contains(got, "HTTP 400") || !strings.Contains(got, "time must be RFC3339") {
		t.Fatalf("FormatError() = %q, want generated SDK detail", got)
	}
}
