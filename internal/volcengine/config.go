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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// DefaultRegion is the Volcengine region used when none is configured.
	DefaultRegion = "cn-beijing"

	// Volcengine credential + routing environment variables. Both the short and
	// long forms are accepted to match the official volcengine CLI/SDK.
	EnvAccessKeyID     = "VOLCENGINE_ACCESS_KEY"
	EnvSecretAccessKey = "VOLCENGINE_SECRET_KEY"
	EnvSessionToken    = "VOLCENGINE_SESSION_TOKEN"
	EnvRegion          = "VOLC_REGION"
	EnvEndpoint        = "VOLC_ENDPOINT"
	EnvProfile         = "VOLC_PROFILE"
	EnvLongRegion      = "VOLCENGINE_REGION"
	EnvLongEndpoint    = "VOLCENGINE_ENDPOINT"

	// ModeAK is a static Access Key / Secret Key profile.
	ModeAK = "ak"
	// ModeConsoleLogin is a browser Console Login profile whose temporary STS
	// credentials are cached under ~/.volcengine/login/cache and refreshed on
	// demand (see console_login.go).
	ModeConsoleLogin = "console-login"
)

// ErrNoCredentials is returned when no credentials could be resolved from env
// or the on-disk profile. The message points the user at browser login (the
// common path) as well as the AK/SK and environment-variable alternatives.
var ErrNoCredentials = fmt.Errorf(
	"missing Volcengine credentials; run `byted-postgresql-cli login` to authenticate in your browser, or `byted-postgresql-cli configure set --access-key <key> --secret-key <secret> --region <region>`, or set %s and %s",
	EnvAccessKeyID, EnvSecretAccessKey)

var (
	regionOverride  string
	profileOverride string
)

// Config is the resolved AK/SK credential + region for a single Client.
type Config struct {
	Provider        Provider
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Region          string
	Endpoint        string
}

// FileConfig is the on-disk profile store (~/.volcengine/config.json), shared
// with the official volcengine CLI so profiles configured there are reused.
type FileConfig struct {
	Current  string              `json:"current"`
	Profiles map[string]*Profile `json:"profiles"`
}

// Profile is one named credential set. AK/SK profiles carry AccessKey/SecretKey
// directly; console-login profiles carry only a LoginSession that points at the
// cached STS token bundle (see console_login.go).
type Profile struct {
	Name         string `json:"name"`
	Mode         string `json:"mode"`
	AccessKey    string `json:"access-key"`
	SecretKey    string `json:"secret-key"`
	Region       string `json:"region"`
	Endpoint     string `json:"endpoint"`
	SessionToken string `json:"session-token"`
	// LoginSession identifies the cached Console Login STS token for
	// console-login profiles. It is empty for AK/SK profiles.
	LoginSession string   `json:"login-session,omitempty"`
	Provider     Provider `json:"provider,omitempty"`
}

// ResolveConfig builds a Config for API calls. Credentials come from env vars
// first, then the selected profile in ~/.volcengine/config.json. The region
// comes from --region/env/profile (in that order), defaulting to DefaultRegion.
//
// The ctx argument is accepted for symmetry with the previous site-based
// resolver and future credential providers; it is currently unused.
func ResolveConfig(ctx context.Context) (Config, error) {
	_ = ctx
	provider := ProviderVolcengine
	spec, err := ProviderSpecFor(provider)
	if err != nil {
		return Config{}, err
	}
	cfg := Config{
		Provider:        provider,
		AccessKeyID:     strings.TrimSpace(os.Getenv(accessKeyEnv(provider))),
		SecretAccessKey: strings.TrimSpace(os.Getenv(secretKeyEnv(provider))),
		SessionToken:    strings.TrimSpace(os.Getenv(sessionTokenEnv(provider))),
		Region:          regionSettingFor(provider),
		Endpoint:        firstEnv(endpointEnv(provider)...),
	}

	// Env AK/SK win outright when either is set.
	if cfg.AccessKeyID != "" || cfg.SecretAccessKey != "" {
		if cfg.AccessKeyID == "" {
			return Config{}, fmt.Errorf("missing %s environment variable", EnvAccessKeyID)
		}
		if cfg.SecretAccessKey == "" {
			return Config{}, fmt.Errorf("missing %s environment variable", EnvSecretAccessKey)
		}
	} else {
		fileCfg, profileName, profile, err := LoadSelectedProfileFor(provider)
		if err != nil {
			return Config{}, err
		}
		if profile == nil {
			return Config{}, ErrNoCredentials
		}
		switch strings.ToLower(strings.TrimSpace(profile.Mode)) {
		case "", ModeAK:
			cfg.AccessKeyID = strings.TrimSpace(profile.AccessKey)
			cfg.SecretAccessKey = strings.TrimSpace(profile.SecretKey)
			cfg.SessionToken = strings.TrimSpace(profile.SessionToken)
		case ModeConsoleLogin:
			// Browser Console Login profiles hold no static AK/SK; resolve (and
			// transparently refresh) the cached temporary STS credentials.
			creds, err := EnsureValidLoginToken(provider, fileCfg, profileName)
			if err != nil {
				return Config{}, err
			}
			cfg.AccessKeyID = creds.AccessKeyID
			cfg.SecretAccessKey = creds.SecretAccessKey
			cfg.SessionToken = creds.SessionToken
		default:
			return Config{}, fmt.Errorf("unsupported Volcengine profile mode %q; only AK/SK and console-login profiles are supported", profile.Mode)
		}
		if cfg.Region == "" {
			cfg.Region = strings.TrimSpace(profile.Region)
		}
		if cfg.Endpoint == "" {
			cfg.Endpoint = strings.TrimSpace(profile.Endpoint)
		}
		if cfg.Provider == "" {
			cfg.Provider = profile.Provider
		}
		if cfg.AccessKeyID == "" {
			return Config{}, fmt.Errorf("missing access-key in Volcengine profile")
		}
		if cfg.SecretAccessKey == "" {
			return Config{}, fmt.Errorf("missing secret-key in Volcengine profile")
		}
	}

	if cfg.Region == "" {
		cfg.Region = spec.DefaultRegion
	}
	if err := ValidateProviderRegion(provider, cfg.Region); err != nil {
		return Config{}, err
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = spec.AIDAPEndpoint(cfg.Region)
	}
	return cfg, nil
}

// VolcengineConfigFilePath returns the shared profile file path.
func VolcengineConfigFilePath() (string, error) {
	return ConfigFilePath(ProviderVolcengine)
}

func ConfigFilePath(provider Provider) (string, error) {
	if provider != ProviderVolcengine {
		return "", fmt.Errorf("unsupported provider %q", provider)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get $HOME directory: %w", err)
	}
	return filepath.Join(home, volcengineSpec.ConfigDir, "config.json"), nil
}

// LoadFileConfig reads the profile store, returning an empty config when the
// file does not exist yet.
func LoadFileConfig() (FileConfig, error) {
	return LoadFileConfigFor(ProviderVolcengine)
}

func LoadFileConfigFor(provider Provider) (FileConfig, error) {
	path, err := ConfigFilePath(provider)
	if err != nil {
		return FileConfig{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return FileConfig{Current: "default", Profiles: map[string]*Profile{}}, nil
	}
	if err != nil {
		return FileConfig{}, fmt.Errorf("failed to read Volcengine config: %w", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return FileConfig{Current: "default", Profiles: map[string]*Profile{}}, nil
	}
	var cfg FileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return FileConfig{}, fmt.Errorf("failed to parse Volcengine config: %w", err)
	}
	if cfg.Current == "" {
		cfg.Current = "default"
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]*Profile{}
	}
	return cfg, nil
}

// SaveFileConfig atomically writes the profile store with owner-only perms.
func SaveFileConfig(cfg FileConfig) error {
	return SaveFileConfigFor(ProviderVolcengine, cfg)
}

func SaveFileConfigFor(provider Provider, cfg FileConfig) error {
	if provider != ProviderVolcengine {
		return fmt.Errorf("unsupported provider %q", provider)
	}
	path, err := ConfigFilePath(provider)
	if err != nil {
		return err
	}
	if cfg.Current == "" {
		cfg.Current = "default"
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]*Profile{}
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create Volcengine config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode Volcengine config: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "config-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary Volcengine config: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write temporary Volcengine config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temporary Volcengine config: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to save Volcengine config: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("failed to chmod Volcengine config: %w", err)
	}
	return nil
}

// LoadCurrentProfile returns the currently selected profile, or nil if none.
func LoadCurrentProfile() (*Profile, error) {
	_, _, profile, err := LoadSelectedProfile()
	return profile, err
}

func LoadCurrentProfileFor(provider Provider) (*Profile, error) {
	_, _, profile, err := LoadSelectedProfileFor(provider)
	return profile, err
}

// LoadSelectedProfile resolves the effective profile name (override > env >
// file "current" > "default") and returns the matching profile.
func LoadSelectedProfile() (FileConfig, string, *Profile, error) {
	return LoadSelectedProfileFor(ProviderVolcengine)
}

func LoadSelectedProfileFor(provider Provider) (FileConfig, string, *Profile, error) {
	cfg, err := LoadFileConfigFor(provider)
	if err != nil {
		return FileConfig{}, "", nil, err
	}
	profileName := profileOverride
	if profileName == "" {
		profileName = firstEnv(profileEnv(provider)...)
	}
	if profileName == "" {
		profileName = cfg.Current
	}
	if profileName == "" {
		profileName = "default"
	}
	profile, ok := cfg.Profiles[profileName]
	if !ok {
		return cfg, profileName, nil, nil
	}
	return cfg, profileName, profile, nil
}

// ValidateProfileForProvider rejects an explicitly selected profile that
// belongs to a different provider's isolated config store.
func ValidateProfileForProvider(provider Provider, profileName string) error {
	profileName = strings.TrimSpace(profileName)
	if profileName == "" {
		return nil
	}
	if _, err := ProviderSpecFor(provider); err != nil {
		return err
	}
	current, err := LoadFileConfigFor(provider)
	if err != nil {
		return err
	}
	if profile := current.Profiles[profileName]; profile != nil {
		if profile.Provider != "" && profile.Provider != provider {
			return fmt.Errorf("profile %q belongs to %s, but the selected region uses the %s provider", profileName, profile.Provider, provider)
		}
		return nil
	}
	return nil
}

// SetRegionOverride records the --region flag value for RegionSetting.
func SetRegionOverride(region string) { regionOverride = strings.TrimSpace(region) }

// SetProfileOverride records the --profile flag value for profile resolution.
func SetProfileOverride(profile string) { profileOverride = strings.TrimSpace(profile) }

// HasRegionOverride reports whether --region was supplied.
func HasRegionOverride() bool { return regionOverride != "" }

// RegionSetting resolves the effective Volcengine region: --region > env > profile.
func RegionSetting() string {
	return regionSettingFor(ProviderVolcengine)
}

func regionSettingFor(provider Provider) string {
	if regionOverride != "" {
		return regionOverride
	}
	if region := firstEnv(regionEnv(provider)...); region != "" {
		return region
	}
	profile, err := LoadCurrentProfileFor(provider)
	if err == nil && profile != nil {
		return strings.TrimSpace(profile.Region)
	}
	return ""
}

// ValidateRegion checks the region resolves for the aidap service.
func ValidateRegion(region string) error {
	return ValidateProviderRegion(ProviderVolcengine, region)
}

func accessKeyEnv(provider Provider) string {
	return EnvAccessKeyID
}

func AccessKeyEnvironment(provider Provider) string { return accessKeyEnv(provider) }

func secretKeyEnv(provider Provider) string {
	return EnvSecretAccessKey
}

func SecretKeyEnvironment(provider Provider) string { return secretKeyEnv(provider) }

func sessionTokenEnv(provider Provider) string {
	return EnvSessionToken
}

func regionEnv(provider Provider) []string {
	return []string{EnvLongRegion, EnvRegion}
}

func RegionEnvironment(provider Provider) string {
	values := regionEnv(provider)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func endpointEnv(provider Provider) []string {
	return []string{EnvLongEndpoint, EnvEndpoint}
}

func profileEnv(provider Provider) []string {
	return []string{EnvProfile}
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}
