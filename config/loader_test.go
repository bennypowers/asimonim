/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package config

import (
	"testing"

	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/testutil"
)

func TestLoad_SimpleYAML(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/config/simple", "/project")

	cfg, err := Load(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg == nil {
		t.Fatal("expected config, got nil")
	}

	if cfg.Prefix != "rh" {
		t.Errorf("expected prefix 'rh', got %q", cfg.Prefix)
	}

	if len(cfg.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(cfg.Files))
	}

	if cfg.Files[0].Path != "./tokens.json" {
		t.Errorf("expected file path './tokens.json', got %q", cfg.Files[0].Path)
	}

	if len(cfg.GroupMarkers) != 2 {
		t.Fatalf("expected 2 group markers, got %d", len(cfg.GroupMarkers))
	}

	if cfg.GroupMarkers[0] != "_" || cfg.GroupMarkers[1] != "@" {
		t.Errorf("expected group markers ['_', '@'], got %v", cfg.GroupMarkers)
	}

	if cfg.Schema != "draft" {
		t.Errorf("expected schema 'draft', got %q", cfg.Schema)
	}

	if cfg.SchemaVersion() != schema.Draft {
		t.Errorf("expected schema version Draft, got %v", cfg.SchemaVersion())
	}
}

func TestLoad_JSON(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/config/per-file-overrides", "/project")

	cfg, err := Load(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg == nil {
		t.Fatal("expected config, got nil")
	}

	if cfg.Prefix != "global" {
		t.Errorf("expected prefix 'global', got %q", cfg.Prefix)
	}

	if len(cfg.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(cfg.Files))
	}

	// Check first file spec
	if cfg.Files[0].Path != "./tokens/base.json" {
		t.Errorf("expected path './tokens/base.json', got %q", cfg.Files[0].Path)
	}
	if cfg.Files[0].Prefix != "base" {
		t.Errorf("expected prefix 'base', got %q", cfg.Files[0].Prefix)
	}

	// Check second file spec with group markers override
	if cfg.Files[1].Path != "./tokens/theme.json" {
		t.Errorf("expected path './tokens/theme.json', got %q", cfg.Files[1].Path)
	}
	if cfg.Files[1].Prefix != "theme" {
		t.Errorf("expected prefix 'theme', got %q", cfg.Files[1].Prefix)
	}
	if len(cfg.Files[1].GroupMarkers) != 1 || cfg.Files[1].GroupMarkers[0] != "_" {
		t.Errorf("expected groupMarkers ['_'], got %v", cfg.Files[1].GroupMarkers)
	}
}

func TestLoad_NotFound(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/project")

	cfg, err := Load(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg != nil {
		t.Errorf("expected nil config when not found, got %+v", cfg)
	}
}

func TestLoadOrDefault_Found(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/config/simple", "/project")

	cfg := LoadOrDefault(mfs, "/project")
	if cfg.Prefix != "rh" {
		t.Errorf("expected prefix 'rh', got %q", cfg.Prefix)
	}
}

func TestLoadOrDefault_NotFound(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/project")

	cfg := LoadOrDefault(mfs, "/project")
	if cfg == nil {
		t.Fatal("expected default config, got nil")
	}

	if cfg.Prefix != "" {
		t.Errorf("expected empty prefix in default, got %q", cfg.Prefix)
	}
}

func TestConfig_OptionsForFile(t *testing.T) {
	cfg := &Config{
		Prefix:       "global",
		GroupMarkers: []string{"DEFAULT"},
		Schema:       "draft",
		Files: []FileSpec{
			{Path: "/tokens/base.json", Prefix: "base"},
			{Path: "/tokens/theme.json", Prefix: "theme", GroupMarkers: []string{"_", "@"}},
		},
	}

	t.Run("matching file with prefix override", func(t *testing.T) {
		opts := cfg.OptionsForFile("/tokens/base.json")
		if opts.Prefix != "base" {
			t.Errorf("expected prefix 'base', got %q", opts.Prefix)
		}
		if len(opts.GroupMarkers) != 1 || opts.GroupMarkers[0] != "DEFAULT" {
			t.Errorf("expected global group markers, got %v", opts.GroupMarkers)
		}
	})

	t.Run("matching file with groupMarkers override", func(t *testing.T) {
		opts := cfg.OptionsForFile("/tokens/theme.json")
		if opts.Prefix != "theme" {
			t.Errorf("expected prefix 'theme', got %q", opts.Prefix)
		}
		if len(opts.GroupMarkers) != 2 {
			t.Fatalf("expected 2 group markers, got %d", len(opts.GroupMarkers))
		}
		if opts.GroupMarkers[0] != "_" || opts.GroupMarkers[1] != "@" {
			t.Errorf("expected group markers ['_', '@'], got %v", opts.GroupMarkers)
		}
	})

	t.Run("non-matching file uses global config", func(t *testing.T) {
		opts := cfg.OptionsForFile("/other/file.json")
		if opts.Prefix != "global" {
			t.Errorf("expected prefix 'global', got %q", opts.Prefix)
		}
		if len(opts.GroupMarkers) != 1 || opts.GroupMarkers[0] != "DEFAULT" {
			t.Errorf("expected global group markers, got %v", opts.GroupMarkers)
		}
	})

	t.Run("schema version is set", func(t *testing.T) {
		opts := cfg.OptionsForFile("/any/file.json")
		if opts.SchemaVersion != schema.Draft {
			t.Errorf("expected schema version Draft, got %v", opts.SchemaVersion)
		}
	})
}

func TestConfig_FilePaths(t *testing.T) {
	cfg := &Config{
		Files: []FileSpec{
			{Path: "./tokens.json"},
			{Path: "npm:@rhds/tokens/json/rhds.tokens.json"},
			{Path: "./other/*.yaml"},
		},
	}

	paths := cfg.FilePaths()
	if len(paths) != 3 {
		t.Fatalf("expected 3 paths, got %d", len(paths))
	}

	expected := []string{
		"./tokens.json",
		"npm:@rhds/tokens/json/rhds.tokens.json",
		"./other/*.yaml",
	}

	for i, path := range paths {
		if path != expected[i] {
			t.Errorf("paths[%d]: expected %q, got %q", i, expected[i], path)
		}
	}
}

func TestFileSpec_UnmarshalYAML_String(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/config/simple", "/project")

	cfg, err := Load(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// simple config has files as strings
	if cfg.Files[0].Path != "./tokens.json" {
		t.Errorf("expected path './tokens.json', got %q", cfg.Files[0].Path)
	}
}

func TestFileSpec_UnmarshalJSON_Object(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/config/per-file-overrides", "/project")

	cfg, err := Load(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// per-file-overrides has files as objects
	if cfg.Files[0].Prefix != "base" {
		t.Errorf("expected prefix 'base', got %q", cfg.Files[0].Prefix)
	}
}

func TestConfig_SchemaVersion_Invalid(t *testing.T) {
	cfg := &Config{Schema: "invalid"}
	if cfg.SchemaVersion() != schema.Unknown {
		t.Errorf("expected Unknown for invalid schema, got %v", cfg.SchemaVersion())
	}
}

func TestConfig_SchemaVersion_Empty(t *testing.T) {
	cfg := &Config{}
	if cfg.SchemaVersion() != schema.Unknown {
		t.Errorf("expected Unknown for empty schema, got %v", cfg.SchemaVersion())
	}
}
