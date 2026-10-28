/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

// Package fs provides filesystem abstractions for asimonim.
//
// FileSystem decouples token loading from the operating system so callers
// can supply in-memory or virtual filesystems. The interface is congruent
// with bennypowers.dev/cem/internal/platform.FileSystem and
// bennypowers.dev/mappa/fs.FileSystem, which allows those implementations
// to be used without an adapter.
//
// OSFileSystem implements FileSystem with the standard os package. Use
// NewOSFileSystem at program entry points and a test filesystem such as
// testutil.NewFixtureFS in tests.
package fs
