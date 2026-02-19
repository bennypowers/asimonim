/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package load provides a high-level API for loading design tokens.
package load

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/fs"
	"bennypowers.dev/asimonim/parser"
	"bennypowers.dev/asimonim/resolver"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/specifier"
	"bennypowers.dev/asimonim/token"
)

var (
	// ErrLocalResolution indicates that local filesystem resolution failed.
	ErrLocalResolution = errors.New("local resolution failed")

	// ErrNetworkFallback indicates that the CDN network fallback also failed.
	ErrNetworkFallback = errors.New("network fallback failed")
)

// Options configures how tokens are loaded.
type Options struct {
	// Root is the directory for local specifier resolution (required for local/npm: paths).
	Root string

	// FS is the filesystem to use. Defaults to OS filesystem if nil.
	FS fs.FileSystem

	// Prefix is the CSS variable prefix for tokens.
	// Takes precedence over config file if set.
	Prefix string

	// GroupMarkers are token names that can be both tokens and groups (draft only).
	// Takes precedence over config file if set.
	GroupMarkers []string

	// SchemaVersion overrides auto-detection from file content.
	// Takes precedence over config file if set.
	SchemaVersion schema.Version

	// Fetcher enables opt-in network fallback for npm: specifiers.
	// When set, if local resolution fails for an npm: specifier,
	// Load will attempt to fetch the content from unpkg.com.
	// Nil means no network fallback (default).
	Fetcher Fetcher

	// FetchTimeout is the maximum time to wait for a network fetch.
	// Defaults to DefaultTimeout when zero. Has no effect if Fetcher is nil.
	FetchTimeout time.Duration
}

// Load loads design tokens from a specifier with full resolution.
//
// The specifier can be:
//   - Local file path: "tokens.json" or "/path/to/tokens.json"
//   - npm package: "npm:@scope/pkg/tokens.json" (requires node_modules)
//   - jsr package: "jsr:@scope/pkg/tokens.json" (requires node_modules)
//
// When Options.Fetcher is set, npm: specifiers that fail local resolution
// will fall back to fetching from unpkg.com.
//
// The loading process:
//  1. Optionally loads config from .config/design-tokens.yaml
//  2. Applies Options values (they take precedence over config)
//  3. Resolves specifier to file content via filesystem (with optional CDN fallback)
//  4. Detects schema version (if not specified)
//  5. Parses tokens
//  6. Resolves $extends (v2025.10)
//  7. Resolves aliases
//  8. Returns *token.Map
func Load(ctx context.Context, spec string, opts Options) (*token.Map, error) {
	// Set up filesystem
	filesystem := opts.FS
	if filesystem == nil {
		filesystem = fs.NewOSFileSystem()
	}

	// Ensure root is absolute
	root := opts.Root
	if root == "" {
		root = "."
	}
	if !filepath.IsAbs(root) {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve root path: %w", err)
		}
		root = absRoot
	}

	// Load config file (optional - not an error if missing)
	cfg := config.LoadOrDefault(filesystem, root)

	// Build effective configuration (Options take precedence)
	prefix := opts.Prefix
	if prefix == "" {
		prefix = cfg.Prefix
	}

	groupMarkers := opts.GroupMarkers
	if len(groupMarkers) == 0 {
		groupMarkers = cfg.GroupMarkers
	}

	schemaVersion := opts.SchemaVersion
	if schemaVersion == schema.Unknown {
		schemaVersion = cfg.SchemaVersion()
	}

	// Resolve specifier to content
	fetchTimeout := opts.FetchTimeout
	if fetchTimeout == 0 {
		fetchTimeout = DefaultTimeout
	}
	content, err := resolveContent(ctx, spec, root, filesystem, opts.Fetcher, fetchTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve specifier %q: %w", spec, err)
	}

	// Parse tokens
	p := parser.NewJSONParser()
	tokens, err := p.Parse(content, parser.Options{
		Prefix:        prefix,
		GroupMarkers:  groupMarkers,
		SchemaVersion: schemaVersion,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse tokens: %w", err)
	}

	// Resolve $extends (for v2025.10)
	tokens, err = resolver.ResolveGroupExtensions(tokens, content)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve $extends: %w", err)
	}

	// Determine schema version for alias resolution
	resolveVersion := schemaVersion
	if resolveVersion == schema.Unknown && len(tokens) > 0 {
		resolveVersion = tokens[0].SchemaVersion
	}
	if resolveVersion == schema.Unknown {
		resolveVersion = schema.Draft
	}

	// Resolve aliases
	if err := resolver.ResolveAliases(tokens, resolveVersion); err != nil {
		return nil, fmt.Errorf("failed to resolve aliases: %w", err)
	}

	return token.NewMap(tokens, prefix), nil
}

// resolveContent resolves a specifier to file content.
// Tries local resolution first. If that fails and a Fetcher is provided,
// falls back to CDN for npm: specifiers.
func resolveContent(ctx context.Context, spec, root string, filesystem fs.FileSystem, fetcher Fetcher, fetchTimeout time.Duration) ([]byte, error) {
	// Create resolver chain
	res, err := specifier.NewDefaultResolver(filesystem, root)
	if err != nil {
		return nil, fmt.Errorf("failed to create resolver: %w", err)
	}

	// Resolve specifier to path
	resolved, err := res.Resolve(spec)
	if err != nil {
		// Local resolution failed — try CDN fallback
		return fetchFromCDN(ctx, spec, fetcher, fetchTimeout, err)
	}

	// Make local paths absolute relative to root
	path := resolved.Path
	if resolved.Kind == specifier.KindLocal && !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}

	// Read file content
	content, readErr := filesystem.ReadFile(path)
	if readErr != nil {
		// File read failed — try CDN fallback (npm: specifiers only;
		// non-npm specifiers return localErr unchanged via CDNURL check)
		localErr := fmt.Errorf("failed to read %s: %w", path, readErr)
		return fetchFromCDN(ctx, spec, fetcher, fetchTimeout, localErr)
	}

	return content, nil
}

// fetchFromCDN attempts to fetch content from CDN as a fallback.
// Returns the original localErr if no fetcher is provided or the specifier
// has no CDN URL.
func fetchFromCDN(ctx context.Context, spec string, fetcher Fetcher, fetchTimeout time.Duration, localErr error) ([]byte, error) {
	if fetcher == nil {
		return nil, localErr
	}

	cdnURL, ok := specifier.CDNURL(spec)
	if !ok {
		return nil, localErr
	}

	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	content, fetchErr := fetcher.Fetch(ctx, cdnURL)
	if fetchErr != nil {
		return nil, fmt.Errorf("%w (%w), %w: %w", ErrLocalResolution, localErr, ErrNetworkFallback, fetchErr)
	}

	return content, nil
}
