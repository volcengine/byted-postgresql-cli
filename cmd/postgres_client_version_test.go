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

import "testing"

func TestPostgresMajorVersion(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int
		ok    bool
	}{
		{name: "server engine version", value: "PostgreSQL_17", want: 17, ok: true},
		{name: "pg_dump output", value: "pg_dump (PostgreSQL) 15.18", want: 15, ok: true},
		{name: "psql output", value: "psql (PostgreSQL) 17.5", want: 17, ok: true},
		{name: "unknown", value: "not available", ok: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := postgresMajorVersion(test.value)
			if got != test.want || ok != test.ok {
				t.Fatalf("postgresMajorVersion(%q) = %d, %v; want %d, %v", test.value, got, ok, test.want, test.ok)
			}
		})
	}
}
