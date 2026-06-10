/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package load provides a high-level API for loading design tokens.
package load

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
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

	// Fetcher enables opt-in network fallback for package specifiers.
	// When set, if local resolution fails for an npm: or jsr: specifier,
	// Load will attempt to fetch the content from a CDN.
	// Nil means no network fallback (default).
	Fetcher Fetcher

	// CDN selects the CDN provider for network fallback.
	// Takes precedence over config file if set.
	// Defaults to "unpkg" when empty. Only "esm.sh" supports jsr: specifiers.
	CDN specifier.CDN

	// FetchTimeout is the maximum time to wait for a network fetch.
	// Defaults to DefaultTimeout when zero. Has no effect if Fetcher is nil.
	FetchTimeout time.Duration
}

// resolvedContent holds content bytes along with provenance for resolving
// relative paths in resolver documents.
type resolvedContent struct {
	Data    []byte
	BaseDir string // local filesystem directory, or ""
	BaseURL string // CDN base URL, or ""
}

// isResolverDocument checks if JSON data represents a DTCG resolver document
// by looking for the "resolutionOrder" field at the root.
func isResolverDocument(data []byte) bool {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return false
	}
	_, hasResolutionOrder := doc["resolutionOrder"]
	return hasResolutionOrder
}

// Load loads design tokens from a specifier with full resolution.
//
// The specifier can be:
//   - Local file path: "tokens.json" or "/path/to/tokens.json"
//   - npm package: "npm:@scope/pkg/tokens.json" (requires node_modules)
//   - jsr package: "jsr:@scope/pkg/tokens.json" (requires node_modules)
//
// When Options.Fetcher is set, npm: and jsr: specifiers that fail local
// resolution will fall back to fetching from a CDN (configurable via Options.CDN).
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

	// Resolve effective CDN (Options take precedence)
	var cdn specifier.CDN
	if opts.CDN != "" {
		parsed, err := specifier.ParseCDN(string(opts.CDN))
		if err != nil {
			return nil, fmt.Errorf("invalid cdn in options: %w", err)
		}
		cdn = parsed
	} else if cfg.CDN != "" {
		parsed, err := specifier.ParseCDN(cfg.CDN)
		if err != nil {
			return nil, fmt.Errorf("invalid cdn in config: %w", err)
		}
		cdn = parsed
	}

	// Resolve specifier to content
	fetchTimeout := opts.FetchTimeout
	if fetchTimeout == 0 {
		fetchTimeout = DefaultTimeout
	}
	rc, err := resolveContent(ctx, spec, root, filesystem, opts.Fetcher, fetchTimeout, cdn)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve specifier %q: %w", spec, err)
	}

	parseOpts := parser.Options{
		Prefix:        prefix,
		GroupMarkers:  groupMarkers,
		SchemaVersion: schemaVersion,
	}

	var tokens []*token.Token
	if isResolverDocument(rc.Data) {
		tokens, err = loadResolver(ctx, rc, filesystem, opts.Fetcher, fetchTimeout, cdn, parseOpts)
	} else {
		tokens, err = loadTokenFile(rc.Data, parseOpts)
	}
	if err != nil {
		return nil, err
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

// SourceTokens holds tokens grouped by their originating source.
type SourceTokens struct {
	Source string
	Path   string
	Tokens []*token.Token
}

// Result holds the result of loading multiple token sources.
type Result struct {
	Sources []SourceTokens
	All     []*token.Token
	Version schema.Version
}

// LoadAll loads tokens from multiple sources with cross-file alias resolution.
// If files is non-empty, each entry is resolved as a specifier. Otherwise,
// files and resolver sources are discovered from cfg.
func LoadAll(ctx context.Context, cfg *config.Config, files []string, opts Options) (*Result, error) {
	filesystem := opts.FS
	if filesystem == nil {
		filesystem = fs.NewOSFileSystem()
	}

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

	specResolver, err := specifier.NewDefaultResolver(filesystem, root)
	if err != nil {
		return nil, fmt.Errorf("failed to create resolver: %w", err)
	}

	var resolvedFiles []*specifier.ResolvedFile
	if len(files) > 0 {
		for _, file := range files {
			rf, err := specResolver.Resolve(file)
			if err != nil {
				return nil, fmt.Errorf("error resolving %s: %w", file, err)
			}
			resolvedFiles = append(resolvedFiles, rf)
		}
	} else {
		resolvedFiles, err = cfg.ResolveFiles(specResolver, filesystem, root)
		if err != nil {
			return nil, fmt.Errorf("error resolving config files: %w", err)
		}
		if len(cfg.Resolvers) > 0 {
			resolverSources, err := cfg.ResolveResolverSources(specResolver, filesystem, root)
			if err != nil {
				return nil, fmt.Errorf("error resolving resolver sources: %w", err)
			}
			resolvedFiles = specifier.DedupResolvedFiles(append(resolvedFiles, resolverSources...))
		}
	}

	if len(resolvedFiles) == 0 {
		return nil, fmt.Errorf("no files specified and no files found in config")
	}

	schemaVersion := opts.SchemaVersion
	if schemaVersion == schema.Unknown {
		schemaVersion = cfg.SchemaVersion()
	}

	jsonParser := parser.NewJSONParser()
	result := &Result{}
	var allTokens []*token.Token

	for _, rf := range resolvedFiles {
		path := rf.Path
		if rf.Kind == specifier.KindLocal && !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		data, err := filesystem.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("error reading %s: %w", rf.Specifier, err)
		}

		version := schemaVersion
		if version == schema.Unknown {
			version, err = schema.DetectVersion(data, nil)
			if err != nil {
				return nil, fmt.Errorf("error detecting schema for %s: %w", rf.Specifier, err)
			}
		}
		if result.Version == schema.Unknown {
			result.Version = version
		}

		parseOpts := cfg.OptionsForFile(rf.Specifier)
		parseOpts.SkipPositions = true
		if opts.Prefix != "" {
			parseOpts.Prefix = opts.Prefix
		}
		if len(opts.GroupMarkers) > 0 {
			parseOpts.GroupMarkers = opts.GroupMarkers
		}
		if version != schema.Unknown {
			parseOpts.SchemaVersion = version
		}

		tokens, err := jsonParser.ParseFile(filesystem, path, parseOpts)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %w", rf.Specifier, err)
		}

		result.Sources = append(result.Sources, SourceTokens{
			Source: rf.Specifier,
			Path:   path,
			Tokens: tokens,
		})
		allTokens = append(allTokens, tokens...)
	}

	if result.Version == schema.Unknown {
		result.Version = schema.Draft
	}
	if err := resolver.ResolveAliases(allTokens, result.Version); err != nil {
		return nil, fmt.Errorf("error resolving aliases: %w", err)
	}

	result.All = allTokens
	return result, nil
}

// loadTokenFile parses a single token file.
func loadTokenFile(data []byte, opts parser.Options) ([]*token.Token, error) {
	p := parser.NewJSONParser()
	tokens, err := p.Parse(data, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tokens: %w", err)
	}
	tokens, err = resolver.ResolveGroupExtensions(tokens, data)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve $extends: %w", err)
	}
	return tokens, nil
}

// loadResolver expands a resolver document by loading all its source files.
func loadResolver(
	ctx context.Context,
	rc resolvedContent,
	filesystem fs.FileSystem,
	fetcher Fetcher,
	fetchTimeout time.Duration,
	cdn specifier.CDN,
	parseOpts parser.Options,
) ([]*token.Token, error) {
	baseDir := rc.BaseDir
	sourcePaths, err := config.ExtractSourcePaths(rc.Data, baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to extract sources from resolver: %w", err)
	}

	var allTokens []*token.Token
	for _, srcPath := range sourcePaths {
		data, err := resolveSourceContent(ctx, srcPath, baseDir, rc.BaseURL, filesystem, fetcher, fetchTimeout, cdn)
		if err != nil {
			return nil, fmt.Errorf("failed to load resolver source %s: %w", srcPath, err)
		}
		tokens, err := loadTokenFile(data, parseOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to parse resolver source %s: %w", srcPath, err)
		}
		allTokens = append(allTokens, tokens...)
	}

	return allTokens, nil
}

// resolveSourceContent reads a resolver source file from local filesystem or CDN.
func resolveSourceContent(
	ctx context.Context,
	srcPath, baseDir, baseURL string,
	filesystem fs.FileSystem,
	fetcher Fetcher,
	fetchTimeout time.Duration,
	cdn specifier.CDN,
) ([]byte, error) {
	// Package specifiers resolve via the specifier chain / CDN
	if specifier.IsPackageSpecifier(srcPath) {
		rc, err := resolveContent(ctx, srcPath, baseDir, filesystem, fetcher, fetchTimeout, cdn)
		if err != nil {
			return nil, err
		}
		return rc.Data, nil
	}

	// Local path: read from filesystem
	if baseDir != "" {
		absPath := srcPath
		if !filepath.IsAbs(srcPath) {
			absPath = filepath.Clean(filepath.Join(baseDir, srcPath))
		}
		// Reject paths that escape the base directory
		rel, relErr := filepath.Rel(baseDir, absPath)
		if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("source path escapes base directory: %s", srcPath)
		}
		data, err := filesystem.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read source %s: %w", srcPath, err)
		}
		return data, nil
	}

	// CDN fallback: resolve $ref path against base URL
	if baseURL != "" && fetcher != nil {
		base, err := url.Parse(baseURL)
		if err == nil {
			ref, err := url.Parse(srcPath)
			if err == nil {
				resolved := base.ResolveReference(ref).String()
				ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
				defer cancel()
				return fetcher.Fetch(ctx, resolved)
			}
		}
	}

	return nil, fmt.Errorf("cannot resolve source %s (no baseDir or baseURL)", srcPath)
}

// resolveContent resolves a specifier to file content with provenance.
func resolveContent(ctx context.Context, spec, root string, filesystem fs.FileSystem, fetcher Fetcher, fetchTimeout time.Duration, cdn specifier.CDN) (resolvedContent, error) {
	res, err := specifier.NewDefaultResolver(filesystem, root)
	if err != nil {
		return resolvedContent{}, fmt.Errorf("failed to create resolver: %w", err)
	}

	resolved, err := res.Resolve(spec)
	if err != nil {
		data, cdnErr := fetchFromCDN(ctx, spec, fetcher, fetchTimeout, cdn, err)
		if cdnErr != nil {
			return resolvedContent{}, cdnErr
		}
		return resolvedContent{Data: data, BaseURL: cdnBaseURL(spec, cdn)}, nil
	}

	path := resolved.Path
	if resolved.Kind == specifier.KindLocal && !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}

	content, readErr := filesystem.ReadFile(path)
	if readErr != nil {
		localErr := fmt.Errorf("failed to read %s: %w", path, readErr)
		data, cdnErr := fetchFromCDN(ctx, spec, fetcher, fetchTimeout, cdn, localErr)
		if cdnErr != nil {
			return resolvedContent{}, cdnErr
		}
		return resolvedContent{Data: data, BaseURL: cdnBaseURL(spec, cdn)}, nil
	}

	return resolvedContent{Data: content, BaseDir: filepath.Dir(path)}, nil
}

// cdnBaseURL computes the base URL directory for a CDN-fetched specifier.
func cdnBaseURL(spec string, cdn specifier.CDN) string {
	cdnURL, _ := specifier.CDNURL(spec, cdn)
	if idx := strings.LastIndexByte(cdnURL, '/'); idx >= 0 {
		return cdnURL[:idx+1]
	}
	return cdnURL
}

// fetchFromCDN attempts to fetch content from CDN as a fallback.
// Returns the original localErr if no fetcher is provided or the specifier
// has no CDN URL for the given CDN provider.
func fetchFromCDN(ctx context.Context, spec string, fetcher Fetcher, fetchTimeout time.Duration, cdn specifier.CDN, localErr error) ([]byte, error) {
	if fetcher == nil {
		return nil, localErr
	}

	cdnURL, ok := specifier.CDNURL(spec, cdn)
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
