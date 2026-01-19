/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package specifier

import "fmt"

// ResolvedFile preserves both the original specifier and the resolved filesystem path.
type ResolvedFile struct {
	// Specifier is the original specifier (e.g., "npm:@rhds/tokens/tokens.json").
	Specifier string

	// Path is the resolved filesystem path (e.g., "/project/node_modules/@rhds/tokens/tokens.json").
	Path string

	// Kind indicates the type of specifier (KindNPM, KindJSR, KindLocal).
	Kind Kind
}

// Resolver resolves specifiers to filesystem paths.
type Resolver interface {
	// Resolve resolves a specifier to a ResolvedFile.
	// Returns an error if resolution fails.
	Resolve(spec string) (*ResolvedFile, error)

	// CanResolve returns true if this resolver can handle the given specifier.
	CanResolve(spec string) bool
}

// ChainResolver tries multiple resolvers in order.
type ChainResolver struct {
	resolvers []Resolver
}

// NewChainResolver creates a resolver that tries each resolver in order.
func NewChainResolver(resolvers ...Resolver) *ChainResolver {
	return &ChainResolver{resolvers: resolvers}
}

// Resolve tries each resolver in order until one succeeds.
func (c *ChainResolver) Resolve(spec string) (*ResolvedFile, error) {
	for _, r := range c.resolvers {
		if r.CanResolve(spec) {
			return r.Resolve(spec)
		}
	}
	return nil, fmt.Errorf("no resolver found for specifier: %s", spec)
}

// CanResolve returns true if any resolver can handle the specifier.
func (c *ChainResolver) CanResolve(spec string) bool {
	for _, r := range c.resolvers {
		if r.CanResolve(spec) {
			return true
		}
	}
	return false
}
