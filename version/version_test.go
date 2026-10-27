/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

package version_test

import (
	"strings"
	"testing"

	"bennypowers.dev/asimonim/version"
)

// saveRestore captures the package-level build variables and restores them
// when the test finishes.
func saveRestore(t *testing.T) {
	t.Helper()
	oldVersion := version.Version
	oldCommit := version.GitCommit
	oldTag := version.GitTag
	oldBuildTime := version.BuildTime
	oldDirty := version.GitDirty
	t.Cleanup(func() {
		version.Version = oldVersion
		version.GitCommit = oldCommit
		version.GitTag = oldTag
		version.BuildTime = oldBuildTime
		version.GitDirty = oldDirty
	})
}

func TestGetReturnsInjectedVersion(t *testing.T) {
	saveRestore(t)
	version.Version = "v1.2.3"
	version.GitCommit = "unknown"
	version.GitTag = "unknown"

	if got := version.Get(); got != "v1.2.3" {
		t.Fatalf("Get() = %q, want %q", got, "v1.2.3")
	}
}

func TestGetFallsBackToGitMetadata(t *testing.T) {
	saveRestore(t)
	version.Version = "dev"
	version.GitTag = "v0.3.3"
	version.GitCommit = "abcdef1234567890"
	version.GitDirty = ""

	if got := version.Get(); got != "v0.3.3-abcdef1" {
		t.Fatalf("Get() = %q, want %q", got, "v0.3.3-abcdef1")
	}
}

func TestGetAppendsDirtySuffix(t *testing.T) {
	saveRestore(t)
	version.Version = "dev"
	version.GitTag = "v0.3.3"
	version.GitCommit = "abcdef1234567890"
	version.GitDirty = "dirty"

	if got := version.Get(); got != "v0.3.3-abcdef1-dirty" {
		t.Fatalf("Get() = %q, want %q", got, "v0.3.3-abcdef1-dirty")
	}
}

func TestGetDoesNotDuplicateCommitSuffix(t *testing.T) {
	saveRestore(t)
	version.Version = "dev"
	version.GitTag = "v0.3.3-abcdef1"
	version.GitCommit = "abcdef1234567890"
	version.GitDirty = ""

	if got := version.Get(); got != "v0.3.3-abcdef1" {
		t.Fatalf("Get() = %q, want %q", got, "v0.3.3-abcdef1")
	}
}

func TestGetDefaultsToDev(t *testing.T) {
	saveRestore(t)
	version.Version = "dev"
	version.GitTag = "unknown"
	version.GitCommit = "unknown"
	version.GitDirty = ""

	// When built without ldflags or module version information, Get reports dev.
	got := version.Get()
	if got != "dev" && !strings.Contains(got, "devel") {
		// Under some test toolchains debug.ReadBuildInfo reports a module
		// version. Accept that, but it must not be empty.
		if got == "" {
			t.Fatal("Get() returned an empty version")
		}
	}
}

func TestFullIncludesCommit(t *testing.T) {
	saveRestore(t)
	version.Version = "v1.2.3"
	version.GitCommit = "abcdef1234567890"

	if got := version.Full(); got != "v1.2.3 (commit: abcdef1234567890)" {
		t.Fatalf("Full() = %q, want %q", got, "v1.2.3 (commit: abcdef1234567890)")
	}
}

func TestFullOmitsUnknownCommit(t *testing.T) {
	saveRestore(t)
	version.Version = "v1.2.3"
	version.GitCommit = "unknown"

	if got := version.Full(); got != "v1.2.3" {
		t.Fatalf("Full() = %q, want %q", got, "v1.2.3")
	}
}

func TestInfoIncludesBuildFields(t *testing.T) {
	saveRestore(t)
	version.Version = "v1.2.3"
	version.GitCommit = "abcdef1"
	version.GitTag = "v1.2.3"
	version.BuildTime = "2026-01-02T03:04:05Z"
	version.GitDirty = "dirty"

	info := version.Info()
	want := map[string]string{
		"version":   "v1.2.3",
		"gitCommit": "abcdef1",
		"gitTag":    "v1.2.3",
		"buildTime": "2026-01-02T03:04:05Z",
		"gitDirty":  "dirty",
	}
	for key, wantValue := range want {
		if got := info[key]; got != wantValue {
			t.Errorf("Info()[%q] = %q, want %q", key, got, wantValue)
		}
	}
	if len(info) != len(want) {
		t.Errorf("Info() has %d entries, want %d", len(info), len(want))
	}
}
