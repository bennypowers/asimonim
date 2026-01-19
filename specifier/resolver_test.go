/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package specifier

import (
	"path/filepath"
	"strings"
	"testing"

	"bennypowers.dev/asimonim/internal/mapfs"
)

func TestLocalResolver_Passthrough(t *testing.T) {
	resolver := NewLocalResolver()

	tests := []struct {
		name string
		spec string
	}{
		{"relative path", "./tokens/colors.json"},
		{"absolute path", "/home/user/tokens.json"},
		{"simple name", "tokens.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rf, err := resolver.Resolve(tt.spec)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rf.Specifier != tt.spec {
				t.Errorf("Specifier = %q, want %q", rf.Specifier, tt.spec)
			}
			if rf.Path != tt.spec {
				t.Errorf("Path = %q, want %q", rf.Path, tt.spec)
			}
			if rf.Kind != KindLocal {
				t.Errorf("Kind = %v, want KindLocal", rf.Kind)
			}
		})
	}
}

func TestLocalResolver_CanResolve(t *testing.T) {
	resolver := NewLocalResolver()

	if !resolver.CanResolve("./tokens.json") {
		t.Error("expected CanResolve to return true for local path")
	}
	if resolver.CanResolve("npm:pkg/file.json") {
		t.Error("expected CanResolve to return false for npm specifier")
	}
	if resolver.CanResolve("jsr:pkg/file.json") {
		t.Error("expected CanResolve to return false for jsr specifier")
	}
}

func TestNPMResolver_ScopedPackage(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/node_modules/@design-tokens/test-package/tokens.json", `{"color":{}}`, 0644)

	resolver := NewNPMResolver(mfs, "/project")

	rf, err := resolver.Resolve("npm:@design-tokens/test-package/tokens.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rf.Specifier != "npm:@design-tokens/test-package/tokens.json" {
		t.Errorf("Specifier = %q, want %q", rf.Specifier, "npm:@design-tokens/test-package/tokens.json")
	}
	expectedPath := "/project/node_modules/@design-tokens/test-package/tokens.json"
	if rf.Path != expectedPath {
		t.Errorf("Path = %q, want %q", rf.Path, expectedPath)
	}
	if rf.Kind != KindNPM {
		t.Errorf("Kind = %v, want KindNPM", rf.Kind)
	}
}

func TestNPMResolver_UnscopedPackage(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/node_modules/simple-tokens/colors.json", `{"color":{}}`, 0644)

	resolver := NewNPMResolver(mfs, "/project")

	rf, err := resolver.Resolve("npm:simple-tokens/colors.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedPath := "/project/node_modules/simple-tokens/colors.json"
	if rf.Path != expectedPath {
		t.Errorf("Path = %q, want %q", rf.Path, expectedPath)
	}
}

func TestNPMResolver_WalksUpDirectoryTree(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/node_modules/parent-tokens/tokens.json", `{"spacing":{}}`, 0644)
	mfs.AddDir("/project/subdir", 0755)

	resolver := NewNPMResolver(mfs, "/project/subdir")

	rf, err := resolver.Resolve("npm:parent-tokens/tokens.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedPath := "/project/node_modules/parent-tokens/tokens.json"
	if rf.Path != expectedPath {
		t.Errorf("Path = %q, want %q", rf.Path, expectedPath)
	}
}

func TestNPMResolver_PackageNotFound(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddDir("/project", 0755)

	resolver := NewNPMResolver(mfs, "/project")

	_, err := resolver.Resolve("npm:nonexistent/tokens.json")
	if err == nil {
		t.Fatal("expected error for nonexistent package")
	}
	if !strings.Contains(err.Error(), "package not found") {
		t.Errorf("error = %q, want to contain 'package not found'", err.Error())
	}
}

func TestNPMResolver_CanResolve(t *testing.T) {
	mfs := mapfs.New()
	resolver := NewNPMResolver(mfs, "/project")

	if !resolver.CanResolve("npm:pkg/file.json") {
		t.Error("expected CanResolve to return true for npm specifier")
	}
	if resolver.CanResolve("jsr:pkg/file.json") {
		t.Error("expected CanResolve to return false for jsr specifier")
	}
	if resolver.CanResolve("./local.json") {
		t.Error("expected CanResolve to return false for local path")
	}
}

func TestJSRResolver_NotImplemented(t *testing.T) {
	resolver := NewJSRResolver()

	_, err := resolver.Resolve("jsr:@std/tokens/mod.json")
	if err == nil {
		t.Fatal("expected error for jsr specifier")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("error = %q, want to contain 'not implemented'", err.Error())
	}
}

func TestJSRResolver_CanResolve(t *testing.T) {
	resolver := NewJSRResolver()

	if !resolver.CanResolve("jsr:pkg/file.json") {
		t.Error("expected CanResolve to return true for jsr specifier")
	}
	if resolver.CanResolve("npm:pkg/file.json") {
		t.Error("expected CanResolve to return false for npm specifier")
	}
}

func TestChainResolver_TriesInOrder(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/node_modules/@scope/pkg/file.json", `{}`, 0644)

	chain := NewChainResolver(
		NewNPMResolver(mfs, "/project"),
		NewJSRResolver(),
		NewLocalResolver(),
	)

	// npm: should be handled by NPMResolver
	rf, err := chain.Resolve("npm:@scope/pkg/file.json")
	if err != nil {
		t.Fatalf("unexpected error for npm: %v", err)
	}
	if rf.Kind != KindNPM {
		t.Errorf("Kind = %v, want KindNPM", rf.Kind)
	}

	// jsr: should be handled by JSRResolver (returns error)
	_, err = chain.Resolve("jsr:pkg/file.json")
	if err == nil {
		t.Fatal("expected error for jsr specifier")
	}

	// local path should be handled by LocalResolver
	rf, err = chain.Resolve("./local.json")
	if err != nil {
		t.Fatalf("unexpected error for local: %v", err)
	}
	if rf.Kind != KindLocal {
		t.Errorf("Kind = %v, want KindLocal", rf.Kind)
	}
}

func TestChainResolver_CanResolve(t *testing.T) {
	mfs := mapfs.New()
	chain := NewChainResolver(
		NewNPMResolver(mfs, "/project"),
		NewLocalResolver(),
	)

	if !chain.CanResolve("npm:pkg/file.json") {
		t.Error("expected CanResolve to return true for npm specifier")
	}
	if !chain.CanResolve("./local.json") {
		t.Error("expected CanResolve to return true for local path")
	}
}

func TestDefaultResolver_EndToEnd(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/node_modules/@rhds/tokens/json/rhds.tokens.json", `{"color":{}}`, 0644)

	resolver := NewDefaultResolver(mfs, "/project")

	// Test npm resolution
	rf, err := resolver.Resolve("npm:@rhds/tokens/json/rhds.tokens.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rf.Specifier != "npm:@rhds/tokens/json/rhds.tokens.json" {
		t.Errorf("Specifier = %q, want %q", rf.Specifier, "npm:@rhds/tokens/json/rhds.tokens.json")
	}
	expectedPath := filepath.Join("/project", "node_modules", "@rhds", "tokens", "json", "rhds.tokens.json")
	if rf.Path != expectedPath {
		t.Errorf("Path = %q, want %q", rf.Path, expectedPath)
	}

	// Test local resolution
	rf, err = resolver.Resolve("./tokens.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rf.Path != "./tokens.json" {
		t.Errorf("Path = %q, want %q", rf.Path, "./tokens.json")
	}

	// Test jsr: returns error
	_, err = resolver.Resolve("jsr:pkg/file.json")
	if err == nil {
		t.Fatal("expected error for jsr specifier")
	}
}
