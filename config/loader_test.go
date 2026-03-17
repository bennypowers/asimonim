/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package config

import (
	"testing"

	"bennypowers.dev/asimonim/internal/mapfs"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/specifier"
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

func TestFileSpec_UnmarshalJSON_StringValue(t *testing.T) {
	var spec FileSpec
	err := spec.UnmarshalJSON([]byte(`"./tokens.json"`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.Path != "./tokens.json" {
		t.Errorf("expected path './tokens.json', got %q", spec.Path)
	}
	if spec.Prefix != "" {
		t.Errorf("expected empty prefix, got %q", spec.Prefix)
	}
}

func TestFileSpec_UnmarshalJSON_ObjectValue(t *testing.T) {
	var spec FileSpec
	err := spec.UnmarshalJSON([]byte(`{"path":"./tokens.json","prefix":"rh","groupMarkers":["_"]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.Path != "./tokens.json" {
		t.Errorf("expected path './tokens.json', got %q", spec.Path)
	}
	if spec.Prefix != "rh" {
		t.Errorf("expected prefix 'rh', got %q", spec.Prefix)
	}
	if len(spec.GroupMarkers) != 1 || spec.GroupMarkers[0] != "_" {
		t.Errorf("expected groupMarkers ['_'], got %v", spec.GroupMarkers)
	}
}

func TestFileSpec_UnmarshalJSON_InvalidJSON(t *testing.T) {
	var spec FileSpec
	err := spec.UnmarshalJSON([]byte(`{not valid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestFileSpec_UnmarshalJSON_NullValue(t *testing.T) {
	var spec FileSpec
	err := spec.UnmarshalJSON([]byte(`null`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// null should result in zero-value FileSpec
	if spec.Path != "" {
		t.Errorf("expected empty path, got %q", spec.Path)
	}
}

func TestFileSpec_UnmarshalJSON_NumberValue(t *testing.T) {
	var spec FileSpec
	// A number is not a valid string or object, so it should fail
	err := spec.UnmarshalJSON([]byte(`42`))
	if err == nil {
		t.Fatal("expected error for numeric JSON value")
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

func TestExpandFiles_GlobPattern(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/config/with-globs", "/project")

	cfg, err := Load(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expanded, err := cfg.ExpandFiles(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error expanding files: %v", err)
	}

	if len(expanded) != 2 {
		t.Fatalf("expected 2 expanded files, got %d: %v", len(expanded), expanded)
	}

	// Both YAML files from the tokens/ dir should be found
	found := map[string]bool{}
	for _, p := range expanded {
		found[p] = true
	}
	if !found["/project/tokens/colors.yaml"] {
		t.Errorf("expected /project/tokens/colors.yaml in expanded files, got %v", expanded)
	}
	if !found["/project/tokens/spacing.yaml"] {
		t.Errorf("expected /project/tokens/spacing.yaml in expanded files, got %v", expanded)
	}
}

func TestExpandFiles_NpmPassthrough(t *testing.T) {
	cfg := &Config{
		Files: []FileSpec{
			{Path: "npm:@acme/tokens/tokens.json"},
		},
	}

	mfs := mapfs.New()
	expanded, err := cfg.ExpandFiles(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(expanded) != 1 {
		t.Fatalf("expected 1 path, got %d", len(expanded))
	}
	if expanded[0] != "npm:@acme/tokens/tokens.json" {
		t.Errorf("expected npm: path passthrough, got %q", expanded[0])
	}
}

func TestExpandFiles_RelativePath(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/tokens.json", `{}`, 0644)

	cfg := &Config{
		Files: []FileSpec{
			{Path: "tokens.json"},
		},
	}

	expanded, err := cfg.ExpandFiles(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(expanded) != 1 {
		t.Fatalf("expected 1 path, got %d", len(expanded))
	}
	// Relative paths should be made absolute relative to rootDir
	if expanded[0] != "/project/tokens.json" {
		t.Errorf("expected /project/tokens.json, got %q", expanded[0])
	}
}

func TestExpandFiles_AbsolutePath(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/absolute/tokens.json", `{}`, 0644)

	cfg := &Config{
		Files: []FileSpec{
			{Path: "/absolute/tokens.json"},
		},
	}

	expanded, err := cfg.ExpandFiles(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(expanded) != 1 {
		t.Fatalf("expected 1 path, got %d", len(expanded))
	}
	if expanded[0] != "/absolute/tokens.json" {
		t.Errorf("expected /absolute/tokens.json, got %q", expanded[0])
	}
}

func TestExpandFiles_EmptyFiles(t *testing.T) {
	cfg := &Config{}

	mfs := mapfs.New()
	expanded, err := cfg.ExpandFiles(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(expanded) != 0 {
		t.Errorf("expected 0 expanded files, got %d", len(expanded))
	}
}

func TestExpandGlob_NoMatches(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/tokens.json", `{}`, 0644)

	cfg := &Config{
		Files: []FileSpec{
			{Path: "/project/*.yaml"},
		},
	}

	expanded, err := cfg.ExpandFiles(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No YAML files exist, so glob should return empty
	if len(expanded) != 0 {
		t.Errorf("expected 0 expanded files, got %d: %v", len(expanded), expanded)
	}
}

func TestExpandGlob_DoubleStarPattern(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/tokens/a.json", `{}`, 0644)
	mfs.AddFile("/project/tokens/sub/b.json", `{}`, 0644)
	mfs.AddFile("/project/tokens/sub/deep/c.json", `{}`, 0644)

	cfg := &Config{
		Files: []FileSpec{
			{Path: "/project/tokens/**/*.json"},
		},
	}

	expanded, err := cfg.ExpandFiles(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(expanded) != 3 {
		t.Fatalf("expected 3 expanded files, got %d: %v", len(expanded), expanded)
	}
}

func TestResolveFiles_WithMockResolver(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/tokens.json", `{}`, 0644)

	cfg := &Config{
		Files: []FileSpec{
			{Path: "tokens.json"},
			{Path: "npm:@acme/tokens/tokens.json"},
		},
	}

	resolver := &mockResolver{
		resolveFunc: func(spec string) (*specifier.ResolvedFile, error) {
			if spec == "npm:@acme/tokens/tokens.json" {
				return &specifier.ResolvedFile{
					Specifier: spec,
					Path:      "/project/node_modules/@acme/tokens/tokens.json",
					Kind:      specifier.KindNPM,
				}, nil
			}
			return &specifier.ResolvedFile{
				Specifier: spec,
				Path:      spec,
				Kind:      specifier.KindLocal,
			}, nil
		},
	}

	results, err := cfg.ResolveFiles(resolver, mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 resolved files, got %d", len(results))
	}

	// First should be local
	if results[0].Kind != specifier.KindLocal {
		t.Errorf("expected first result to be KindLocal, got %v", results[0].Kind)
	}

	// Second should be npm
	if results[1].Kind != specifier.KindNPM {
		t.Errorf("expected second result to be KindNPM, got %v", results[1].Kind)
	}
	if results[1].Path != "/project/node_modules/@acme/tokens/tokens.json" {
		t.Errorf("expected npm resolved path, got %q", results[1].Path)
	}
}

func TestResolveResolvers_WithMockResolver(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/resolver.json", `{}`, 0644)

	cfg := &Config{
		Resolvers: []string{
			"resolver.json",
			"npm:@acme/tokens/tokens.resolver.json",
		},
	}

	resolver := &mockResolver{
		resolveFunc: func(spec string) (*specifier.ResolvedFile, error) {
			if spec == "npm:@acme/tokens/tokens.resolver.json" {
				return &specifier.ResolvedFile{
					Specifier: spec,
					Path:      "/project/node_modules/@acme/tokens/tokens.resolver.json",
					Kind:      specifier.KindNPM,
				}, nil
			}
			return &specifier.ResolvedFile{
				Specifier: spec,
				Path:      spec,
				Kind:      specifier.KindLocal,
			}, nil
		},
	}

	results, err := cfg.ResolveResolvers(resolver, mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 resolved resolvers, got %d", len(results))
	}
}

// mockResolver implements specifier.Resolver for tests.
type mockResolver struct {
	resolveFunc func(spec string) (*specifier.ResolvedFile, error)
}

func (m *mockResolver) Resolve(spec string) (*specifier.ResolvedFile, error) {
	return m.resolveFunc(spec)
}

func (m *mockResolver) CanResolve(string) bool {
	return true
}

func TestLoad_WithResolvers(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/config/with-resolvers", "/project")

	cfg, err := Load(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg == nil {
		t.Fatal("expected config, got nil")
	}

	if len(cfg.Resolvers) != 2 {
		t.Fatalf("expected 2 resolvers, got %d", len(cfg.Resolvers))
	}

	if cfg.Resolvers[0] != "./tokens.resolver.json" {
		t.Errorf("expected resolver[0] './tokens.resolver.json', got %q", cfg.Resolvers[0])
	}

	if cfg.Resolvers[1] != "npm:@acme/tokens/tokens.resolver.json" {
		t.Errorf("expected resolver[1] 'npm:@acme/tokens/tokens.resolver.json', got %q", cfg.Resolvers[1])
	}

	// Files should still work alongside resolvers
	if len(cfg.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(cfg.Files))
	}
	if cfg.Files[0].Path != "./overrides.json" {
		t.Errorf("expected file path './overrides.json', got %q", cfg.Files[0].Path)
	}
}
