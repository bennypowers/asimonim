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

// JSRNodeModulesResolver resolves jsr: specifiers via the npm compatibility layer.
// Packages must be installed via `npx jsr add @scope/pkg`.
//
// JSR packages installed via the npm compatibility layer appear in node_modules
// under the @jsr scope with the following naming convention:
//   - jsr:@scope/pkg → @jsr/scope__pkg
//   - jsr:pkg → @jsr/pkg
type JSRNodeModulesResolver struct {
	fs      asimfs.FileSystem
	rootDir string
}

// NewJSRNodeModulesResolver creates a resolver for jsr: package specifiers
// that looks in node_modules/@jsr/.
func NewJSRNodeModulesResolver(fs asimfs.FileSystem, rootDir string) *JSRNodeModulesResolver {
	return &JSRNodeModulesResolver{
		fs:      fs,
		rootDir: rootDir,
	}
}

// Resolve resolves a jsr: specifier to a filesystem path.
// It translates jsr:@scope/pkg/file to node_modules/@jsr/scope__pkg/file
// and walks up the directory tree looking for node_modules.
func (r *JSRNodeModulesResolver) Resolve(spec string) (*ResolvedFile, error) {
	parsed := Parse(spec)
	if parsed.Kind != KindJSR {
		return nil, fmt.Errorf("not a jsr specifier: %s", spec)
	}

	// Translate the package name to the npm compatibility format
	npmPackageName := jsrToNPMCompatPackage(parsed.Package)

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
		nodeModulesPath := filepath.Join(dir, "node_modules", "@jsr", npmPackageName, parsed.File)
		if r.fs.Exists(nodeModulesPath) {
			return &ResolvedFile{
				Specifier: spec,
				Path:      nodeModulesPath,
				Kind:      KindJSR,
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

	return nil, fmt.Errorf("jsr package not found: %s (looked in node_modules/@jsr starting from %s)", parsed.Package, startDir)
}

// CanResolve returns true for jsr: specifiers.
func (r *JSRNodeModulesResolver) CanResolve(spec string) bool {
	return strings.HasPrefix(spec, "jsr:")
}

// jsrToNPMCompatPackage converts a JSR package name to its npm compatibility layer name.
// For scoped packages (@scope/pkg), it becomes scope__pkg.
// For unscoped packages (pkg), it stays as pkg.
func jsrToNPMCompatPackage(pkg string) string {
	if scopedPkg, ok := strings.CutPrefix(pkg, "@"); ok {
		// @scope/pkg → scope__pkg
		// Remove the leading @ and replace / with __
		return strings.Replace(scopedPkg, "/", "__", 1)
	}
	return pkg
}
