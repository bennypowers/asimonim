/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package convert

import (
	"testing"

	"bennypowers.dev/asimonim/internal/mapfs"
	"bennypowers.dev/asimonim/token"
)

func TestGetSplitKey(t *testing.T) {
	tests := []struct {
		name    string
		tok     *token.Token
		splitBy string
		want    string
	}{
		{
			name:    "topLevel default",
			tok:     &token.Token{Path: []string{"color", "brand", "primary"}},
			splitBy: "topLevel",
			want:    "color",
		},
		{
			name:    "empty splitBy defaults to topLevel",
			tok:     &token.Token{Path: []string{"color", "primary"}},
			splitBy: "",
			want:    "color",
		},
		{
			name:    "topLevel with empty path",
			tok:     &token.Token{Path: []string{}},
			splitBy: "topLevel",
			want:    "other",
		},
		{
			name:    "type split",
			tok:     &token.Token{Type: "color", Path: []string{"a"}},
			splitBy: "type",
			want:    "color",
		},
		{
			name:    "type split empty type",
			tok:     &token.Token{Type: "", Path: []string{"a"}},
			splitBy: "type",
			want:    "other",
		},
		{
			name:    "path[0]",
			tok:     &token.Token{Path: []string{"color", "brand", "primary"}},
			splitBy: "path[0]",
			want:    "color",
		},
		{
			name:    "path[1]",
			tok:     &token.Token{Path: []string{"color", "brand", "primary"}},
			splitBy: "path[1]",
			want:    "brand",
		},
		{
			name:    "path[N] out of bounds",
			tok:     &token.Token{Path: []string{"color"}},
			splitBy: "path[5]",
			want:    "color",
		},
		{
			name:    "unknown split strategy falls back to topLevel",
			tok:     &token.Token{Path: []string{"color", "primary"}},
			splitBy: "unknown",
			want:    "color",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSplitKey(tt.tok, tt.splitBy)
			if got != tt.want {
				t.Errorf("getSplitKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSanitizeGroupName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"color", "color"},
		{"color-brand", "color-brand"},
		{"../etc/passwd", "__etc_passwd"},
		{"foo/bar", "foo_bar"},
		{"foo\\bar", "foo_bar"},
		{"hello world", "hello_world"},
		{"valid.name", "valid.name"},
		{"under_score", "under_score"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeGroupName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeGroupName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGroupTokens(t *testing.T) {
	tokens := []*token.Token{
		{Name: "color-primary", Path: []string{"color", "primary"}, Type: "color"},
		{Name: "color-secondary", Path: []string{"color", "secondary"}, Type: "color"},
		{Name: "spacing-small", Path: []string{"spacing", "small"}, Type: "dimension"},
	}

	groups := groupTokens(tokens, "topLevel")

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if len(groups["color"]) != 2 {
		t.Errorf("expected 2 color tokens, got %d", len(groups["color"]))
	}
	if len(groups["spacing"]) != 1 {
		t.Errorf("expected 1 spacing token, got %d", len(groups["spacing"]))
	}
}

func TestGroupTokens_ByType(t *testing.T) {
	tokens := []*token.Token{
		{Name: "color-primary", Type: "color", Path: []string{"color", "primary"}},
		{Name: "spacing-small", Type: "dimension", Path: []string{"spacing", "small"}},
		{Name: "spacing-large", Type: "dimension", Path: []string{"spacing", "large"}},
	}

	groups := groupTokens(tokens, "type")

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups (color, dimension), got %d", len(groups))
	}
	if len(groups["dimension"]) != 2 {
		t.Errorf("expected 2 dimension tokens, got %d", len(groups["dimension"]))
	}
}

func TestEnsureDir(t *testing.T) {
	mfs := mapfs.New()

	// Current dir should be a no-op
	err := ensureDir(mfs, "file.txt")
	if err != nil {
		t.Errorf("ensureDir for current dir failed: %v", err)
	}

	// Nested path should create parent dirs
	err = ensureDir(mfs, "/output/subdir/file.txt")
	if err != nil {
		t.Errorf("ensureDir for nested path failed: %v", err)
	}

	if !mfs.Exists("/output/subdir") {
		t.Error("expected /output/subdir to be created")
	}
}

func TestResolveHeader(t *testing.T) {
	mfs := mapfs.New()

	// Test inline header (flag takes precedence)
	header, err := resolveHeader(mfs, "Copyright 2026", "fallback")
	if err != nil {
		t.Fatalf("resolveHeader error: %v", err)
	}
	if header != "Copyright 2026" {
		t.Errorf("expected inline header, got %q", header)
	}

	// Test config fallback
	header, err = resolveHeader(mfs, "", "Config Header")
	if err != nil {
		t.Fatalf("resolveHeader error: %v", err)
	}
	if header != "Config Header" {
		t.Errorf("expected config header, got %q", header)
	}

	// Test empty header
	header, err = resolveHeader(mfs, "", "")
	if err != nil {
		t.Fatalf("resolveHeader error: %v", err)
	}
	if header != "" {
		t.Errorf("expected empty header, got %q", header)
	}

	// Test @file reference
	mfs.AddFile("/header.txt", "File-based header", 0644)
	header, err = resolveHeader(mfs, "@/header.txt", "")
	if err != nil {
		t.Fatalf("resolveHeader @file error: %v", err)
	}
	if header != "File-based header" {
		t.Errorf("expected file-based header, got %q", header)
	}

	// Test @file with nonexistent file
	_, err = resolveHeader(mfs, "@/nonexistent.txt", "")
	if err == nil {
		t.Error("expected error for nonexistent header file")
	}
}

func TestComputeTypesPath(t *testing.T) {
	path := computeTypesPath("/output/{group}.ts")
	if path != "/output/types.ts" {
		t.Errorf("computeTypesPath() = %q, want %q", path, "/output/types.ts")
	}
}

func TestComputeSharedTypesImport(t *testing.T) {
	imp := computeSharedTypesImport("/output/{group}.ts", "/output/color.ts")
	if imp != "./types.ts" {
		t.Errorf("computeSharedTypesImport() = %q, want %q", imp, "./types.ts")
	}
}
