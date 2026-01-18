/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package version provides version information for the asimonim CLI.
package version

import (
	"fmt"
	"runtime/debug"
	"strings"
)

var (
	// Version information, set at build time via ldflags
	Version   = "dev"
	GitCommit = "unknown"
	GitTag    = "unknown"
	BuildTime = "unknown"
	GitDirty  = ""
)

// Get returns the version string for the application.
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
func Info() map[string]string {
	return map[string]string{
		"version":   Get(),
		"gitCommit": GitCommit,
		"gitTag":    GitTag,
		"buildTime": BuildTime,
		"gitDirty":  GitDirty,
	}
}
