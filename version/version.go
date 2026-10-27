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
// When no metadata is injected, Get reads the Go build information. If
// asimonim is the main module, it uses that module's version. If asimonim is
// embedded as a dependency, it reports the asimonim dependency version rather
// than the host program's version. If no version is available, Get returns
// "dev".
package version

import (
	"fmt"
	"runtime/debug"
	"strings"
)

// modulePath is the import path recorded in Go build information.
const modulePath = "bennypowers.dev/asimonim"

var (
	// Version is the version string, set at build time via ldflags.
	// It may be a release tag such as "v0.3.3", git-describe output, or
	// "dev" when the binary was built without version metadata.
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

// readBuildInfo is indirected so tests can inject build information.
var readBuildInfo = debug.ReadBuildInfo

// Get returns the version string for the application.
//
// It prefers the Version injected at build time, then a usable version from
// Go build information, then Git metadata, and finally "dev". When asimonim
// is embedded as a dependency, the build information lookup returns the
// asimonim module version rather than the host program's version.
func Get() string {
	return get(Version, GitCommit, GitTag, GitDirty, readBuildInfo)
}

// get implements Get with its inputs supplied explicitly so the version
// resolution logic can be tested without build-time injection.
func get(version, gitCommit, gitTag, gitDirty string, read func() (*debug.BuildInfo, bool)) string {
	if version != "dev" {
		return version
	}

	if read != nil {
		if info, ok := read(); ok {
			if v := moduleVersion(info); v != "" {
				return v
			}
		}
	}

	if gitTag != "unknown" && gitCommit != "unknown" {
		v := gitTag
		if gitCommit != "" {
			commitSuffix := gitCommit
			if len(gitCommit) > 7 {
				commitSuffix = gitCommit[:7]
			}
			if !strings.HasSuffix(gitTag, commitSuffix) {
				v = fmt.Sprintf("%s-%s", gitTag, commitSuffix)
			}
		}
		if gitDirty == "dirty" {
			v += "-dirty"
		}
		return v
	}

	return "dev"
}

// moduleVersion returns the asimonim module version recorded in build info.
// It reports the main module version only when asimonim is the main module;
// otherwise it scans dependencies so embedders report the asimonim version,
// not the host program's version.
func moduleVersion(info *debug.BuildInfo) string {
	if info == nil {
		return ""
	}
	if info.Main.Path == modulePath {
		return usableVersion(info.Main.Version)
	}
	for _, dep := range info.Deps {
		if dep == nil || dep.Path != modulePath {
			continue
		}
		module := dep
		if dep.Replace != nil {
			module = dep.Replace
		}
		if module.Path != modulePath {
			continue
		}
		if v := usableVersion(module.Version); v != "" {
			return v
		}
	}
	return ""
}

// usableVersion filters out versions that do not identify a release.
func usableVersion(v string) string {
	if v == "" || v == "(devel)" {
		return ""
	}
	return v
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
