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

package writer

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		name   string
		format string
		wantOK bool
	}{
		{name: "table", format: "table", wantOK: true},
		{name: "json with whitespace", format: " JSON ", wantOK: true},
		{name: "yaml", format: "yaml", wantOK: true},
		{name: "csv", format: "csv", wantOK: true},
		{name: "tsv", format: "tsv", wantOK: true},
		{name: "unsupported", format: "xml", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFormat(tt.format)
			if (err == nil) != tt.wantOK {
				t.Fatalf("ValidateFormat(%q) error = %v, wantOK = %v", tt.format, err, tt.wantOK)
			}
			if !tt.wantOK && err.Error() != `unsupported output format "xml"; choose table, json, yaml, csv, or tsv` {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestWriteListNormalizesNilSliceToEmptyJSONArray(t *testing.T) {
	var items []struct {
		ID string `json:"id"`
	}
	var output bytes.Buffer
	w := New("json")
	w.Out = &output
	if err := w.WriteList(items, []string{"ID"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(output.String()); got != "[]" {
		t.Fatalf("WriteList() = %q, want []", got)
	}
}

func TestWriteListNormalizesNilSliceToEmptyYAMLArray(t *testing.T) {
	var items []struct {
		ID string `yaml:"id"`
	}
	var output bytes.Buffer
	w := New("yaml")
	w.Out = &output
	if err := w.WriteList(items, []string{"ID"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(output.String()); got != "[]" {
		t.Fatalf("WriteList() = %q, want []", got)
	}
}

func TestWriteListYAMLHandlesNilPointersAndUnexportedFields(t *testing.T) {
	type item struct {
		ID     string `json:"id"`
		Detail *struct {
			Name string `json:"name"`
		} `json:"detail"`
		hidden string
	}
	var output strings.Builder
	writer := &Writer{Format: FormatYAML, Out: &output}
	if err := writer.WriteList([]item{{ID: "one"}}, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "id: one") || strings.Contains(output.String(), "hidden") {
		t.Fatalf("YAML output = %q", output.String())
	}
}

func TestWriteListTableHandlesNilPointerItem(t *testing.T) {
	var output strings.Builder
	writer := &Writer{Format: FormatTable, Out: &output}
	var item *struct {
		ID string `json:"id"`
	}
	if err := writer.WriteList([]*struct {
		ID string `json:"id"`
	}{item}, []string{"ID"}); err != nil {
		t.Fatal(err)
	}
}

func TestWriteYAMLPreservesIntegerValues(t *testing.T) {
	item := struct {
		DataSizeUsedBytes int64 `json:"DataSizeUsedBytes"`
		Large             int64 `json:"Large"`
	}{
		DataSizeUsedBytes: 271552286,
		Large:             9007199254740993,
	}
	var output bytes.Buffer
	w := New("yaml")
	w.Out = &output
	if err := w.WriteItem(item, nil); err != nil {
		t.Fatal(err)
	}
	want := "data_size_used_bytes: 271552286\nlarge: 9007199254740993\n"
	if output.String() != want {
		t.Fatalf("WriteItem() YAML = %q, want %q", output.String(), want)
	}
}

func TestWriteListShowsMessageForEmptyTable(t *testing.T) {
	var items []string
	var output bytes.Buffer
	w := New("table")
	w.Out = &output
	if err := w.WriteList(items, []string{"ID"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "No results found.") {
		t.Fatalf("WriteList() = %q, want empty-result message", output.String())
	}
}

func TestWriteListTableKeepsFieldHeaders(t *testing.T) {
	items := []struct {
		ProjectName    string `json:"ProjectName"`
		WorkspaceCount int    `json:"WorkspaceCount"`
	}{
		{ProjectName: "gongna", WorkspaceCount: 1},
	}
	var output bytes.Buffer
	w := New("table")
	w.Out = &output
	if err := w.WriteList(items, []string{"ProjectName", "WorkspaceCount"}); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	if !strings.Contains(got, "ProjectName") || !strings.Contains(got, "WorkspaceCount") {
		t.Fatalf("WriteList() = %q, want Go-style field headers", got)
	}
	if strings.Contains(got, "PROJECT NAME") || strings.Contains(got, "WORKSPACE COUNT") {
		t.Fatalf("WriteList() = %q, want unsplit headers", got)
	}
}

func TestStructuredOutputUsesSnakeCaseFields(t *testing.T) {
	item := struct {
		ProjectID   string `json:"ProjectId"`
		ProjectName string `json:"ProjectName"`
		Compute     struct {
			MinCU float64 `json:"AutoScalingLimitMinCU"`
		} `json:"ComputeSettings"`
	}{
		ProjectID:   "p-1",
		ProjectName: "demo",
	}
	item.Compute.MinCU = 1

	var output bytes.Buffer
	w := New("json")
	w.Out = &output
	if err := w.WriteItem(item, nil); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["project_id"] != "p-1" || got["project_name"] != "demo" {
		t.Fatalf("unexpected normalized fields: %s", output.String())
	}
	nested, ok := got["compute_settings"].(map[string]any)
	if !ok || nested["auto_scaling_limit_min_cu"] != float64(1) {
		t.Fatalf("unexpected normalized nested fields: %s", output.String())
	}
	if _, found := got["ProjectId"]; found {
		t.Fatalf("structured output retained Go-style field name: %s", output.String())
	}
}

func TestSnakeCaseHandlesAcronyms(t *testing.T) {
	for input, want := range map[string]string{
		"ProjectId":             "project_id",
		"AutoScalingLimitMinCU": "auto_scaling_limit_min_cu",
		"HTTPStatusCode":        "http_status_code",
		"already_snake_case":    "already_snake_case",
	} {
		if got := snakeCase(input); got != want {
			t.Fatalf("snakeCase(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestOutputNormalizesTimestampPrecision(t *testing.T) {
	item := struct {
		CreateTime string `json:"CreateTime"`
		FinishTime string `json:"FinishTime"`
	}{
		CreateTime: "2026-08-09T10:52:29.000Z",
		FinishTime: "2026-08-09T10:53:29Z",
	}

	for _, format := range []string{"table", "csv", "tsv", "json", "yaml"} {
		t.Run(format, func(t *testing.T) {
			var output bytes.Buffer
			w := New(format)
			w.Out = &output
			if err := w.WriteItem(item, []string{"CreateTime", "FinishTime"}); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(output.String(), ".000Z") {
				t.Fatalf("timestamp retained millisecond precision: %q", output.String())
			}
			if !strings.Contains(output.String(), "2026-08-09T10:52:29Z") {
				t.Fatalf("normalized timestamp missing: %q", output.String())
			}
		})
	}
}

func TestPrettyOutputHumanizesWindowSizeSeconds(t *testing.T) {
	item := struct {
		WindowSizeSeconds int64 `json:"WindowSizeSeconds"`
	}{WindowSizeSeconds: 172800}

	var table bytes.Buffer
	tableWriter := New("table")
	tableWriter.Out = &table
	if err := tableWriter.WriteItem(item, []string{"WindowSizeSeconds"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(table.String(), "48h") || strings.Contains(table.String(), "172800") {
		t.Fatalf("table output = %q, want humanized duration", table.String())
	}

	var csv bytes.Buffer
	csvWriter := New("csv")
	csvWriter.Out = &csv
	if err := csvWriter.WriteItem(item, []string{"WindowSizeSeconds"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(csv.String(), "172800") || strings.Contains(csv.String(), "48h") {
		t.Fatalf("csv output = %q, want raw seconds", csv.String())
	}
}

func TestPrettyOutputNormalizesSentinelValues(t *testing.T) {
	item := struct {
		Plugin                string `json:"Plugin"`
		Database              string `json:"Database"`
		ConfirmedFlushLSN     string `json:"ConfirmedFlushLSN"`
		Rolconnlimit          int    `json:"Rolconnlimit"`
		SuspendTimeoutSeconds int    `json:"SuspendTimeoutSeconds"`
		Default               bool   `json:"Default"`
		Protected             bool   `json:"Protected"`
		StartParentTime       string `json:"StartParentTime"`
	}{
		Rolconnlimit:          -1,
		SuspendTimeoutSeconds: -1,
		StartParentTime:       "1970-01-01T00:00:00.000Z",
	}

	var output bytes.Buffer
	w := New("table")
	w.Out = &output
	if err := w.WriteItem(item, []string{
		"Plugin", "Database", "ConfirmedFlushLSN", "Rolconnlimit",
		"SuspendTimeoutSeconds", "Default", "Protected", "StartParentTime",
	}); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	for _, want := range []string{
		"Plugin                   -",
		"Database                 -",
		"ConfirmedFlushLSN        -",
		"Rolconnlimit             unlimited",
		"SuspendTimeoutSeconds    never",
		"Default                  false",
		"Protected                false",
		"StartParentTime          -",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("pretty output = %q, missing %q", got, want)
		}
	}
}

func TestWriteDelimitedFormats(t *testing.T) {
	items := []struct {
		ID   string `json:"ProjectId"`
		Name string `json:"ProjectName"`
	}{
		{ID: "p-1", Name: `demo, "primary"`},
	}
	for _, test := range []struct {
		format string
		want   string
	}{
		{format: "csv", want: "project_id,project_name\np-1,\"demo, \"\"primary\"\"\"\n"},
		{format: "tsv", want: "project_id\tproject_name\np-1\t\"demo, \"\"primary\"\"\"\n"},
	} {
		t.Run(test.format, func(t *testing.T) {
			var output bytes.Buffer
			w := New(test.format)
			w.Out = &output
			if err := w.WriteList(items, []string{"ProjectId", "ProjectName"}); err != nil {
				t.Fatal(err)
			}
			if output.String() != test.want {
				t.Fatalf("WriteList() = %q, want %q", output.String(), test.want)
			}
		})
	}
}

// WriteItem must never silently emit nothing for csv/tsv: the formats are
// advertised in every command's --help, so a detail command has to produce a
// header row and a value row even when the caller passes explicit fields.
func TestWriteItemDelimitedWithFields(t *testing.T) {
	item := struct {
		ID   string `json:"ProjectId"`
		Name string `json:"ProjectName"`
	}{ID: "p-1", Name: "demo"}

	for _, test := range []struct {
		format string
		want   string
	}{
		{format: "csv", want: "project_id,project_name\np-1,demo\n"},
		{format: "tsv", want: "project_id\tproject_name\np-1\tdemo\n"},
	} {
		t.Run(test.format, func(t *testing.T) {
			var output bytes.Buffer
			w := New(test.format)
			w.Out = &output
			if err := w.WriteItem(item, []string{"ProjectId", "ProjectName"}); err != nil {
				t.Fatal(err)
			}
			if output.String() != test.want {
				t.Fatalf("WriteItem() = %q, want %q", output.String(), test.want)
			}
		})
	}
}

// When fields are nil (status, monitoring), csv/tsv must derive columns from the
// object instead of printing an empty line with exit 0.
func TestWriteItemDelimitedWithoutFieldsDerivesColumns(t *testing.T) {
	item := struct {
		Auth struct {
			Site   string `json:"Site"`
			Active bool   `json:"Active"`
		} `json:"Authentication"`
		Count int `json:"Count"`
	}{}
	item.Auth.Site = "cn"
	item.Auth.Active = true
	item.Count = 3

	var output bytes.Buffer
	w := New("csv")
	w.Out = &output
	if err := w.WriteItem(item, nil); err != nil {
		t.Fatal(err)
	}
	// Columns are flattened with dotted keys and sorted for stability.
	want := "authentication.active,authentication.site,count\ntrue,cn,3\n"
	if output.String() != want {
		t.Fatalf("WriteItem() = %q, want %q", output.String(), want)
	}
}

func TestWriteItemKVWithoutFieldsIsNotEmpty(t *testing.T) {
	item := map[string]any{"Site": "cn"}
	var output bytes.Buffer
	w := New("table")
	w.Out = &output
	if err := w.WriteItem(item, nil); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(output.String()) == "" {
		t.Fatal("WriteItem() with nil fields produced empty KV output")
	}
	if !strings.Contains(output.String(), "site") || !strings.Contains(output.String(), "cn") {
		t.Fatalf("WriteItem() KV output missing derived field: %q", output.String())
	}
}
