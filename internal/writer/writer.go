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
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v3"
)

type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
	FormatCSV   Format = "csv"
	FormatTSV   Format = "tsv"
)

type Writer struct {
	Format Format
	Out    io.Writer
}

func ValidateFormat(format string) error {
	f := Format(strings.ToLower(strings.TrimSpace(format)))
	switch f {
	case FormatJSON, FormatYAML, FormatTable, FormatCSV, FormatTSV:
		return nil
	default:
		return fmt.Errorf("unsupported output format %q; choose table, json, yaml, csv, or tsv", format)
	}
}

func New(format string) *Writer {
	f := Format(strings.ToLower(strings.TrimSpace(format)))
	switch f {
	case FormatJSON, FormatYAML, FormatTable, FormatCSV, FormatTSV:
	default:
		f = FormatTable
	}
	return &Writer{Format: f, Out: os.Stdout}
}

// WriteList writes a slice of items with the given field names as columns for
// the table format. For JSON/YAML the full item is dumped.
func (w *Writer) WriteList(items any, fields []string) error {
	items = normalizeList(items)
	normalized, err := normalizeOutput(items)
	if err != nil {
		return err
	}
	switch w.Format {
	case FormatJSON:
		enc := json.NewEncoder(w.Out)
		enc.SetIndent("", "  ")
		return enc.Encode(normalized)
	case FormatYAML:
		yamlValue, err := normalizeYAMLOutput(items)
		if err != nil {
			return err
		}
		return yaml.NewEncoder(w.Out).Encode(yamlValue)
	case FormatCSV, FormatTSV:
		return w.writeDelimited(items, fields, w.Format == FormatTSV)
	default:
		return w.writeTable(items, fields)
	}
}

func (w *Writer) writeDelimited(items any, fields []string, tabSeparated bool) error {
	v := reflect.ValueOf(items)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("writer: not a slice (%s)", v.Kind())
	}
	delimiter := ','
	if tabSeparated {
		delimiter = '\t'
	}
	output := csv.NewWriter(w.Out)
	output.Comma = delimiter
	headers := make([]string, len(fields))
	for i, field := range fields {
		headers[i] = snakeCase(field)
	}
	if err := output.Write(headers); err != nil {
		return err
	}
	for i := 0; i < v.Len(); i++ {
		if err := output.Write(extractRow(v.Index(i), fields)); err != nil {
			return err
		}
	}
	output.Flush()
	return output.Error()
}

// WriteItem writes a single object.
func (w *Writer) WriteItem(item any, fields []string) error {
	normalized, err := normalizeOutput(item)
	if err != nil {
		return err
	}
	switch w.Format {
	case FormatJSON:
		enc := json.NewEncoder(w.Out)
		enc.SetIndent("", "  ")
		return enc.Encode(normalized)
	case FormatYAML:
		yamlValue, err := normalizeYAMLOutput(item)
		if err != nil {
			return err
		}
		return yaml.NewEncoder(w.Out).Encode(yamlValue)
	case FormatCSV, FormatTSV:
		// csv/tsv are advertised for every command, so a single object must
		// emit a real header+row. When the caller passes no fields, derive
		// them (and the values) from the normalized object so the output is
		// never silently empty.
		return w.writeDelimitedItem(normalized, item, fields, w.Format == FormatTSV)
	default:
		return w.writeKV(normalized, item, fields)
	}
}

// writeDelimitedItem writes one object as a header row plus a single value row.
func (w *Writer) writeDelimitedItem(normalized any, item any, fields []string, tabSeparated bool) error {
	var headers, values []string
	if len(fields) > 0 {
		// Reuse the same extraction the table and list CSV paths use so a
		// detail command renders identically across formats.
		headers = make([]string, len(fields))
		for i, f := range fields {
			headers[i] = snakeCase(f)
		}
		v := reflect.ValueOf(item)
		values = extractRow(v, fields)
	} else {
		headers, values = itemColumns(normalized)
	}
	delimiter := ','
	if tabSeparated {
		delimiter = '\t'
	}
	output := csv.NewWriter(w.Out)
	output.Comma = delimiter
	if err := output.Write(headers); err != nil {
		return err
	}
	if err := output.Write(values); err != nil {
		return err
	}
	output.Flush()
	return output.Error()
}

func (w *Writer) writeTable(items any, fields []string) error {
	v := reflect.ValueOf(items)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("writer: not a slice (%s)", v.Kind())
	}
	t := tablewriter.NewWriter(w.Out)
	headers := make([]string, len(fields))
	for i, f := range fields {
		headers[i] = f
	}
	t.SetHeader(headers)
	t.SetAutoFormatHeaders(false)
	t.SetAutoWrapText(false)
	t.SetBorder(false)
	t.SetHeaderLine(false)
	t.SetColumnSeparator(" ")
	if v.Len() == 0 {
		fmt.Fprintln(w.Out, "No results found.")
		return nil
	}
	for i := 0; i < v.Len(); i++ {
		row := extractRow(v.Index(i), fields)
		t.Append(row)
	}
	t.Render()
	return nil
}

func normalizeList(items any) any {
	if items == nil {
		return []any{}
	}
	v := reflect.ValueOf(items)
	if v.Kind() == reflect.Slice && v.IsNil() {
		return reflect.MakeSlice(v.Type(), 0, 0).Interface()
	}
	return items
}

func normalizeOutput(value any) (any, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode structured output: %w", err)
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil, fmt.Errorf("decode structured output: %w", err)
	}
	return normalizeOutputKeys(decoded), nil
}

func normalizeOutputKeys(value any) any {
	switch value := value.(type) {
	case []any:
		for i := range value {
			value[i] = normalizeOutputKeys(value[i])
		}
		return value
	case map[string]any:
		normalized := make(map[string]any, len(value))
		for key, item := range value {
			normalizedKey := snakeCase(key)
			item = normalizeOutputKeys(item)
			if isTimestampField(normalizedKey) {
				item = normalizeTimestampValue(item)
			}
			normalized[normalizedKey] = item
		}
		return normalized
	default:
		return value
	}
}

func normalizeYAMLOutput(value any) (any, error) {
	return normalizeYAMLValue(reflect.ValueOf(value))
}

func normalizeYAMLValue(value reflect.Value) (any, error) {
	if !value.IsValid() {
		return nil, nil
	}
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if !value.IsValid() {
			return nil, nil
		}
		if value.IsNil() {
			return nil, nil
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Struct:
		result := make(map[string]any)
		typeOfValue := value.Type()
		for i := 0; i < value.NumField(); i++ {
			field := typeOfValue.Field(i)
			if field.PkgPath != "" {
				continue
			}
			name, options := structuredFieldName(field)
			if name == "-" || (options["omitempty"] && value.Field(i).IsZero()) {
				continue
			}
			if !value.Field(i).IsValid() || !value.Field(i).CanInterface() {
				continue
			}
			normalized, err := normalizeYAMLValue(value.Field(i))
			if err != nil {
				return nil, err
			}
			if isTimestampField(name) {
				normalized = normalizeTimestampValue(normalized)
			}
			result[name] = normalized
		}
		return result, nil
	case reflect.Slice, reflect.Array:
		result := make([]any, value.Len())
		for i := 0; i < value.Len(); i++ {
			normalized, err := normalizeYAMLValue(value.Index(i))
			if err != nil {
				return nil, err
			}
			result[i] = normalized
		}
		return result, nil
	case reflect.Map:
		result := make(map[string]any, value.Len())
		iter := value.MapRange()
		for iter.Next() {
			keyValue := iter.Key()
			if !keyValue.IsValid() || !keyValue.CanInterface() {
				continue
			}
			key := fmt.Sprint(keyValue.Interface())
			normalized, err := normalizeYAMLValue(iter.Value())
			if err != nil {
				return nil, err
			}
			if isTimestampField(snakeCase(key)) {
				normalized = normalizeTimestampValue(normalized)
			}
			result[snakeCase(key)] = normalized
		}
		return result, nil
	default:
		if !value.IsValid() || !value.CanInterface() {
			return nil, nil
		}
		return value.Interface(), nil
	}
}

func structuredFieldName(field reflect.StructField) (string, map[string]bool) {
	options := make(map[string]bool)
	name := field.Name
	tag := field.Tag.Get("json")
	if tag != "" {
		parts := strings.Split(tag, ",")
		if parts[0] != "" {
			name = parts[0]
		}
		for _, option := range parts[1:] {
			options[option] = true
		}
	}
	return snakeCase(name), options
}

func snakeCase(value string) string {
	var out strings.Builder
	runes := []rune(value)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				previous := runes[i-1]
				var next rune
				if i+1 < len(runes) {
					next = runes[i+1]
				}
				if unicode.IsLower(previous) || unicode.IsDigit(previous) ||
					(unicode.IsUpper(previous) && unicode.IsLower(next)) {
					out.WriteByte('_')
				}
			}
			out.WriteRune(unicode.ToLower(r))
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}

func (w *Writer) writeKV(normalized any, item any, fields []string) error {
	// With explicit fields, keep the original struct/map extraction so callers
	// control column order and naming. With no fields, fall back to the
	// normalized object so a detail command is never silently blank.
	if len(fields) == 0 {
		headers, values := itemColumns(normalized)
		for i, f := range headers {
			fmt.Fprintf(w.Out, "%-24s %s\n", f, normalizePrettyValue(f, values[i]))
		}
		return nil
	}
	v := reflect.ValueOf(item)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	row := extractRow(v, fields)
	for i, f := range fields {
		value := normalizePrettyValue(f, row[i])
		fmt.Fprintf(w.Out, "%-24s %s\n", f, value)
	}
	return nil
}

// itemColumns derives an alphabetically stable set of columns and their string
// values from a normalized (map-shaped) object, flattening nested objects into
// dotted keys so nothing is dropped. Scalars and slices have no field names, so
// they collapse to a single "value" column.
func itemColumns(normalized any) ([]string, []string) {
	flat := map[string]string{}
	switch m := normalized.(type) {
	case map[string]any:
		flattenObject("", m, flat)
	default:
		return []string{"value"}, []string{scalarString(normalized)}
	}
	headers := make([]string, 0, len(flat))
	for k := range flat {
		headers = append(headers, k)
	}
	sort.Strings(headers)
	values := make([]string, len(headers))
	for i, h := range headers {
		values[i] = flat[h]
	}
	return headers, values
}

// flattenObject walks a normalized object, joining nested keys with dots so
// {"auth":{"site":"cn"}} becomes {"auth.site":"cn"}.
func flattenObject(prefix string, value map[string]any, out map[string]string) {
	for key, item := range value {
		full := key
		if prefix != "" {
			full = prefix + "." + key
		}
		if nested, ok := item.(map[string]any); ok {
			flattenObject(full, nested, out)
			continue
		}
		out[full] = scalarString(item)
	}
}

func scalarString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case float64:
		// JSON numbers decode to float64; render integers without a trailing .0.
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return fmt.Sprint(v)
	}
}

func extractRow(v reflect.Value, fields []string) []string {
	out := make([]string, len(fields))
	for v.IsValid() && (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return out
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return out
	}
	if v.Kind() == reflect.Map {
		for i, f := range fields {
			mv := v.MapIndex(reflect.ValueOf(f))
			if mv.IsValid() && mv.CanInterface() {
				out[i] = fmt.Sprint(mv.Interface())
				if isTimestampField(snakeCase(f)) {
					out[i] = normalizeTimestamp(out[i])
				}
			}
		}
		return out
	}
	if v.Kind() != reflect.Struct {
		for i := range fields {
			if v.CanInterface() {
				out[i] = fmt.Sprint(v.Interface())
			}
		}
		return out
	}
	t := v.Type()
	for i, f := range fields {
		out[i] = ""
		// Match by json tag or field name (case-insensitive).
		for j := 0; j < t.NumField(); j++ {
			sf := t.Field(j)
			tag := strings.Split(sf.Tag.Get("json"), ",")[0]
			if tag == f || strings.EqualFold(sf.Name, f) {
				fv := v.Field(j)
				if fv.IsValid() && fv.Kind() == reflect.Bool {
					out[i] = strconv.FormatBool(fv.Bool())
				} else if fv.IsValid() && fv.CanInterface() {
					out[i] = fmt.Sprint(fv.Interface())
				}
				if isTimestampField(snakeCase(f)) {
					out[i] = normalizeTimestamp(out[i])
				}
				break
			}
		}
	}
	return out
}

var timestampFields = map[string]struct{}{
	"branch_create_time": {},
	"create_time":        {},
	"end_time":           {},
	"finish_time":        {},
	"last_analyze":       {},
	"last_autoanalyze":   {},
	"last_autovacuum":    {},
	"last_vacuum":        {},
	"start_parent_time":  {},
	"start_time":         {},
	"update_time":        {},
}

func isTimestampField(field string) bool {
	_, ok := timestampFields[field]
	return ok
}

func normalizeTimestampValue(value any) any {
	if text, ok := value.(string); ok {
		return normalizeTimestamp(text)
	}
	return value
}

func normalizeTimestamp(value string) string {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	return parsed.Format(time.RFC3339)
}

func humanDurationSeconds(value string) string {
	seconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || seconds < 0 {
		return value
	}
	if seconds%3600 == 0 {
		return fmt.Sprintf("%dh", seconds/3600)
	}
	if seconds%60 == 0 {
		return fmt.Sprintf("%dm", seconds/60)
	}
	return fmt.Sprintf("%ds", seconds)
}

func normalizePrettyValue(field, value string) string {
	field = snakeCase(field)
	value = strings.TrimSpace(value)
	if value == "" || value == "<nil>" {
		return "-"
	}
	switch field {
	case "rolconnlimit":
		if value == "-1" {
			return "unlimited"
		}
	case "suspend_timeout_seconds":
		if value == "-1" {
			return "never"
		}
	case "window_size_seconds":
		return humanDurationSeconds(value)
	}
	if isTimestampField(field) {
		parsed, err := time.Parse(time.RFC3339Nano, value)
		if err == nil {
			if parsed.Year() == 1970 {
				return "-"
			}
			return parsed.Format(time.RFC3339)
		}
	}
	return value
}
