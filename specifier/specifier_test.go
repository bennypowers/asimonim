/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package specifier

import "testing"

func TestParse_NPMScoped(t *testing.T) {
	spec := Parse("npm:@rhds/tokens/tokens.json")

	if spec.Kind != KindNPM {
		t.Errorf("expected Kind to be KindNPM, got %v", spec.Kind)
	}
	if spec.Package != "@rhds/tokens" {
		t.Errorf("expected Package to be '@rhds/tokens', got '%s'", spec.Package)
	}
	if spec.File != "tokens.json" {
		t.Errorf("expected File to be 'tokens.json', got '%s'", spec.File)
	}
	if spec.Raw != "npm:@rhds/tokens/tokens.json" {
		t.Errorf("expected Raw to be 'npm:@rhds/tokens/tokens.json', got '%s'", spec.Raw)
	}
}

func TestParse_NPMUnscoped(t *testing.T) {
	spec := Parse("npm:simple-tokens/colors.json")

	if spec.Kind != KindNPM {
		t.Errorf("expected Kind to be KindNPM, got %v", spec.Kind)
	}
	if spec.Package != "simple-tokens" {
		t.Errorf("expected Package to be 'simple-tokens', got '%s'", spec.Package)
	}
	if spec.File != "colors.json" {
		t.Errorf("expected File to be 'colors.json', got '%s'", spec.File)
	}
}

func TestParse_NPMNestedPath(t *testing.T) {
	spec := Parse("npm:@scope/pkg/json/tokens.json")

	if spec.Kind != KindNPM {
		t.Errorf("expected Kind to be KindNPM, got %v", spec.Kind)
	}
	if spec.Package != "@scope/pkg" {
		t.Errorf("expected Package to be '@scope/pkg', got '%s'", spec.Package)
	}
	if spec.File != "json/tokens.json" {
		t.Errorf("expected File to be 'json/tokens.json', got '%s'", spec.File)
	}
}

func TestParse_JSRScoped(t *testing.T) {
	spec := Parse("jsr:@std/tokens/mod.json")

	if spec.Kind != KindJSR {
		t.Errorf("expected Kind to be KindJSR, got %v", spec.Kind)
	}
	if spec.Package != "@std/tokens" {
		t.Errorf("expected Package to be '@std/tokens', got '%s'", spec.Package)
	}
	if spec.File != "mod.json" {
		t.Errorf("expected File to be 'mod.json', got '%s'", spec.File)
	}
}

func TestParse_JSRUnscopedIsLocal(t *testing.T) {
	// JSR requires scoped packages; unscoped specs are treated as local paths.
	spec := Parse("jsr:tokens/colors.json")

	if spec.Kind != KindLocal {
		t.Errorf("expected Kind to be KindLocal, got %v", spec.Kind)
	}
}

func TestParse_LocalPath(t *testing.T) {
	spec := Parse("./tokens/colors.json")

	if spec.Kind != KindLocal {
		t.Errorf("expected Kind to be KindLocal, got %v", spec.Kind)
	}
	if spec.File != "./tokens/colors.json" {
		t.Errorf("expected File to be './tokens/colors.json', got '%s'", spec.File)
	}
	if spec.Package != "" {
		t.Errorf("expected Package to be empty, got '%s'", spec.Package)
	}
}

func TestParse_AbsolutePath(t *testing.T) {
	spec := Parse("/home/user/tokens.json")

	if spec.Kind != KindLocal {
		t.Errorf("expected Kind to be KindLocal, got %v", spec.Kind)
	}
	if spec.File != "/home/user/tokens.json" {
		t.Errorf("expected File to be '/home/user/tokens.json', got '%s'", spec.File)
	}
}

func TestIsPackageSpecifier(t *testing.T) {
	tests := []struct {
		spec     string
		expected bool
	}{
		{"npm:@scope/pkg/file.json", true},
		{"npm:pkg/file.json", true},
		{"jsr:@scope/pkg/file.json", true},
		{"jsr:pkg/file.json", false},
		{"./local/path.json", false},
		{"/absolute/path.json", false},
		{"relative/path.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			result := IsPackageSpecifier(tt.spec)
			if result != tt.expected {
				t.Errorf("IsPackageSpecifier(%q) = %v, want %v", tt.spec, result, tt.expected)
			}
		})
	}
}

func TestSpecifier_IsNPM(t *testing.T) {
	npm := Parse("npm:pkg/file.json")
	if !npm.IsNPM() {
		t.Error("expected IsNPM() to return true for npm specifier")
	}

	local := Parse("./file.json")
	if local.IsNPM() {
		t.Error("expected IsNPM() to return false for local path")
	}
}

func TestSpecifier_IsJSR(t *testing.T) {
	jsr := Parse("jsr:@scope/pkg/file.json")
	if !jsr.IsJSR() {
		t.Error("expected IsJSR() to return true for jsr specifier")
	}

	local := Parse("./file.json")
	if local.IsJSR() {
		t.Error("expected IsJSR() to return false for local path")
	}
}

func TestSpecifier_IsLocal(t *testing.T) {
	local := Parse("./file.json")
	if !local.IsLocal() {
		t.Error("expected IsLocal() to return true for local path")
	}

	npm := Parse("npm:pkg/file.json")
	if npm.IsLocal() {
		t.Error("expected IsLocal() to return false for npm specifier")
	}
}
