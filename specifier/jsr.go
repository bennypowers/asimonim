/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package specifier

import (
	"fmt"
	"strings"
)

// JSRResolver is a stub resolver for jsr: specifiers.
// JSR (JavaScript Registry) resolution is not yet implemented.
type JSRResolver struct{}

// NewJSRResolver creates a resolver for jsr: package specifiers.
func NewJSRResolver() *JSRResolver {
	return &JSRResolver{}
}

// Resolve returns an error as JSR resolution is not yet implemented.
func (r *JSRResolver) Resolve(spec string) (*ResolvedFile, error) {
	parsed := Parse(spec)
	if parsed.Kind != KindJSR {
		return nil, fmt.Errorf("not a jsr specifier: %s", spec)
	}
	return nil, fmt.Errorf("jsr: specifier resolution not implemented: %s", spec)
}

// CanResolve returns true for jsr: specifiers.
func (r *JSRResolver) CanResolve(spec string) bool {
	return strings.HasPrefix(spec, "jsr:")
}
