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
	"encoding/json"
	"fmt"

	"github.com/volcengine/byted-postgresql-cli/internal/pagination"
)

const (
	// EngineTypePostgreSQL is the Volcengine AIDAP engine type this CLI manages. Every
	// workspace is created with it, so the PG CLI never creates Supabase or veDB
	// MySQL workspaces sharing the platform.
	EngineTypePostgreSQL = "PostgreSQL"
	// engineVersionPostgreSQL17 is the engine version used when creating a new
	// PostgreSQL workspace, and the value used to filter listings by engine.
	engineVersionPostgreSQL17 = "PostgreSQL_17"
	// filterNameDBEngineVersion is the documented DescribeWorkspaces filter Name
	// that selects an engine server-side (so filtering happens before pagination
	// and Total counts only the matching engine). Volcengine AIDAP recognizes
	// "DBEngineVersion" with values like "PostgreSQL_17"/"Supabase_1_24"; the
	// undocumented "EngineType" filter is silently ignored by the gateway.
	filterNameDBEngineVersion = "DBEngineVersion"

	// MaxPageLimit is the largest page size accepted by Volcengine AIDAP list APIs.
	MaxPageLimit      = pagination.MaxLimit
	maxWorkspaceLimit = MaxPageLimit
	// DefaultListLimit is the default page size list endpoints use when the caller
	// does not specify a count. The backend defaults to 10 when Limit is omitted;
	// this constant makes that explicit.
	DefaultListLimit = pagination.DefaultLimit
	// InteractiveListLimit is the page size used by interactive resource pickers.
	InteractiveListLimit = 20

	deletionProtectionEnabled  = "Enabled"
	deletionProtectionDisabled = "Disabled"

	sortOrderDesc        = "Desc"
	initSourceParentData = "ParentData"

	// ServiceTypeDatabase is the only compute service type a PostgreSQL workspace
	// runs. (Supabase workspaces additionally run a "Supabase" service type; the
	// PG CLI never touches those.)
	ServiceTypeDatabase = "Database"
	// ComputeRolePrimary marks the read-write compute of a branch.
	ComputeRolePrimary = "Primary"

	// ACL types and modify modes for the network allow-list API.
	ACLTypeAllow        = "Allow"
	ACLModifyModeCover  = "Cover"
	ACLModifyModeAppend = "Append"
	ACLModifyModeDelete = "Delete"
)

// remarshal decodes a Volcengine AIDAP gateway response (returned by the SDK as
// map[string]interface{}, since the generator leaves result schemas untyped)
// into a locally-shaped struct, tolerating extra/missing fields — encoding/json
// only fails on structurally invalid JSON. Volcengine AIDAP wraps every payload in
// {"ResponseMetadata":{...},"Result":{...}}; when the input has that shape, this
// decodes the inner Result so callers can define flat structs.
func remarshal(v interface{}, out interface{}) error {
	if m, ok := v.(map[string]interface{}); ok {
		if inner, ok := m["Result"]; ok {
			v = inner
		}
	}
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to encode Volcengine response: %w", err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("failed to decode Volcengine response: %w", err)
	}
	return nil
}
