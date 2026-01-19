/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package specifier

import asimfs "bennypowers.dev/asimonim/fs"

// NewDefaultResolver creates a resolver chain that handles npm:, jsr:, and local paths.
// The rootDir is the starting directory for node_modules lookup.
func NewDefaultResolver(fs asimfs.FileSystem, rootDir string) Resolver {
	return NewChainResolver(
		NewNPMResolver(fs, rootDir),
		NewJSRResolver(),
		NewLocalResolver(),
	)
}
