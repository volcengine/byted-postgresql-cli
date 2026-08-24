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
	"strings"
	"testing"

	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

func TestEndpointTableRowsFormatsAddresses(t *testing.T) {
	rows := endpointTableRows([]volcengine.Endpoint{{
		EndpointID:   "ep-1",
		EndpointName: "primary",
		EndpointType: "Proxy",
		Addresses: []volcengine.EndpointAddress{
			{AddressType: "Public", AddressDomain: "public.example.com", AddressPort: 5432, IPAddress: "1.2.3.4"},
			{AddressType: "Inner", AddressDomain: "inner.example.com", AddressPort: 5432, IPAddress: "10.0.0.1"},
		},
	}})
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if strings.Contains(rows[0].Addresses, "[{") {
		t.Fatalf("addresses still look like a Go structure: %q", rows[0].Addresses)
	}
	for _, want := range []string{"Public public.example.com:5432 (1.2.3.4)", "Inner inner.example.com:5432 (10.0.0.1)"} {
		if !strings.Contains(rows[0].Addresses, want) {
			t.Fatalf("addresses = %q, want %q", rows[0].Addresses, want)
		}
	}
}
