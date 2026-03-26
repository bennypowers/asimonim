/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package mcp provides an MCP server for design tokens.
package mcp

import (
	"fmt"
	"strings"

	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/fs"
	"bennypowers.dev/asimonim/parser"
	"bennypowers.dev/asimonim/resolver"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/specifier"
	"bennypowers.dev/asimonim/token"
)

// sourceTokens holds tokens grouped by their originating source.
type sourceTokens struct {
	// Source is the file path or package specifier (e.g., "tokens.json", "@rhds/tokens").
	Source string
	// Tokens are the parsed tokens from this source.
	Tokens []*token.Token
}

// parseResult holds the result of parsing workspace tokens.
type parseResult struct {
	// Sources are tokens grouped by originating file/package.
	Sources []sourceTokens
	// AllTokens is the flattened list of all tokens (alias-resolved).
	AllTokens []*token.Token
	// Version is the detected schema version.
	Version schema.Version
}

// parseWorkspaceTokens discovers and parses all token files from config or explicit paths.
// It resolves aliases across all parsed tokens.
func parseWorkspaceTokens(
	filesystem fs.FileSystem,
	cfg *config.Config,
	files []string,
	cwd string,
) (*parseResult, error) {
	specResolver, err := specifier.NewDefaultResolver(filesystem, cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to create resolver: %w", err)
	}

	jsonParser := parser.NewJSONParser()

	var resolvedFiles []*specifier.ResolvedFile
	if len(files) == 0 {
		resolvedFiles, err = cfg.ResolveFiles(specResolver, filesystem, ".")
		if err != nil {
			return nil, fmt.Errorf("error resolving config files: %w", err)
		}

		if len(cfg.Resolvers) > 0 {
			resolverSources, err := cfg.ResolveResolverSources(specResolver, filesystem, cwd)
			if err != nil {
				return nil, fmt.Errorf("error resolving resolver sources: %w", err)
			}
			resolvedFiles = specifier.DedupResolvedFiles(append(resolvedFiles, resolverSources...))
		}
	} else {
		for _, file := range files {
			rf, err := specResolver.Resolve(file)
			if err != nil {
				return nil, fmt.Errorf("error resolving %s: %w", file, err)
			}
			resolvedFiles = append(resolvedFiles, rf)
		}
	}

	if len(resolvedFiles) == 0 {
		return nil, fmt.Errorf("no files specified and no files found in config")
	}

	var schemaVersion schema.Version
	if cfg.SchemaVersion() != schema.Unknown {
		schemaVersion = cfg.SchemaVersion()
	}

	result := &parseResult{}
	var allTokens []*token.Token

	for _, rf := range resolvedFiles {
		data, err := filesystem.ReadFile(rf.Path)
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

		opts := cfg.OptionsForFile(rf.Specifier)
		opts.SkipPositions = true
		if version != schema.Unknown {
			opts.SchemaVersion = version
		}

		tokens, err := jsonParser.ParseFile(filesystem, rf.Path, opts)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %w", rf.Specifier, err)
		}

		source := sourceLabel(rf)
		result.Sources = append(result.Sources, sourceTokens{
			Source: source,
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

	result.AllTokens = allTokens
	return result, nil
}

// sourceLabel returns a human-readable label for a resolved file.
// For npm: specifiers, it extracts the package name.
// For local files, it returns the specifier as-is.
func sourceLabel(rf *specifier.ResolvedFile) string {
	spec := rf.Specifier
	if strings.HasPrefix(spec, "npm:") {
		// Extract package name: npm:@scope/pkg/path -> @scope/pkg
		// or npm:pkg/path -> pkg
		trimmed := strings.TrimPrefix(spec, "npm:")
		if strings.HasPrefix(trimmed, "@") {
			// Scoped: @scope/pkg/rest...
			parts := strings.SplitN(trimmed, "/", 3)
			if len(parts) >= 2 {
				return parts[0] + "/" + parts[1]
			}
		} else {
			// Unscoped: pkg/rest...
			parts := strings.SplitN(trimmed, "/", 2)
			return parts[0]
		}
	}
	if strings.HasPrefix(spec, "jsr:") {
		trimmed := strings.TrimPrefix(spec, "jsr:")
		parts := strings.SplitN(trimmed, "/", 3)
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	}
	return spec
}

// filterTokens filters tokens by type and group prefix.
func filterTokens(tokens []*token.Token, tokenType, group string) []*token.Token {
	if tokenType == "" && group == "" {
		return tokens
	}
	result := make([]*token.Token, 0, len(tokens))
	for _, tok := range tokens {
		if tokenType != "" && tok.Type != tokenType {
			continue
		}
		if group != "" && !strings.HasPrefix(tok.DotPath(), group) {
			continue
		}
		result = append(result, tok)
	}
	return result
}
