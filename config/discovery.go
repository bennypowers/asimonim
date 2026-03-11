/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package config

import (
	"encoding/json"
	"path/filepath"
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
	if err != nil || projectPkg == nil {
		return nil, nil
	}

	var result []*specifier.ResolvedFile

	for depName := range projectPkg.Dependencies {
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
		resolverPath := filepath.Join(depDir, depPkg.DesignTokens.Resolver)
		if filesystem.Exists(resolverPath) {
			return &specifier.ResolvedFile{
				Specifier: "npm:" + depName + "/" + depPkg.DesignTokens.Resolver,
				Path:      resolverPath,
				Kind:      specifier.KindNPM,
			}, nil
		}
	}

	// Check "designTokens" export condition
	resolverFile := resolveExportCondition(depPkg.Exports, "designTokens")
	if resolverFile != "" {
		resolverPath := filepath.Join(depDir, resolverFile)
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
