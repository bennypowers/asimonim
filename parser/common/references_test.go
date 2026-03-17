/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package common_test

import (
	"testing"

	"bennypowers.dev/asimonim/parser/common"
	"bennypowers.dev/asimonim/schema"
)

func TestExtractReferences(t *testing.T) {
	tests := []struct {
		name    string
		content string
		version schema.Version
		want    int
		paths   []string
	}{
		{
			name:    "single curly brace ref",
			content: "{color.primary}",
			version: schema.Draft,
			want:    1,
			paths:   []string{"color.primary"},
		},
		{
			name:    "multiple refs",
			content: "{a.b} and {c.d}",
			version: schema.Draft,
			want:    2,
			paths:   []string{"a.b", "c.d"},
		},
		{
			name:    "no refs",
			content: "plain text",
			version: schema.Draft,
			want:    0,
		},
		{
			name:    "v2025_10 curly brace",
			content: "{color.primary}",
			version: schema.V2025_10,
			want:    1,
			paths:   []string{"color.primary"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs, err := common.ExtractReferences(tt.content, tt.version)
			if err != nil {
				t.Fatalf("ExtractReferences() error = %v", err)
			}
			if len(refs) != tt.want {
				t.Fatalf("ExtractReferences() returned %d refs, want %d", len(refs), tt.want)
			}
			for i, ref := range refs {
				if ref.Type != common.CurlyBraceReference {
					t.Errorf("ref[%d].Type = %v, want CurlyBraceReference", i, ref.Type)
				}
				if i < len(tt.paths) && ref.Path != tt.paths[i] {
					t.Errorf("ref[%d].Path = %q, want %q", i, ref.Path, tt.paths[i])
				}
			}
		})
	}
}

func TestExtractReferencesFromValue(t *testing.T) {
	t.Run("string value with curly ref", func(t *testing.T) {
		refs, err := common.ExtractReferencesFromValue("{color.primary}", schema.Draft)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(refs) != 1 {
			t.Fatalf("got %d refs, want 1", len(refs))
		}
		// {color.primary} → path "color.primary"
		if refs[0].Path != "color.primary" {
			t.Errorf("path = %q, want %q", refs[0].Path, "color.primary")
		}
	})

	t.Run("map with $ref in v2025_10", func(t *testing.T) {
		value := map[string]any{"$ref": "#/color/primary"}
		refs, err := common.ExtractReferencesFromValue(value, schema.V2025_10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(refs) != 1 {
			t.Fatalf("got %d refs, want 1", len(refs))
		}
		if refs[0].Type != common.JSONPointerReference {
			t.Errorf("type = %v, want JSONPointerReference", refs[0].Type)
		}
		// $ref "#/color/primary" → path "color/primary" (prefix stripped)
		if refs[0].Path != "color/primary" {
			t.Errorf("path = %q, want %q", refs[0].Path, "color/primary")
		}
	})

	t.Run("map with $ref in draft returns error", func(t *testing.T) {
		value := map[string]any{"$ref": "#/color/primary"}
		_, err := common.ExtractReferencesFromValue(value, schema.Draft)
		if err != schema.ErrInvalidReference {
			t.Errorf("expected ErrInvalidReference, got %v", err)
		}
	})

	t.Run("map without $ref", func(t *testing.T) {
		value := map[string]any{"colorSpace": "srgb"}
		refs, err := common.ExtractReferencesFromValue(value, schema.V2025_10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(refs) != 0 {
			t.Errorf("got %d refs, want 0", len(refs))
		}
	})

	t.Run("non-string non-map value", func(t *testing.T) {
		refs, err := common.ExtractReferencesFromValue(42, schema.Draft)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(refs) != 0 {
			t.Errorf("got %d refs, want 0", len(refs))
		}
	})
}

func TestConvertJSONPointerToTokenPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"#/color/brand/primary", "color.brand.primary"},
		{"color/brand/primary", "color.brand.primary"},
		{"#/single", "single"},
		{"single", "single"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := common.ConvertJSONPointerToTokenPath(tt.input)
			if got != tt.want {
				t.Errorf("ConvertJSONPointerToTokenPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestConvertTokenPathToJSONPointer(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"color.brand.primary", "#/color/brand/primary"},
		{"single", "#/single"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := common.ConvertTokenPathToJSONPointer(tt.input)
			if got != tt.want {
				t.Errorf("ConvertTokenPathToJSONPointer(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
