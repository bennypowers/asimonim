/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package specifier

import (
	"fmt"
	"path/filepath"
	"strings"

	asimfs "bennypowers.dev/asimonim/fs"
)

// NPMResolver resolves npm: specifiers to node_modules paths.
type NPMResolver struct {
	fs      asimfs.FileSystem
	rootDir string
}

// NewNPMResolver creates a resolver for npm: package specifiers.
// The rootDir is the starting directory for node_modules lookup.
func NewNPMResolver(fs asimfs.FileSystem, rootDir string) *NPMResolver {
	return &NPMResolver{
		fs:      fs,
		rootDir: rootDir,
	}
}

// Resolve resolves an npm: specifier to a filesystem path.
// It walks up the directory tree looking for node_modules.
func (r *NPMResolver) Resolve(spec string) (*ResolvedFile, error) {
	parsed := Parse(spec)
	if parsed.Kind != KindNPM {
		return nil, fmt.Errorf("not an npm specifier: %s", spec)
	}

	// Convert rootDir to absolute path for proper walk-up
	dir := r.rootDir
	if !filepath.IsAbs(dir) {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path %s: %w", dir, err)
		}
		dir = absDir
	}

	startDir := dir

	// Walk up directory tree looking for node_modules
	for {
		nodeModulesPath := filepath.Join(dir, "node_modules", parsed.Package, parsed.File)
		if r.fs.Exists(nodeModulesPath) {
			return &ResolvedFile{
				Specifier: spec,
				Path:      nodeModulesPath,
				Kind:      KindNPM,
			}, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	return nil, fmt.Errorf("package not found: %s (looked in node_modules starting from %s)", parsed.Package, startDir)
}

// CanResolve returns true for npm: specifiers.
func (r *NPMResolver) CanResolve(spec string) bool {
	return strings.HasPrefix(spec, "npm:")
}
