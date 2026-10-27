/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

package version

import (
	"runtime/debug"
	"testing"
)

func buildInfo(main debug.Module, deps ...*debug.Module) func() (*debug.BuildInfo, bool) {
	return func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: main, Deps: deps}, true
	}
}

func TestGetUsesMainModuleVersionForCLI(t *testing.T) {
	got := get("dev", "unknown", "unknown", "", buildInfo(debug.Module{
		Path:    modulePath,
		Version: "v1.2.3",
	}))
	if got != "v1.2.3" {
		t.Fatalf("get() = %q, want %q", got, "v1.2.3")
	}
}

func TestGetUsesAsimonimDependencyVersionForEmbedder(t *testing.T) {
	got := get("dev", "unknown", "unknown", "", buildInfo(
		debug.Module{Path: "example.com/host", Version: "v9.9.9"},
		&debug.Module{Path: modulePath, Version: "v0.3.3"},
	))
	if got != "v0.3.3" {
		t.Fatalf("get() = %q, want the asimonim dependency version %q", got, "v0.3.3")
	}
}

func TestGetUsesReplacementVersionForEmbedder(t *testing.T) {
	got := get("dev", "unknown", "unknown", "", buildInfo(
		debug.Module{Path: "example.com/host", Version: "v9.9.9"},
		&debug.Module{
			Path:    modulePath,
			Version: "v0.0.0",
			Replace: &debug.Module{Path: modulePath, Version: "v0.4.0"},
		},
	))
	if got != "v0.4.0" {
		t.Fatalf("get() = %q, want replacement version %q", got, "v0.4.0")
	}
}

func TestGetFallsBackWhenDependencyMissing(t *testing.T) {
	got := get("dev", "unknown", "unknown", "", buildInfo(
		debug.Module{Path: "example.com/host", Version: "v9.9.9"},
	))
	if got != "dev" {
		t.Fatalf("get() = %q, want %q", got, "dev")
	}
}

func TestGetFallsBackToGitMetadataWhenDependencyMissing(t *testing.T) {
	got := get("dev", "abcdef1234567890", "v0.3.3", "", buildInfo(
		debug.Module{Path: "example.com/host", Version: "v9.9.9"},
	))
	if got != "v0.3.3-abcdef1" {
		t.Fatalf("get() = %q, want %q", got, "v0.3.3-abcdef1")
	}
}

func TestGetPrefersInjectedVersion(t *testing.T) {
	got := get("v2.0.0", "unknown", "unknown", "", func() (*debug.BuildInfo, bool) {
		t.Fatal("build info should not be consulted when Version is injected")
		return nil, false
	})
	if got != "v2.0.0" {
		t.Fatalf("get() = %q, want %q", got, "v2.0.0")
	}
}

func TestGetFallsBackWhenInjectedVersionIsEmpty(t *testing.T) {
	got := get("", "unknown", "unknown", "", buildInfo(debug.Module{
		Path:    modulePath,
		Version: "v1.2.3",
	}))
	if got != "v1.2.3" {
		t.Fatalf("get() = %q, want fallback version %q", got, "v1.2.3")
	}
}

func TestModuleVersionSkipsLocalPathReplacement(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{Path: "example.com/host", Version: "v9.9.9"},
		Deps: []*debug.Module{{
			Path:    modulePath,
			Version: "v0.3.3",
			Replace: &debug.Module{Path: "/local/asimonim", Version: "v0.5.0"},
		}},
	}
	if got := moduleVersion(info); got != "" {
		t.Fatalf("moduleVersion() = %q, want empty for local path replacement", got)
	}
}
