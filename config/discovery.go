/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package config

import (
	"encoding/json"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	asimfs "bennypowers.dev/asimonim/fs"
	"bennypowers.dev/asimonim/specifier"
)

// packageJSON represents the fields we care about from a package.json.
type packageJSON struct {
	Name            string           `json:"name"`
	Dependencies    map[string]string `json:"dependencies"`
	Exports         json.RawMessage  `json:"exports"`
	DesignTokens    *designTokensMeta `json:"designTokens"`
}

// designTokensMeta represents the "designTokens" field in package.json.
type designTokensMeta struct {
	Resolver string `json:"resolver"`
}

// DiscoverResolvers scans direct dependencies for DTCG resolver files.
// It checks each dependency's package.json for:
//   - A "designTokens" field with a "resolver" path
//   - A "designTokens" export condition in the "exports" field
//
// Returns resolved file entries for all discovered resolvers.
func DiscoverResolvers(filesystem asimfs.FileSystem, rootDir string) ([]*specifier.ResolvedFile, error) {
	// Read the project's package.json to find direct dependencies
	projectPkg, err := readPackageJSON(filesystem, filepath.Join(rootDir, "package.json"))
	if err != nil {
		return nil, err
	}
	if projectPkg == nil {
		// No package.json found — discovery is a no-op
		return nil, nil
	}

	depNames := slices.Sorted(maps.Keys(projectPkg.Dependencies))

	var result []*specifier.ResolvedFile

	for _, depName := range depNames {
		resolved, err := discoverResolverInDep(filesystem, rootDir, depName)
		if err != nil {
			continue
		}
		if resolved != nil {
			result = append(result, resolved)
		}
	}

	return result, nil
}

// discoverResolverInDep checks a single dependency for a resolver file.
func discoverResolverInDep(filesystem asimfs.FileSystem, rootDir, depName string) (*specifier.ResolvedFile, error) {
	depDir := findDepDir(filesystem, rootDir, depName)
	if depDir == "" {
		return nil, nil
	}

	depPkgPath := filepath.Join(depDir, "package.json")
	depPkg, err := readPackageJSON(filesystem, depPkgPath)
	if err != nil || depPkg == nil {
		return nil, nil
	}

	// Check "designTokens" field first (higher priority — explicit)
	if depPkg.DesignTokens != nil && depPkg.DesignTokens.Resolver != "" {
		resolverPath, err := safeDependencyPath(depDir, depPkg.DesignTokens.Resolver)
		if err != nil {
			// Explicit resolver is invalid — don't fall through to exports
			return nil, nil
		}
		if filesystem.Exists(resolverPath) {
			return &specifier.ResolvedFile{
				Specifier: "npm:" + depName + "/" + depPkg.DesignTokens.Resolver,
				Path:      resolverPath,
				Kind:      specifier.KindNPM,
			}, nil
		}
		// Explicit resolver declared but file missing — don't fall through
		return nil, nil
	}

	// Check "designTokens" export condition (fallback when no explicit resolver)
	resolverFile := resolveExportCondition(depPkg.Exports, "designTokens")
	if resolverFile != "" {
		resolverPath, err := safeDependencyPath(depDir, resolverFile)
		if err != nil {
			return nil, nil
		}
		if filesystem.Exists(resolverPath) {
			return &specifier.ResolvedFile{
				Specifier: "npm:" + depName + "/" + resolverFile,
				Path:      resolverPath,
				Kind:      specifier.KindNPM,
			}, nil
		}
	}

	return nil, nil
}

// safeDependencyPath resolves a relative path within a dependency directory,
// rejecting absolute paths and paths that escape the dependency root.
func safeDependencyPath(depDir, raw string) (string, error) {
	if filepath.IsAbs(raw) {
		return "", fmt.Errorf("absolute path not allowed: %s", raw)
	}
	joined := filepath.Join(depDir, raw)
	cleaned := filepath.Clean(joined)
	rel, err := filepath.Rel(depDir, cleaned)
	if err != nil {
		return "", fmt.Errorf("invalid path: %s", raw)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes dependency directory: %s", raw)
	}
	return cleaned, nil
}

// findDepDir locates a dependency's directory by walking up from rootDir.
func findDepDir(filesystem asimfs.FileSystem, rootDir, depName string) string {
	dir := rootDir
	for {
		candidate := filepath.Join(dir, "node_modules", depName)
		if filesystem.Exists(filepath.Join(candidate, "package.json")) {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// readPackageJSON reads and parses a package.json file.
func readPackageJSON(filesystem asimfs.FileSystem, path string) (*packageJSON, error) {
	if !filesystem.Exists(path) {
		return nil, nil
	}
	data, err := filesystem.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}

// resolveExportCondition extracts a file path from a package.json "exports" field
// for the given condition name.
//
// Supports these export shapes:
//
//	{ ".": { "designTokens": "./tokens.resolver.json" } }
//	{ "designTokens": "./tokens.resolver.json" }
func resolveExportCondition(exports json.RawMessage, condition string) string {
	if len(exports) == 0 {
		return ""
	}

	// Try as condition map: { "designTokens": "./path" }
	var condMap map[string]json.RawMessage
	if err := json.Unmarshal(exports, &condMap); err != nil {
		return ""
	}

	// Check top-level condition: { "designTokens": "./path" }
	if raw, ok := condMap[condition]; ok {
		return unquoteExportValue(raw)
	}

	// Check subpath ".": { "designTokens": "./path" }
	if dotRaw, ok := condMap["."]; ok {
		var dotMap map[string]json.RawMessage
		if err := json.Unmarshal(dotRaw, &dotMap); err == nil {
			if raw, ok := dotMap[condition]; ok {
				return unquoteExportValue(raw)
			}
		}
	}

	return ""
}

// unquoteExportValue extracts a string value from a JSON-encoded value.
func unquoteExportValue(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	// Normalize: strip leading "./"
	return strings.TrimPrefix(s, "./")
}
