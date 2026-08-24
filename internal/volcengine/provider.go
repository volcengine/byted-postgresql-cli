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
	"fmt"
	"strings"

	"github.com/volcengine/volcengine-go-sdk/service/aidap"
	"github.com/volcengine/volcengine-go-sdk/volcengine/endpoints"
)

type Provider string

const (
	ProviderVolcengine Provider = "volcengine"
)

type ProviderSpec struct {
	Provider        Provider
	DisplayName     string
	DefaultRegion   string
	ConsoleEndpoint string
	ConfigDir       string
	EnvPrefix       string
	AIDAPEndpoint   func(string) string
}

var volcengineSpec = ProviderSpec{
	Provider:        ProviderVolcengine,
	DisplayName:     "Volcengine",
	DefaultRegion:   "cn-beijing",
	ConsoleEndpoint: DefaultConsoleEndpoint,
	ConfigDir:       ".volcengine",
	EnvPrefix:       "VOLCENGINE",
	AIDAPEndpoint:   func(string) string { return "" },
}

func (p Provider) String() string { return string(p) }

func ProviderSpecFor(provider Provider) (ProviderSpec, error) {
	if provider != ProviderVolcengine {
		return ProviderSpec{}, fmt.Errorf("unsupported provider %q", provider)
	}
	return volcengineSpec, nil
}

func ValidateProviderRegion(provider Provider, region string) error {
	region = strings.TrimSpace(region)
	if region == "" {
		return fmt.Errorf("region cannot be empty")
	}
	if provider != ProviderVolcengine {
		return fmt.Errorf("unsupported provider %q", provider)
	}
	if _, err := endpoints.NewStandardEndpointResolver().EndpointFor(aidap.ServiceName, region); err != nil {
		return fmt.Errorf("invalid Volcengine region %q: %w", region, err)
	}
	return nil
}

func ProviderForConfig(provider, region string) (Provider, error) {
	if strings.TrimSpace(provider) != "" {
		p := Provider(strings.ToLower(strings.TrimSpace(provider)))
		if _, err := ProviderSpecFor(p); err != nil {
			return "", err
		}
		if err := ValidateProviderRegion(p, region); err != nil {
			return "", err
		}
		return p, nil
	}
	return ProviderVolcengine, nil
}
