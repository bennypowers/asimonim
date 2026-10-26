/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

package specifier

// LocalResolver handles local filesystem paths (non-package specifiers).
type LocalResolver struct{}

// NewLocalResolver creates a resolver for local filesystem paths.
func NewLocalResolver() *LocalResolver {
	return &LocalResolver{}
}

// Resolve returns the path unchanged for local files.
func (r *LocalResolver) Resolve(spec string) (*ResolvedFile, error) {
	return &ResolvedFile{
		Specifier: spec,
		Path:      spec,
		Kind:      KindLocal,
	}, nil
}

// CanResolve returns true for paths that are not package specifiers.
func (r *LocalResolver) CanResolve(spec string) bool {
	return !IsPackageSpecifier(spec)
}
