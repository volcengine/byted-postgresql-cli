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
	"fmt"

	"github.com/spf13/cobra"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

type listPageOutput struct {
	Items      any  `json:"items" yaml:"items"`
	Count      int  `json:"count" yaml:"count"`
	Total      int  `json:"total" yaml:"total"`
	Limit      int  `json:"limit" yaml:"limit"`
	Offset     int  `json:"offset" yaml:"offset"`
	NextOffset *int `json:"next_offset,omitempty" yaml:"next_offset,omitempty"`
}

func requestedListLimit(cmd *cobra.Command, limit int) int {
	if limit == 0 && cmd.Flags().Changed("limit") {
		return volcengine.MaxPageLimit
	}
	return limit
}

func writeListPage(cmd *cobra.Command, g *Globals, items any, fields []string, total, limit, offset int, noun string) error {
	shown := listLength(items)
	limit = requestedListLimit(cmd, limit)
	if limit <= 0 {
		limit = volcengine.DefaultListLimit
	}
	var nextOffset *int
	if total > offset+shown {
		next := offset + shown
		nextOffset = &next
	}
	if g.Writer().Format == "json" || g.Writer().Format == "yaml" {
		return g.Writer().WriteItem(listPageOutput{
			Items: items, Count: shown, Total: total, Limit: limit, Offset: offset, NextOffset: nextOffset,
		}, nil)
	}
	if err := g.Writer().WriteList(items, fields); err != nil {
		return err
	}
	if g.Writer().Format == "csv" || g.Writer().Format == "tsv" {
		return nil
	}
	if nextOffset == nil {
		return nil
	}
	remaining := total - offset - shown
	summary := fmt.Sprintf("Showing %d of %d %s (offset=%d, limit=%d). %d remaining; use --offset %d --limit %d to fetch the next page",
		shown, total, noun, offset, limit, remaining, *nextOffset, limit)
	fmt.Fprintln(cmd.OutOrStdout(), summary+".")
	return nil
}

func writeListSummary(cmd *cobra.Command, g *Globals, items any, fields []string, total *int, noun string) error {
	count := listLength(items)
	if g.Writer().Format == "json" || g.Writer().Format == "yaml" {
		return g.Writer().WriteItem(struct {
			Items any  `json:"items" yaml:"items"`
			Count int  `json:"count" yaml:"count"`
			Total *int `json:"total,omitempty" yaml:"total,omitempty"`
		}{Items: items, Count: count, Total: total}, nil)
	}
	if g.Writer().Format == "csv" || g.Writer().Format == "tsv" {
		return g.Writer().WriteList(items, fields)
	}
	if total == nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Showing %d %s.\n", count, noun)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Showing %d of %d %s.\n", count, *total, noun)
	}
	return g.Writer().WriteList(items, fields)
}

func listLength(items any) int {
	switch value := items.(type) {
	case []volcengine.Workspace:
		return len(value)
	case []volcengine.Branch:
		return len(value)
	case []volcengine.Database:
		return len(value)
	case []volcengine.DBAccount:
		return len(value)
	case []volcengine.Operation:
		return len(value)
	case []volcengine.Compute:
		return len(value)
	case []volcengine.Endpoint:
		return len(value)
	default:
		return 0
	}
}
