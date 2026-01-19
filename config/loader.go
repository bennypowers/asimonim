/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package config

import (
	"encoding/json"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v3"

	asimfs "bennypowers.dev/asimonim/fs"
	"bennypowers.dev/asimonim/specifier"
)

// ConfigFileName is the base name of the config file without extension.
const ConfigFileName = "design-tokens"

// ConfigDir is the directory where config files are stored.
const ConfigDir = ".config"

// configExtensions are the supported config file extensions in priority order.
var configExtensions = []string{".yaml", ".yml", ".json"}

// Load searches for .config/design-tokens.{yaml,yml,json} from rootDir.
// Returns nil if no config found (not an error).
func Load(filesystem asimfs.FileSystem, rootDir string) (*Config, error) {
	for _, ext := range configExtensions {
		configPath := filepath.Join(rootDir, ConfigDir, ConfigFileName+ext)
		if !filesystem.Exists(configPath) {
			continue
		}

		data, err := filesystem.ReadFile(configPath)
		if err != nil {
			return nil, err
		}

		cfg := &Config{}
		switch ext {
		case ".yaml", ".yml":
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, err
			}
		case ".json":
			if err := json.Unmarshal(data, cfg); err != nil {
				return nil, err
			}
		}

		return cfg, nil
	}

	return nil, nil
}

// LoadOrDefault returns config or defaults if not found.
func LoadOrDefault(filesystem asimfs.FileSystem, rootDir string) *Config {
	cfg, err := Load(filesystem, rootDir)
	if err != nil || cfg == nil {
		return Default()
	}
	return cfg
}

// ExpandFiles expands glob patterns in Files and returns absolute paths.
// Paths starting with npm: are passed through unchanged.
func (c *Config) ExpandFiles(filesystem asimfs.FileSystem, rootDir string) ([]string, error) {
	var result []string

	for _, spec := range c.Files {
		expanded, err := expandFilePath(filesystem, rootDir, spec.Path)
		if err != nil {
			return nil, err
		}
		result = append(result, expanded...)
	}

	return result, nil
}

// ResolveFiles expands glob patterns and resolves package specifiers to filesystem paths.
// Returns ResolvedFile entries that preserve both the original specifier and resolved path.
func (c *Config) ResolveFiles(resolver specifier.Resolver, filesystem asimfs.FileSystem, rootDir string) ([]*specifier.ResolvedFile, error) {
	var result []*specifier.ResolvedFile

	for _, spec := range c.Files {
		expanded, err := expandFilePath(filesystem, rootDir, spec.Path)
		if err != nil {
			return nil, err
		}

		for _, path := range expanded {
			resolved, err := resolver.Resolve(path)
			if err != nil {
				return nil, err
			}
			result = append(result, resolved)
		}
	}

	return result, nil
}

// expandFilePath expands a single file path which may contain globs.
// npm: paths are passed through unchanged.
func expandFilePath(filesystem asimfs.FileSystem, rootDir, pattern string) ([]string, error) {
	// npm: protocol paths are passed through unchanged
	if strings.HasPrefix(pattern, "npm:") {
		return []string{pattern}, nil
	}

	// Make pattern absolute if relative
	if !filepath.IsAbs(pattern) {
		pattern = filepath.Join(rootDir, pattern)
	}

	// Check if pattern contains glob characters
	if !containsGlob(pattern) {
		// Not a glob, return the path directly (errors handled when file is read)
		return []string{pattern}, nil
	}

	// Expand glob pattern using fs.WalkDir
	return expandGlob(filesystem, pattern)
}

// containsGlob returns true if the pattern contains glob characters.
func containsGlob(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

// expandGlob expands a glob pattern against the filesystem.
func expandGlob(filesystem asimfs.FileSystem, pattern string) ([]string, error) {
	// Find the base directory (non-glob prefix)
	baseDir := pattern
	for containsGlob(baseDir) {
		baseDir = filepath.Dir(baseDir)
	}

	// Get the relative pattern from baseDir
	relPattern := strings.TrimPrefix(pattern, baseDir)
	relPattern = strings.TrimPrefix(relPattern, string(filepath.Separator))

	var matches []string

	err := fs.WalkDir(filesystem, baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip directories we can't read
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		// Get path relative to baseDir for matching
		relPath := strings.TrimPrefix(path, baseDir)
		relPath = strings.TrimPrefix(relPath, string(filepath.Separator))

		// Match against the pattern (doublestar handles both simple and ** globs)
		if matchDoublestar(relPattern, relPath) {
			matches = append(matches, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return matches, nil
}

// matchDoublestar provides ** glob matching using the doublestar library.
// Supports complex patterns like packages/**/tokens/**/data.json
func matchDoublestar(pattern, path string) bool {
	matched, _ := doublestar.Match(pattern, path)
	return matched
}
