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
	"testing"

	"bennypowers.dev/asimonim/internal/mapfs"
)

// updateGolden enables updating golden files with actual output when -update flag is set.
var updateGolden = flag.Bool("update", false, "update golden files with actual output")

// NewFixtureFS loads fixture files from testdata and returns a MapFileSystem
// with files mapped to the specified root path.
func NewFixtureFS(t *testing.T, fixtureDir string, rootPath string) *mapfs.MapFileSystem {
	t.Helper()

	mfs := mapfs.New()

	// Try multiple possible paths since Go test changes working directory
	possiblePaths := []string{
		filepath.Join("testdata", fixtureDir),
		filepath.Join("..", "testdata", fixtureDir),
		filepath.Join("..", "..", "testdata", fixtureDir),
	}

	var fixturePath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			fixturePath = path
			break
		}
	}
	if fixturePath == "" {
		t.Fatalf("Could not find fixtures at %s (tried all paths)", fixtureDir)
	}

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

	possiblePaths := []string{
		filepath.Join("testdata", fixturePath),
		filepath.Join("..", "testdata", fixturePath),
		filepath.Join("..", "..", "testdata", fixturePath),
	}

	for _, path := range possiblePaths {
		content, err := os.ReadFile(path)
		if err == nil {
			return content
		}
	}
	t.Fatalf("Failed to read fixture %s (tried all paths)", fixturePath)
	return nil
}

// UpdateGoldenFile writes actual output to the golden file when -update flag is set.
func UpdateGoldenFile(t *testing.T, goldenPath string, actual []byte) {
	t.Helper()
	if !*updateGolden {
		return
	}

	possiblePaths := []string{
		filepath.Join("testdata", goldenPath),
		filepath.Join("..", "testdata", goldenPath),
		filepath.Join("..", "..", "testdata", goldenPath),
	}

	var targetPath string
	for _, path := range possiblePaths {
		parentDir := filepath.Dir(path)
		if _, err := os.Stat(parentDir); err == nil {
			targetPath = path
			break
		}
	}
	if targetPath == "" {
		targetPath = possiblePaths[0]
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		t.Fatalf("Failed to create directory for golden file %s: %v", goldenPath, err)
	}

	if err := os.WriteFile(targetPath, actual, 0644); err != nil {
		t.Fatalf("Failed to write golden file %s: %v", goldenPath, err)
	}

	t.Logf("Updated golden file: %s", targetPath)
}
