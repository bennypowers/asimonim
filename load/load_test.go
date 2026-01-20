/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package load_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"bennypowers.dev/asimonim/load"
	"bennypowers.dev/asimonim/schema"
)

func testdataDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata")
}

func TestLoad_SimpleFile(t *testing.T) {
	root := testdataDir()
	tokenMap, err := load.Load("simple.json", load.Options{
		Root: root,
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if tokenMap.Len() != 2 {
		t.Errorf("expected 2 tokens, got %d", tokenMap.Len())
	}

	// Check primary token
	primary, ok := tokenMap.Get("color-primary")
	if !ok {
		t.Fatal("expected to find color-primary")
	}
	if primary.Value != "#FF6B35" {
		t.Errorf("primary.Value = %q, want %q", primary.Value, "#FF6B35")
	}

	// Check secondary token (alias resolution)
	secondary, ok := tokenMap.Get("color-secondary")
	if !ok {
		t.Fatal("expected to find color-secondary")
	}
	if !secondary.IsResolved {
		t.Error("expected secondary to be resolved")
	}
}

func TestLoad_WithPrefix(t *testing.T) {
	root := testdataDir()
	tokenMap, err := load.Load("simple.json", load.Options{
		Root:   root,
		Prefix: "rh",
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should find by short name
	tok, ok := tokenMap.Get("color-primary")
	if !ok {
		t.Fatal("expected to find token by short name")
	}
	if tok.Prefix != "rh" {
		t.Errorf("tok.Prefix = %q, want %q", tok.Prefix, "rh")
	}

	// Should also find by full CSS name
	tok2, ok := tokenMap.Get("--rh-color-primary")
	if !ok {
		t.Fatal("expected to find token by full CSS name")
	}
	if tok2.Value != "#FF6B35" {
		t.Errorf("tok2.Value = %q, want %q", tok2.Value, "#FF6B35")
	}
}

func TestLoad_WithSchemaVersion(t *testing.T) {
	root := testdataDir()
	tokenMap, err := load.Load("simple.json", load.Options{
		Root:          root,
		SchemaVersion: schema.Draft,
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	tok, ok := tokenMap.Get("color-primary")
	if !ok {
		t.Fatal("expected to find token")
	}
	if tok.SchemaVersion != schema.Draft {
		t.Errorf("tok.SchemaVersion = %v, want %v", tok.SchemaVersion, schema.Draft)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	root := testdataDir()
	_, err := load.Load("nonexistent.json", load.Options{
		Root: root,
	})
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	root := testdataDir()

	// Create an invalid JSON file for this test
	_, err := load.Load("../load_test.go", load.Options{
		Root: root,
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
