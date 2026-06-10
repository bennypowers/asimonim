/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package mcp provides an MCP server for design tokens.
package mcp

import (
	"context"
	"strings"

	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/fs"
	"bennypowers.dev/asimonim/load"
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
// Delegates to load.LoadAll for orchestration.
func parseWorkspaceTokens(
	filesystem fs.FileSystem,
	cfg *config.Config,
	files []string,
	cwd string,
) (*parseResult, error) {
	lr, err := load.LoadAll(context.Background(), cfg, files, load.Options{
		Root: cwd,
		FS:   filesystem,
	})
	if err != nil {
		return nil, err
	}

	result := &parseResult{
		AllTokens: lr.All,
		Version:   lr.Version,
	}
	for _, src := range lr.Sources {
		result.Sources = append(result.Sources, sourceTokens{
			Source: sourceLabel(&specifier.ResolvedFile{Specifier: src.Source}),
			Tokens: src.Tokens,
		})
	}
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
