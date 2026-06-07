/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	asimfs "bennypowers.dev/asimonim/fs"
	"github.com/tidwall/jsonc"
)

// configKeys are the package.json keys to check, in priority order.
var configKeys = []string{"asimonim", "designTokensLanguageServer", "design-tokens-language-server"}

// LoadFromPackageJSON reads design tokens configuration from package.json.
// Checks "asimonim", "designTokensLanguageServer", and "design-tokens-language-server"
// keys in priority order. Returns nil if no config found (not an error).
func LoadFromPackageJSON(filesystem asimfs.FileSystem, rootDir string) (*Config, error) {
	if rootDir == "" {
		return nil, nil
	}

	pkgPath := filepath.Join(rootDir, "package.json")
	if !filesystem.Exists(pkgPath) {
		return nil, nil
	}

	data, err := filesystem.ReadFile(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json: %w", err)
	}

	data = jsonc.ToJSON(data)

	var pkgJSON map[string]json.RawMessage
	if err := json.Unmarshal(data, &pkgJSON); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}

	var configRaw json.RawMessage
	var configKey string
	for _, key := range configKeys {
		if raw, ok := pkgJSON[key]; ok {
			configRaw = raw
			configKey = key
			break
		}
	}

	if configRaw == nil {
		return nil, nil
	}

	var configMap map[string]json.RawMessage
	if err := json.Unmarshal(configRaw, &configMap); err != nil {
		return nil, fmt.Errorf("%s must be an object", configKey)
	}

	return buildConfigFromMap(configMap)
}

func buildConfigFromMap(m map[string]json.RawMessage) (*Config, error) {
	cfg := &Config{}

	if raw, ok := m["prefix"]; ok {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			cfg.Prefix = s
		}
	}

	if raw, ok := m["cdn"]; ok {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			cfg.CDN = s
		}
	}

	if raw, ok := m["schema"]; ok {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			cfg.Schema = s
		}
	}

	if raw, ok := m["groupMarkers"]; ok {
		var markers []string
		if err := json.Unmarshal(raw, &markers); err == nil {
			cfg.GroupMarkers = markers
		}
	}

	if raw, ok := m["resolvers"]; ok {
		var resolvers []string
		if err := json.Unmarshal(raw, &resolvers); err == nil {
			cfg.Resolvers = resolvers
		}
	}

	if raw, ok := m["tokensFiles"]; ok {
		files, err := parseTokensFiles(raw)
		if err != nil {
			return nil, err
		}
		cfg.Files = files
	}

	if raw, ok := m["files"]; ok && cfg.Files == nil {
		files, err := parseTokensFiles(raw)
		if err != nil {
			return nil, err
		}
		cfg.Files = files
	}

	return cfg, nil
}

func parseTokensFiles(raw json.RawMessage) ([]FileSpec, error) {
	// Try as single string
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		return []FileSpec{{Path: single}}, nil
	}

	// Try as array
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("tokensFiles must be a string or array")
	}

	specs := make([]FileSpec, 0, len(arr))
	for _, item := range arr {
		var s string
		if err := json.Unmarshal(item, &s); err == nil {
			specs = append(specs, FileSpec{Path: s})
			continue
		}

		var spec FileSpec
		if err := json.Unmarshal(item, &spec); err != nil {
			return nil, fmt.Errorf("invalid tokensFiles entry: %w", err)
		}
		specs = append(specs, spec)
	}

	return specs, nil
}
