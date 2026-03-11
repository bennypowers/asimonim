/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package config

import (
	"encoding/json"
	"testing"

	"bennypowers.dev/asimonim/internal/mapfs"
	"bennypowers.dev/asimonim/specifier"
	"bennypowers.dev/asimonim/testutil"
)

func TestDiscoverResolvers_DesignTokensField(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/config/discovery", "/project")

	results, err := DiscoverResolvers(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find @acme/tokens (designTokens field) and @brand/theme (export condition)
	// but not lodash (no design tokens)
	found := make(map[string]*specifier.ResolvedFile)
	for _, r := range results {
		found[r.Specifier] = r
	}

	acme, ok := found["npm:@acme/tokens/tokens.resolver.json"]
	if !ok {
		t.Fatal("expected to discover @acme/tokens resolver via designTokens field")
	}
	expectedPath := "/project/node_modules/@acme/tokens/tokens.resolver.json"
	if acme.Path != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, acme.Path)
	}
	if acme.Kind != specifier.KindNPM {
		t.Errorf("expected KindNPM, got %v", acme.Kind)
	}
}

func TestDiscoverResolvers_ExportCondition(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/config/discovery", "/project")

	results, err := DiscoverResolvers(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := make(map[string]*specifier.ResolvedFile)
	for _, r := range results {
		found[r.Specifier] = r
	}

	brand, ok := found["npm:@brand/theme/theme.resolver.json"]
	if !ok {
		t.Fatal("expected to discover @brand/theme resolver via designTokens export condition")
	}
	expectedPath := "/project/node_modules/@brand/theme/theme.resolver.json"
	if brand.Path != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, brand.Path)
	}
}

func TestDiscoverResolvers_TopLevelExportCondition(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/config/discovery-toplevel-export", "/project")

	results, err := DiscoverResolvers(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Specifier != "npm:@design/system/system.resolver.json" {
		t.Errorf("unexpected specifier: %s", results[0].Specifier)
	}
}

func TestDiscoverResolvers_SkipsNonTokenDeps(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/config/discovery", "/project")

	results, err := DiscoverResolvers(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, r := range results {
		if r.Specifier == "npm:lodash" || r.Path == "project/node_modules/lodash" {
			t.Error("should not discover lodash as a resolver")
		}
	}
}

func TestDiscoverResolvers_NoPackageJSON(t *testing.T) {
	mfs := mapfs.New()

	results, err := DiscoverResolvers(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results != nil {
		t.Errorf("expected nil results, got %v", results)
	}
}

func TestDiscoverResolvers_DesignTokensFieldPriority(t *testing.T) {
	// When a package has both designTokens field and export condition,
	// the designTokens field should be preferred
	mfs := testutil.NewFixtureFS(t, "fixtures/config/discovery-priority", "/project")

	results, err := DiscoverResolvers(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// designTokens field takes priority
	if results[0].Specifier != "npm:@both/pkg/field.resolver.json" {
		t.Errorf("expected designTokens field to take priority, got %s", results[0].Specifier)
	}
}

func TestSafeDependencyPath(t *testing.T) {
	tests := []struct {
		name    string
		depDir  string
		raw     string
		wantErr bool
	}{
		{name: "relative path", depDir: "/pkg", raw: "tokens.resolver.json", wantErr: false},
		{name: "subdirectory", depDir: "/pkg", raw: "dist/resolver.json", wantErr: false},
		{name: "dot-slash", depDir: "/pkg", raw: "./resolver.json", wantErr: false},
		{name: "absolute path", depDir: "/pkg", raw: "/etc/passwd", wantErr: true},
		{name: "escapes with ..", depDir: "/pkg", raw: "../../etc/passwd", wantErr: true},
		{name: "single ..", depDir: "/pkg", raw: "..", wantErr: true},
		{name: "escape via subdir", depDir: "/pkg", raw: "sub/../../etc/passwd", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := safeDependencyPath(tt.depDir, tt.raw)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for path %q", tt.raw)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for path %q: %v", tt.raw, err)
			}
		})
	}
}

func TestDiscoverResolvers_InvalidResolverNoFallthrough(t *testing.T) {
	// When designTokens.resolver is declared but invalid (path traversal),
	// it should NOT fall through to the export condition
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"name": "test",
		"dependencies": { "@evil/pkg": "^1.0.0" }
	}`, 0644)
	mfs.AddFile("/project/node_modules/@evil/pkg/package.json", `{
		"name": "@evil/pkg",
		"designTokens": { "resolver": "../../escape.json" },
		"exports": { ".": { "designTokens": "./legit.resolver.json" } }
	}`, 0644)
	mfs.AddFile("/project/node_modules/@evil/pkg/legit.resolver.json", `{"version":"2025.10"}`, 0644)
	mfs.AddFile("/project/escape.json", `{"version":"2025.10"}`, 0644)

	results, err := DiscoverResolvers(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected no results (invalid explicit resolver should block fallthrough), got %d", len(results))
	}
}

func TestDiscoverResolvers_MissingResolverNoFallthrough(t *testing.T) {
	// When designTokens.resolver is declared but the file doesn't exist,
	// it should NOT fall through to the export condition
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"name": "test",
		"dependencies": { "@missing/pkg": "^1.0.0" }
	}`, 0644)
	mfs.AddFile("/project/node_modules/@missing/pkg/package.json", `{
		"name": "@missing/pkg",
		"designTokens": { "resolver": "gone.resolver.json" },
		"exports": { ".": { "designTokens": "./fallback.resolver.json" } }
	}`, 0644)
	mfs.AddFile("/project/node_modules/@missing/pkg/fallback.resolver.json", `{"version":"2025.10"}`, 0644)

	results, err := DiscoverResolvers(mfs, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected no results (missing explicit resolver should block fallthrough), got %d", len(results))
	}
}

func TestDiscoverResolvers_CorruptPackageJSON(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{not valid json`, 0644)

	results, err := DiscoverResolvers(mfs, "/project")
	if err == nil {
		t.Fatal("expected error for corrupt package.json")
	}
	if results != nil {
		t.Errorf("expected nil results, got %v", results)
	}
}

func TestResolveExportCondition(t *testing.T) {
	tests := []struct {
		name     string
		exports  string
		expected string
	}{
		{
			name:     "dot subpath with condition",
			exports:  `{".":{"designTokens":"./tokens.resolver.json","import":"./dist/index.js"}}`,
			expected: "tokens.resolver.json",
		},
		{
			name:     "top-level condition",
			exports:  `{"designTokens":"./resolver.json","import":"./index.js"}`,
			expected: "resolver.json",
		},
		{
			name:     "no matching condition",
			exports:  `{".":{"import":"./dist/index.js"}}`,
			expected: "",
		},
		{
			name:     "string exports",
			exports:  `"./dist/index.js"`,
			expected: "",
		},
		{
			name:     "empty exports",
			exports:  ``,
			expected: "",
		},
		{
			name:     "null exports",
			exports:  `null`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw json.RawMessage
			if tt.exports != "" {
				raw = json.RawMessage(tt.exports)
			}
			result := resolveExportCondition(raw, "designTokens")
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
