/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
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
