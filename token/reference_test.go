/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package token_test

import (
	"testing"

	"bennypowers.dev/asimonim/token"
)

func TestParseCurlyBraceRef(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantPath string
		wantOK   bool
	}{
		{"valid ref", "{color.primary}", "color.primary", true},
		{"nested ref", "{color.brand.primary}", "color.brand.primary", true},
		{"no braces", "color.primary", "", false},
		{"empty braces", "{}", "", false},
		{"partial ref", "some {color.primary} text", "color.primary", true},
		{"nested braces", "{{nested}}", "nested", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, ok := token.ParseCurlyBraceRef(tt.input)
			if ok != tt.wantOK {
				t.Errorf("ParseCurlyBraceRef(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if path != tt.wantPath {
				t.Errorf("ParseCurlyBraceRef(%q) path = %q, want %q", tt.input, path, tt.wantPath)
			}
		})
	}
}

func TestParseJSONPointerRef(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantPath string
		wantOK   bool
	}{
		{"valid pointer", "#/color/primary", "color.primary", true},
		{"nested pointer", "#/color/brand/primary", "color.brand.primary", true},
		{"no hash prefix", "color/primary", "", false},
		{"just hash", "#/", "", false},
		{"empty", "", "", false},
		// RFC 6901: ~1 decodes to /, ~0 decodes to ~
		{"escaped slash", "#/color~1brand/primary", "color/brand.primary", true},
		{"escaped tilde", "#/color~0brand/primary", "color~brand.primary", true},
		{"both escapes", "#/a~0b~1c", "a~b/c", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, ok := token.ParseJSONPointerRef(tt.input)
			if ok != tt.wantOK {
				t.Errorf("ParseJSONPointerRef(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if path != tt.wantPath {
				t.Errorf("ParseJSONPointerRef(%q) path = %q, want %q", tt.input, path, tt.wantPath)
			}
		})
	}
}

func TestIsCurlyBraceRef(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"{color.primary}", true},
		{"some {ref} text", true},
		{"no ref here", false},
		{"{}", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := token.IsCurlyBraceRef(tt.input); got != tt.want {
				t.Errorf("IsCurlyBraceRef(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsJSONPointerRef(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"#/color/primary", true},
		{"#/a", true},
		{"color/primary", false},
		{"#/", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := token.IsJSONPointerRef(tt.input); got != tt.want {
				t.Errorf("IsJSONPointerRef(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractAllRefs(t *testing.T) {
	tests := []struct {
		name string
		input string
		want  []string
	}{
		{"single ref", "{color.primary}", []string{"color.primary"}},
		{"multiple refs", "{a.b} and {c.d}", []string{"a.b", "c.d"}},
		{"no refs", "plain text", []string{}},
		{"embedded refs", "rgb({r}, {g}, {b})", []string{"r", "g", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := token.ExtractAllRefs(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("ExtractAllRefs(%q) returned %d refs, want %d", tt.input, len(got), len(tt.want))
			}
			for i, ref := range got {
				if ref != tt.want[i] {
					t.Errorf("ExtractAllRefs(%q)[%d] = %q, want %q", tt.input, i, ref, tt.want[i])
				}
			}
		})
	}
}
