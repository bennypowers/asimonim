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

// findTestdata locates a path under testdata/ by walking up parent directories.
func findTestdata(t *testing.T, relPath string) string {
	t.Helper()
	for i := range 5 {
		prefix := strings.Repeat("../", i)
		candidate := filepath.Join(prefix+"testdata", relPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	t.Fatalf("Could not find testdata/%s (tried up to 4 parent dirs)", relPath)
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
		if strings.Join(tok.Path, ".") == dotPath {
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
func findTestdataOrCreate(t *testing.T, relPath string) string {
	t.Helper()
	for i := range 5 {
		prefix := strings.Repeat("../", i)
		candidate := filepath.Join(prefix+"testdata", relPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	// Fallback: create under the nearest testdata/ that exists
	for i := range 5 {
		prefix := strings.Repeat("../", i)
		candidate := prefix + "testdata"
		if _, err := os.Stat(candidate); err == nil {
			target := filepath.Join(candidate, relPath)
			if err := os.MkdirAll(target, 0755); err != nil {
				t.Fatalf("Failed to create %s: %v", target, err)
			}
			return target
		}
	}
	t.Fatalf("Could not find testdata/ directory for golden file %s", relPath)
	return ""
}
