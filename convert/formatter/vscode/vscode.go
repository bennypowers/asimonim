/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package vscode provides VSCode snippets formatting for design tokens.
package vscode

import (
	"encoding/json"
	"strings"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/parser/common"
	"bennypowers.dev/asimonim/token"
)

// Snippet represents a VSCode snippet entry.
type Snippet struct {
	Scope       string   `json:"scope"`
	Prefix      []string `json:"prefix"`
	Body        []string `json:"body"`
	Description string   `json:"description,omitempty"`
}

// Formatter outputs VSCode snippets JSON.
type Formatter struct{}

// New creates a new VSCode snippets formatter.
func New() *Formatter {
	return &Formatter{}
}

// Format converts tokens to VSCode snippets JSON format.
func (f *Formatter) Format(tokens []*token.Token, opts formatter.Options) ([]byte, error) {
	snippets := make(map[string]Snippet)

	sorted := formatter.SortTokens(tokens)

	for _, tok := range sorted {
		name := formatter.ToKebabCase(strings.Join(tok.Path, "-"))
		if opts.Prefix != "" {
			name = opts.Prefix + "-" + name
		}

		snippet := buildSnippet(tok, name, opts)
		snippets[name] = snippet
	}

	return json.MarshalIndent(snippets, "", "  ")
}

// buildSnippet creates a VSCode snippet from a token.
func buildSnippet(tok *token.Token, name string, _ formatter.Options) Snippet {
	// Build prefixes: token name, camelCase version, and value for colors
	prefixes := buildPrefixes(tok, name)

	// CSS variable reference
	cssVar := "var(--" + name + ")"

	snippet := Snippet{
		Scope:  "css,scss,less,stylus,postcss",
		Prefix: prefixes,
		Body:   []string{cssVar},
	}

	if tok.Description != "" {
		snippet.Description = tok.Description
	}

	return snippet
}

// buildPrefixes generates the prefix array for autocomplete.
func buildPrefixes(tok *token.Token, name string) []string {
	prefixes := []string{name}

	// Add camelCase version
	camelName := formatter.ToCamelCase(name)
	if camelName != name {
		prefixes = append(prefixes, camelName)
	}

	// Add underscore-separated version for fuzzy matching
	underscoreName := strings.ReplaceAll(name, "-", "_")
	if underscoreName != name {
		prefixes = append(prefixes, underscoreName)
	}

	// For color tokens, add the hex value as a prefix for color picker matching
	if tok.Type == token.TypeColor {
		if hexValue := extractColorPrefix(tok); hexValue != "" {
			prefixes = append(prefixes, hexValue)
		}
	}

	return prefixes
}

// extractColorPrefix extracts a color value suitable for use as a snippet prefix.
func extractColorPrefix(tok *token.Token) string {
	value := formatter.ResolvedValue(tok)

	// Try to parse as structured color
	if colorVal, err := common.ParseColorValue(value, tok.SchemaVersion); err == nil {
		if objColor, ok := colorVal.(*common.ObjectColorValue); ok {
			if objColor.Hex != nil && *objColor.Hex != "" {
				// Remove # prefix for cleaner autocomplete
				return strings.TrimPrefix(*objColor.Hex, "#")
			}
		}
		// For string colors, use the value directly
		if strColor, ok := colorVal.(*common.StringColorValue); ok {
			v := strColor.Value
			// Only use hex-like values as prefixes
			if strings.HasPrefix(v, "#") {
				return strings.TrimPrefix(v, "#")
			}
		}
	}

	// Fallback: check if string value looks like a hex color
	if s, ok := value.(string); ok {
		if hex, found := strings.CutPrefix(s, "#"); found {
			return hex
		}
	}

	return ""
}
