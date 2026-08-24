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

package api

import (
	"errors"
	"fmt"
	"net/http"
)

// APIError describes an HTTP API failure while preserving the request path and
// raw response body for classification and diagnostics.
type APIError struct {
	Status  int
	Message string
	Path    string
	Body    []byte
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("request failed with status %d", e.Status)
}

// StatusCode allows shared CLI error formatting to classify APIError in the
// same way as Volcengine SDK request failures.
func (e *APIError) StatusCode() int {
	return e.Status
}

// RequestPath returns the failed request path for actionable CLI diagnostics.
func (e *APIError) RequestPath() string {
	return e.Path
}

// IsAuthError reports whether err represents an unauthenticated request.
func IsAuthError(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusUnauthorized
}

// IsNotFound reports whether err represents a missing resource.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound
}
