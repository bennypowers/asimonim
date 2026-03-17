/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package testutil provides testing utilities for asimonim.
package testutil

import (
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bennypowers.dev/asimonim/internal/mapfs"
	"bennypowers.dev/asimonim/parser"
	"bennypowers.dev/asimonim/resolver"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/token"
)

// updateGolden enables updating golden files with actual output when -update flag is set.
var updateGolden = flag.Bool("update", false, "update golden files with actual output")

// NewFixtureFS loads fixture files from testdata and returns a MapFileSystem
// with files mapped to the specified root path.
func NewFixtureFS(t *testing.T, fixtureDir string, rootPath string) *mapfs.MapFileSystem {
	t.Helper()

	mfs := mapfs.New()

	// Try multiple possible paths since Go test changes working directory
	fixturePath := findTestdata(t, fixtureDir)

	// Walk fixture directory and load all files
	err := filepath.WalkDir(fixturePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(fixturePath, path)
		if err != nil {
			return err
		}

		virtualPath := filepath.Join(rootPath, relPath)
		mfs.AddFile(virtualPath, string(content), 0644)

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to load fixtures from %s: %v", fixtureDir, err)
	}

	return mfs
}

// LoadFixtureFile reads a single fixture file and returns its content.
func LoadFixtureFile(t *testing.T, fixturePath string) []byte {
	t.Helper()

	fullPath := findTestdata(t, fixturePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read fixture %s: %v", fixturePath, err)
	}
	return content
}

// maxParentProbe is the maximum number of parent directories to search for testdata/.
const maxParentProbe = 5

// probeTestdata searches for relPath under testdata/ by walking up parent directories.
// Returns the found path or empty string if not found.
func probeTestdata(relPath string) string {
	for i := range maxParentProbe {
		prefix := strings.Repeat("../", i)
		candidate := filepath.Join(prefix+"testdata", relPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// probeTestdataRoot returns the nearest testdata/ directory by walking up parents.
func probeTestdataRoot() string {
	for i := range maxParentProbe {
		candidate := strings.Repeat("../", i) + "testdata"
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// findTestdata locates a path under testdata/ by walking up parent directories.
func findTestdata(t *testing.T, relPath string) string {
	t.Helper()
	if path := probeTestdata(relPath); path != "" {
		return path
	}
	t.Fatalf("Could not find testdata/%s (tried up to %d parent dirs)", relPath, maxParentProbe-1)
	return ""
}

// ParseFixtureTokens loads a fixture file, parses tokens, and resolves aliases.
// The fixtureDir is relative to testdata/ (e.g., "fixtures/v2025-10-colors").
// Returns the parsed and resolved tokens.
func ParseFixtureTokens(t *testing.T, fixtureDir string, schemaVersion schema.Version) []*token.Token {
	t.Helper()

	mfs := NewFixtureFS(t, fixtureDir, "/test")

	p := parser.NewJSONParser()
	tokens, err := p.ParseFile(mfs, "/test/tokens.json", parser.Options{
		SchemaVersion: schemaVersion,
		SkipPositions: true,
	})
	if err != nil {
		t.Fatalf("failed to parse %s/tokens.json: %v", fixtureDir, err)
	}

	if err := resolver.ResolveAliases(tokens, schemaVersion); err != nil {
		t.Fatalf("failed to resolve aliases in %s: %v", fixtureDir, err)
	}

	return tokens
}

// TokenByPath returns the first token matching the given dot-separated path
// (e.g., "color.oklch"). Fails the test if not found.
func TokenByPath(t *testing.T, tokens []*token.Token, dotPath string) *token.Token {
	t.Helper()
	for _, tok := range tokens {
		if tok.DotPath() == dotPath {
			return tok
		}
	}
	t.Fatalf("token not found: %s", dotPath)
	return nil
}

// UpdateGoldenFile writes actual output to the golden file when -update flag is set.
func UpdateGoldenFile(t *testing.T, goldenPath string, actual []byte) {
	t.Helper()
	if !*updateGolden {
		return
	}

	// Find the testdata root by locating the parent directory
	parentDir := filepath.Dir(goldenPath)
	targetDir := findTestdataOrCreate(t, parentDir)
	targetPath := filepath.Join(targetDir, filepath.Base(goldenPath))

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		t.Fatalf("Failed to create directory for golden file %s: %v", goldenPath, err)
	}

	if err := os.WriteFile(targetPath, actual, 0644); err != nil {
		t.Fatalf("Failed to write golden file %s: %v", goldenPath, err)
	}

	t.Logf("Updated golden file: %s", targetPath)
}

// findTestdataOrCreate locates a path under testdata/, creating it if needed.
// Walks ancestor directories to find the deepest existing subtree before
// falling back to the testdata root.
func findTestdataOrCreate(t *testing.T, relPath string) string {
	t.Helper()
	if path := probeTestdata(relPath); path != "" {
		return path
	}
	// Walk ancestors to find the deepest existing subtree
	for parent := filepath.Dir(relPath); parent != "." && parent != relPath; parent = filepath.Dir(parent) {
		if base := probeTestdata(parent); base != "" {
			suffix, err := filepath.Rel(parent, relPath)
			if err != nil {
				t.Fatalf("Failed to resolve %s relative to %s: %v", relPath, parent, err)
			}
			target := filepath.Join(base, suffix)
			if err := os.MkdirAll(target, 0755); err != nil {
				t.Fatalf("Failed to create %s: %v", target, err)
			}
			return target
		}
	}
	// Fallback: create under the nearest testdata/ root
	if root := probeTestdataRoot(); root != "" {
		target := filepath.Join(root, relPath)
		if err := os.MkdirAll(target, 0755); err != nil {
			t.Fatalf("Failed to create %s: %v", target, err)
		}
		return target
	}
	t.Fatalf("Could not find testdata/ directory for golden file %s", relPath)
	return ""
}
