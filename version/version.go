/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

// Package version provides build and version information for asimonim.
//
// The package is part of the public API so downstream tools that embed
// asimonim can report the version they are using in diagnostics and user
// agent strings. Get returns the human-readable version, Info returns a
// structured view for JSON output, and Full includes the Git commit when
// available.
//
// Build metadata is normally injected with -ldflags. For example:
//
//	go build -ldflags "-X bennypowers.dev/asimonim/version.Version=v0.3.3"
//
// When no metadata is injected, Get falls back to the module version
// recorded by the Go toolchain, then to "dev".
package version

import (
	"fmt"
	"runtime/debug"
	"strings"
)

var (
	// Version is the semantic version, set at build time via ldflags.
	// It is "dev" when the binary was built without version metadata.
	Version = "dev"

	// GitCommit is the Git commit hash, set at build time via ldflags.
	GitCommit = "unknown"

	// GitTag is the exact Git tag for the build, set at build time via ldflags.
	GitTag = "unknown"

	// BuildTime is the build timestamp, set at build time via ldflags.
	BuildTime = "unknown"

	// GitDirty is "dirty" when the worktree had uncommitted changes at
	// build time, and empty otherwise.
	GitDirty = ""
)

// Get returns the version string for the application.
//
// It prefers the Version injected at build time, then the module version
// reported by the Go toolchain, then Git metadata, and finally "dev".
func Get() string {
	if Version != "dev" {
		return Version
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			return info.Main.Version
		}
	}

	if GitTag != "unknown" && GitCommit != "unknown" {
		version := GitTag
		if GitCommit != "" {
			commitSuffix := GitCommit
			if len(GitCommit) > 7 {
				commitSuffix = GitCommit[:7]
			}
			if !strings.HasSuffix(GitTag, commitSuffix) {
				version = fmt.Sprintf("%s-%s", GitTag, commitSuffix)
			}
		}
		if GitDirty == "dirty" {
			version += "-dirty"
		}
		return version
	}

	return "dev"
}

// Full returns detailed version information.
func Full() string {
	version := Get()
	if GitCommit != "unknown" {
		return fmt.Sprintf("%s (commit: %s)", version, GitCommit)
	}
	return version
}

// Info returns detailed build information.
//
// The returned map always contains the keys "version", "gitCommit",
// "gitTag", "buildTime", and "gitDirty".
func Info() map[string]string {
	return map[string]string{
		"version":   Get(),
		"gitCommit": GitCommit,
		"gitTag":    GitTag,
		"buildTime": BuildTime,
		"gitDirty":  GitDirty,
	}
}
