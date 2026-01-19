/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package specifier

import (
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

func TestNodeModulesResolver_ScopedPackage(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/node_modules/@design-tokens/test-package/tokens.json", `{"color":{}}`, 0644)

	resolver, err := NewNodeModulesResolver(mfs, "/project")
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

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

func TestNodeModulesResolver_UnscopedPackage(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/node_modules/simple-tokens/colors.json", `{"color":{}}`, 0644)

	resolver, err := NewNodeModulesResolver(mfs, "/project")
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	rf, err := resolver.Resolve("npm:simple-tokens/colors.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedPath := "/project/node_modules/simple-tokens/colors.json"
	if rf.Path != expectedPath {
		t.Errorf("Path = %q, want %q", rf.Path, expectedPath)
	}
}

func TestNodeModulesResolver_WalksUpDirectoryTree(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/node_modules/parent-tokens/tokens.json", `{"spacing":{}}`, 0644)
	mfs.AddDir("/project/subdir", 0755)

	resolver, err := NewNodeModulesResolver(mfs, "/project/subdir")
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	rf, err := resolver.Resolve("npm:parent-tokens/tokens.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedPath := "/project/node_modules/parent-tokens/tokens.json"
	if rf.Path != expectedPath {
		t.Errorf("Path = %q, want %q", rf.Path, expectedPath)
	}
}

func TestNodeModulesResolver_PackageNotFound(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddDir("/project", 0755)

	resolver, err := NewNodeModulesResolver(mfs, "/project")
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	_, err = resolver.Resolve("npm:nonexistent/tokens.json")
	if err == nil {
		t.Fatal("expected error for nonexistent package")
	}
	if !strings.Contains(err.Error(), "package not found") {
		t.Errorf("error = %q, want to contain 'package not found'", err.Error())
	}
}

func TestNodeModulesResolver_CanResolve(t *testing.T) {
	mfs := mapfs.New()
	resolver, err := NewNodeModulesResolver(mfs, "/project")
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

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

func TestJSRNodeModulesResolver_ScopedPackage(t *testing.T) {
	mfs := mapfs.New()
	// JSR scoped package: jsr:@design-tokens/test → @jsr/design-tokens__test
	mfs.AddFile("/project/node_modules/@jsr/design-tokens__test/tokens.json", `{"color":{}}`, 0644)

	resolver, err := NewJSRNodeModulesResolver(mfs, "/project")
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	rf, err := resolver.Resolve("jsr:@design-tokens/test/tokens.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rf.Specifier != "jsr:@design-tokens/test/tokens.json" {
		t.Errorf("Specifier = %q, want %q", rf.Specifier, "jsr:@design-tokens/test/tokens.json")
	}
	expectedPath := "/project/node_modules/@jsr/design-tokens__test/tokens.json"
	if rf.Path != expectedPath {
		t.Errorf("Path = %q, want %q", rf.Path, expectedPath)
	}
	if rf.Kind != KindJSR {
		t.Errorf("Kind = %v, want KindJSR", rf.Kind)
	}
}

func TestJSRNodeModulesResolver_UnscopedPackage(t *testing.T) {
	mfs := mapfs.New()
	// JSR unscoped package: jsr:simple-tokens → @jsr/simple-tokens
	mfs.AddFile("/project/node_modules/@jsr/simple-tokens/colors.json", `{"color":{}}`, 0644)

	resolver, err := NewJSRNodeModulesResolver(mfs, "/project")
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	rf, err := resolver.Resolve("jsr:simple-tokens/colors.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedPath := "/project/node_modules/@jsr/simple-tokens/colors.json"
	if rf.Path != expectedPath {
		t.Errorf("Path = %q, want %q", rf.Path, expectedPath)
	}
}

func TestJSRNodeModulesResolver_WalksUpDirectoryTree(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/node_modules/@jsr/std__tokens/tokens.json", `{"spacing":{}}`, 0644)
	mfs.AddDir("/project/subdir", 0755)

	resolver, err := NewJSRNodeModulesResolver(mfs, "/project/subdir")
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	rf, err := resolver.Resolve("jsr:@std/tokens/tokens.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedPath := "/project/node_modules/@jsr/std__tokens/tokens.json"
	if rf.Path != expectedPath {
		t.Errorf("Path = %q, want %q", rf.Path, expectedPath)
	}
}

func TestJSRNodeModulesResolver_PackageNotFound(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddDir("/project", 0755)

	resolver, err := NewJSRNodeModulesResolver(mfs, "/project")
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	_, err = resolver.Resolve("jsr:@nonexistent/pkg/tokens.json")
	if err == nil {
		t.Fatal("expected error for nonexistent package")
	}
	if !strings.Contains(err.Error(), "jsr package not found") {
		t.Errorf("error = %q, want to contain 'jsr package not found'", err.Error())
	}
}

func TestJSRNodeModulesResolver_CanResolve(t *testing.T) {
	mfs := mapfs.New()
	resolver, err := NewJSRNodeModulesResolver(mfs, "/project")
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

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
	mfs.AddFile("/project/node_modules/@jsr/std__tokens/mod.json", `{}`, 0644)

	npmResolver, err := NewNodeModulesResolver(mfs, "/project")
	if err != nil {
		t.Fatalf("failed to create npm resolver: %v", err)
	}
	jsrResolver, err := NewJSRNodeModulesResolver(mfs, "/project")
	if err != nil {
		t.Fatalf("failed to create jsr resolver: %v", err)
	}
	chain := NewChainResolver(
		npmResolver,
		jsrResolver,
		NewLocalResolver(),
	)

	// npm: should be handled by NodeModulesResolver
	rf, err := chain.Resolve("npm:@scope/pkg/file.json")
	if err != nil {
		t.Fatalf("unexpected error for npm: %v", err)
	}
	if rf.Kind != KindNPM {
		t.Errorf("Kind = %v, want KindNPM", rf.Kind)
	}

	// jsr: should be handled by JSRNodeModulesResolver
	rf, err = chain.Resolve("jsr:@std/tokens/mod.json")
	if err != nil {
		t.Fatalf("unexpected error for jsr: %v", err)
	}
	if rf.Kind != KindJSR {
		t.Errorf("Kind = %v, want KindJSR", rf.Kind)
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
	npmResolver, err := NewNodeModulesResolver(mfs, "/project")
	if err != nil {
		t.Fatalf("failed to create npm resolver: %v", err)
	}
	chain := NewChainResolver(
		npmResolver,
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
	mfs.AddFile("/project/node_modules/@jsr/luca__cases/mod.json", `{"case":{}}`, 0644)

	resolver, err := NewDefaultResolver(mfs, "/project")
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	// Test npm resolution
	rf, err := resolver.Resolve("npm:@rhds/tokens/json/rhds.tokens.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rf.Specifier != "npm:@rhds/tokens/json/rhds.tokens.json" {
		t.Errorf("Specifier = %q, want %q", rf.Specifier, "npm:@rhds/tokens/json/rhds.tokens.json")
	}
	expectedPath := "/project/node_modules/@rhds/tokens/json/rhds.tokens.json"
	if rf.Path != expectedPath {
		t.Errorf("Path = %q, want %q", rf.Path, expectedPath)
	}

	// Test jsr resolution
	rf, err = resolver.Resolve("jsr:@luca/cases/mod.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rf.Specifier != "jsr:@luca/cases/mod.json" {
		t.Errorf("Specifier = %q, want %q", rf.Specifier, "jsr:@luca/cases/mod.json")
	}
	expectedJSRPath := "/project/node_modules/@jsr/luca__cases/mod.json"
	if rf.Path != expectedJSRPath {
		t.Errorf("Path = %q, want %q", rf.Path, expectedJSRPath)
	}
	if rf.Kind != KindJSR {
		t.Errorf("Kind = %v, want KindJSR", rf.Kind)
	}

	// Test local resolution
	rf, err = resolver.Resolve("./tokens.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rf.Path != "./tokens.json" {
		t.Errorf("Path = %q, want %q", rf.Path, "./tokens.json")
	}
}
