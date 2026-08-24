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

package pagination

import "fmt"

const (
	DefaultLimit = 10
	MaxLimit     = 100
	MaxOffset    = 100000
)

// Policy contains the client-side pagination contract shared by CLI and MCP.
// A zero limit means "use the default"; a negative limit is invalid.
type Policy struct {
	DefaultLimit int
	MaxLimit     int
	MaxOffset    int
}

var DefaultPolicy = Policy{
	DefaultLimit: DefaultLimit,
	MaxLimit:     MaxLimit,
	MaxOffset:    MaxOffset,
}

type Page struct {
	Limit  int
	Offset int
}

type Metadata struct {
	HasMore    bool
	NextOffset *int
}

func (p Policy) Normalize(limit, offset int) (Page, error) {
	if p.DefaultLimit <= 0 || p.MaxLimit <= 0 || p.MaxOffset < 0 {
		return Page{}, fmt.Errorf("invalid pagination policy")
	}
	if limit < 0 || limit > p.MaxLimit {
		return Page{}, fmt.Errorf("limit must be between 0 and %d", p.MaxLimit)
	}
	if offset < 0 || offset > p.MaxOffset {
		return Page{}, fmt.Errorf("offset must be between 0 and %d", p.MaxOffset)
	}
	if limit == 0 {
		limit = p.DefaultLimit
	}
	return Page{Limit: limit, Offset: offset}, nil
}

func (p Page) Metadata(count, total int) Metadata {
	next := p.Offset + count
	if count == 0 || next >= total {
		return Metadata{}
	}
	return Metadata{HasMore: true, NextOffset: &next}
}
