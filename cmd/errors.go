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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

func FormatError(err error) string {
	if err == nil {
		return ""
	}

	if strings.Contains(err.Error(), "fetch latest release") {
		return fmt.Sprintf("Unable to check for CLI updates: %s was not found in the public npm registry. Confirm that the package has been published, or run `npm view @byted-postgresql/cli --registry https://registry.npmjs.org` to inspect its status. You can also run `npm install -g @byted-postgresql/cli --registry https://registry.npmjs.org` manually. Details: %s", "@byted-postgresql/cli", err.Error())
	}

	if strings.Contains(err.Error(), "OperationDenied_DeletionProtection") {
		return "Unable to delete the workspace because deletion protection is enabled. Run `byted-postgresql-cli workspaces deletion-protection <workspace-id> --disable` and retry."
	}

	// The Volcengine SDK surfaces gateway failures as errors implementing
	// StatusCode() int (volcengineerr.RequestFailure). Use it to render an
	// actionable message for the common 403/404 cases.
	var statusErr interface{ StatusCode() int }
	if errors.As(err, &statusErr) && statusErr.StatusCode() != 0 {
		path := ""
		var pathErr interface{ RequestPath() string }
		if errors.As(err, &pathErr) {
			path = pathErr.RequestPath()
		}
		return formatHTTPError(statusErr.StatusCode(), path, extractBackendDetail(err.Error()))
	}

	if status := statusFromErrorText(err.Error()); status != 0 {
		var sdkErr interface{ Body() []byte }
		if errors.As(err, &sdkErr) {
			return formatHTTPError(status, "", extractBackendDetail(string(sdkErr.Body())))
		}
		return formatHTTPError(status, "", "")
	}

	return err.Error()
}

var httpStatusPattern = regexp.MustCompile(`\b([45][0-9]{2})\s+[A-Za-z]`)

func statusFromErrorText(message string) int {
	match := httpStatusPattern.FindStringSubmatch(message)
	if len(match) != 2 {
		return 0
	}
	status, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return status
}

func extractBackendDetail(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	var payload any
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return body
	}
	if detail := findBackendDetail(payload); detail != "" {
		return detail
	}
	return body
}

func findBackendDetail(value any) string {
	switch value := value.(type) {
	case map[string]any:
		for _, key := range []string{"message", "Message", "error", "detail", "description"} {
			if detail, ok := value[key].(string); ok && strings.TrimSpace(detail) != "" {
				return strings.TrimSpace(detail)
			}
		}
		for _, key := range []string{"result", "Result", "response", "Response", "error", "Error", "ResponseMetadata"} {
			if nested, ok := value[key]; ok {
				if detail := findBackendDetail(nested); detail != "" {
					return detail
				}
			}
		}
	}
	return ""
}

func formatHTTPError(status int, path, detail string) string {
	switch status {
	case http.StatusForbidden, http.StatusNotFound:
		resource := "the requested resource"
		if status == http.StatusForbidden {
			resource = "the requested project or resource"
		}
		lines := []string{
			fmt.Sprintf("Unable to access %s (HTTP %d).", resource, status),
			"",
			"Possible causes:",
			"- the resource does not exist;",
			"- the current account lacks permission;",
			"- the selected region is incorrect.",
			"",
			"Next steps:",
			"1. Run `byted-postgresql-cli workspaces list` to confirm the workspace ID.",
			"2. Run `byted-postgresql-cli status` to confirm the credentials and region.",
		}
		if status == http.StatusForbidden {
			lines = append(lines, "3. Ask the project administrator to grant access.")
		} else {
			lines = append(lines, "3. Retry with the confirmed resource ID.")
		}
		if path != "" {
			lines = append(lines, "", fmt.Sprintf("Request path: %s", path))
		}
		return strings.Join(lines, "\n")
	default:
		if detail == "" {
			return fmt.Sprintf("Request failed (HTTP %d).", status)
		}
		return fmt.Sprintf("Request failed (HTTP %d): %s", status, detail)
	}
}
