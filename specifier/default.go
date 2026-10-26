/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

package specifier

import asimfs "bennypowers.dev/asimonim/fs"

// NewDefaultResolver creates a resolver chain that handles npm:, jsr:, and local paths.
// The rootDir must be an absolute path - this is required for compatibility
// with virtual/in-memory filesystems that don't have a working directory concept.
func NewDefaultResolver(fs asimfs.FileSystem, rootDir string) (Resolver, error) {
	npmResolver, err := NewNodeModulesResolver(fs, rootDir)
	if err != nil {
		return nil, err
	}
	jsrResolver, err := NewJSRNodeModulesResolver(fs, rootDir)
	if err != nil {
		return nil, err
	}
	return NewChainResolver(
		npmResolver,
		jsrResolver,
		NewLocalResolver(),
	), nil
}
