/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	asimfs "bennypowers.dev/asimonim/fs"
	"bennypowers.dev/asimonim/schema"
)

// configKeys are the package.json keys to check, in priority order.
var configKeys = []string{"asimonim", "designTokensLanguageServer", "design-tokens-language-server"}

// LoadFromPackageJSON reads design tokens configuration from package.json.
// Checks "asimonim", "designTokensLanguageServer", and "design-tokens-language-server"
// keys in priority order. It reads only prefix, cdn, schema, groupMarkers,
// resolvers, and tokensFiles/files; other Config fields such as header and
// outputs are ignored when loading from package.json. Returns nil if no
// config found (not an error).
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

	data = schema.NormalizeJSON(data)

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

func parseStringField(m map[string]json.RawMessage, key string) (string, error) {
	raw, ok := m[key]
	if !ok {
		return "", nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", fmt.Errorf("invalid type for %s: expected string", key)
	}
	return s, nil
}

func buildConfigFromMap(m map[string]json.RawMessage) (*Config, error) {
	cfg := &Config{}

	var err error
	if cfg.Prefix, err = parseStringField(m, "prefix"); err != nil {
		return nil, err
	}
	if cfg.CDN, err = parseStringField(m, "cdn"); err != nil {
		return nil, err
	}
	if cfg.Schema, err = parseStringField(m, "schema"); err != nil {
		return nil, err
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
		if single == "" {
			return nil, fmt.Errorf("tokensFiles entry missing path")
		}
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
			if s == "" {
				return nil, fmt.Errorf("tokensFiles entry missing path")
			}
			specs = append(specs, FileSpec{Path: s})
			continue
		}

		var spec FileSpec
		if err := json.Unmarshal(item, &spec); err != nil {
			return nil, fmt.Errorf("invalid tokensFiles entry: %w", err)
		}
		if spec.Path == "" {
			return nil, fmt.Errorf("tokensFiles entry missing path")
		}
		specs = append(specs, spec)
	}

	return specs, nil
}
