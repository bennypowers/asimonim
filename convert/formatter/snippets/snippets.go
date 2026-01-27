/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package snippets provides editor snippet formatting for design tokens.
package snippets

import (
	"encoding/json"
	"strings"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/parser/common"
	"bennypowers.dev/asimonim/token"
)

// Type represents the snippet output format.
type Type string

const (
	// TypeVSCode outputs VSCode/JSON snippets format.
	TypeVSCode Type = "vscode"

	// TypeTextMate outputs TextMate/plist snippets format.
	TypeTextMate Type = "textmate"

	// TypeZed outputs Zed editor snippets format.
	TypeZed Type = "zed"
)

// Options configures the snippets formatter.
type Options struct {
	formatter.Options

	// Type specifies the snippet output format.
	// Defaults to TypeVSCode.
	Type Type
}

// Snippet represents a VSCode snippet entry.
type Snippet struct {
	Scope       string   `json:"scope"`
	Prefix      []string `json:"prefix"`
	Body        []string `json:"body"`
	Description string   `json:"description,omitempty"`
}

// ZedSnippet represents a Zed editor snippet entry.
// Zed uses a single prefix string (only first recognized) and no scope field.
type ZedSnippet struct {
	Prefix      string   `json:"prefix,omitempty"`
	Body        []string `json:"body"`
	Description string   `json:"description,omitempty"`
}

// Formatter outputs editor snippets.
type Formatter struct {
	opts Options
}

// New creates a new snippets formatter with default options.
func New() *Formatter {
	return &Formatter{opts: Options{Type: TypeVSCode}}
}

// NewWithOptions creates a new snippets formatter with the given options.
func NewWithOptions(opts Options) *Formatter {
	if opts.Type == "" {
		opts.Type = TypeVSCode
	}
	return &Formatter{opts: opts}
}

// Format converts tokens to editor snippets format.
func (f *Formatter) Format(tokens []*token.Token, opts formatter.Options) ([]byte, error) {
	switch f.opts.Type {
	case TypeTextMate:
		return f.formatTextMate(tokens, opts)
	case TypeZed:
		return f.formatZed(tokens, opts)
	default:
		return f.formatVSCode(tokens, opts)
	}
}

// formatVSCode outputs VSCode JSON snippets format.
func (f *Formatter) formatVSCode(tokens []*token.Token, opts formatter.Options) ([]byte, error) {
	snippetMap := make(map[string]Snippet)

	sorted := formatter.SortTokens(tokens)

	// Build token index for light-dark detection
	tokenIndex := buildTokenIndex(sorted, opts.Prefix)

	for _, tok := range sorted {
		name := formatter.ToKebabCase(strings.Join(tok.Path, "-"))
		if opts.Prefix != "" {
			name = opts.Prefix + "-" + name
		}

		// Check if this token is part of a light-dark group
		if group := findLightDarkGroup(tok, tokenIndex); group != nil {
			// Only emit the combined snippet for the root token
			if isRootToken(tok, group) {
				snippet := buildLightDarkSnippet(group, name, opts)
				snippetMap[name] = snippet
			}
			// Skip individual snippets for light/dark children
			continue
		}

		snippet := buildSnippet(tok, name, opts)
		snippetMap[name] = snippet
	}

	return json.MarshalIndent(snippetMap, "", "  ")
}

// formatTextMate outputs TextMate plist snippets format.
func (f *Formatter) formatTextMate(tokens []*token.Token, opts formatter.Options) ([]byte, error) {
	// TextMate format uses XML plist - for now, return a simple implementation
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<array>
`)

	sorted := formatter.SortTokens(tokens)

	for _, tok := range sorted {
		name := formatter.ToKebabCase(strings.Join(tok.Path, "-"))
		if opts.Prefix != "" {
			name = opts.Prefix + "-" + name
		}

		cssVar := "var(--" + name + ")"

		sb.WriteString("  <dict>\n")
		sb.WriteString("    <key>name</key>\n")
		sb.WriteString("    <string>" + name + "</string>\n")
		sb.WriteString("    <key>tabTrigger</key>\n")
		sb.WriteString("    <string>" + name + "</string>\n")
		sb.WriteString("    <key>content</key>\n")
		sb.WriteString("    <string>" + cssVar + "</string>\n")
		sb.WriteString("    <key>scope</key>\n")
		sb.WriteString("    <string>source.css, source.scss</string>\n")
		sb.WriteString("  </dict>\n")
	}

	sb.WriteString("</array>\n</plist>\n")

	return []byte(sb.String()), nil
}

// formatZed outputs Zed editor JSON snippets format.
// Zed snippets use a single prefix string and no scope field.
func (f *Formatter) formatZed(tokens []*token.Token, opts formatter.Options) ([]byte, error) {
	snippetMap := make(map[string]ZedSnippet)

	sorted := formatter.SortTokens(tokens)

	// Build token index for light-dark detection
	tokenIndex := buildTokenIndex(sorted, opts.Prefix)

	for _, tok := range sorted {
		name := formatter.ToKebabCase(strings.Join(tok.Path, "-"))
		if opts.Prefix != "" {
			name = opts.Prefix + "-" + name
		}

		// Check if this token is part of a light-dark group
		if group := findLightDarkGroup(tok, tokenIndex); group != nil {
			// Only emit the combined snippet for the root token
			if isRootToken(tok, group) {
				snippet := buildZedLightDarkSnippet(group, name, opts)
				snippetMap[name] = snippet
			}
			// Skip individual snippets for light/dark children
			continue
		}

		snippet := buildZedSnippet(tok, name, opts)
		snippetMap[name] = snippet
	}

	return json.MarshalIndent(snippetMap, "", "  ")
}

// buildZedSnippet creates a Zed editor snippet from a token.
func buildZedSnippet(tok *token.Token, name string, _ formatter.Options) ZedSnippet {
	// CSS variable reference
	cssVar := "var(--" + name + ")"

	snippet := ZedSnippet{
		Prefix: name,
		Body:   []string{cssVar},
	}

	if tok.Description != "" {
		snippet.Description = tok.Description
	}

	return snippet
}

// buildZedLightDarkSnippet creates a Zed snippet with light-dark() pattern.
func buildZedLightDarkSnippet(group *lightDarkGroup, name string, opts formatter.Options) ZedSnippet {
	lightName := formatter.ToKebabCase(strings.Join(group.Light.Path, "-"))
	darkName := formatter.ToKebabCase(strings.Join(group.Dark.Path, "-"))
	if opts.Prefix != "" {
		lightName = opts.Prefix + "-" + lightName
		darkName = opts.Prefix + "-" + darkName
	}

	// Get resolved color values for fallbacks
	lightValue := getColorValue(group.Light)
	darkValue := getColorValue(group.Dark)

	// Build the light-dark() pattern with fallbacks
	var body string
	if lightValue != "" && darkValue != "" {
		body = "var(--" + name + ", light-dark(\n  var(--" + lightName + ", " + lightValue + "),\n  var(--" + darkName + ", " + darkValue + ")\n))"
	} else {
		body = "var(--" + name + ", light-dark(\n  var(--" + lightName + "),\n  var(--" + darkName + ")\n))"
	}

	snippet := ZedSnippet{
		Prefix: name,
		Body:   []string{body},
	}

	if group.Root.Description != "" {
		snippet.Description = group.Root.Description
	}

	return snippet
}

// lightDarkGroup represents a detected light-dark token group.
type lightDarkGroup struct {
	Root  *token.Token
	Light *token.Token
	Dark  *token.Token
}

// tokenIndexEntry holds a token and its computed name.
type tokenIndexEntry struct {
	Token *token.Token
	Name  string
}

// buildTokenIndex creates a map from token path to token for efficient lookup.
func buildTokenIndex(tokens []*token.Token, prefix string) map[string]*tokenIndexEntry {
	index := make(map[string]*tokenIndexEntry, len(tokens))
	for _, tok := range tokens {
		path := strings.Join(tok.Path, ".")
		name := formatter.ToKebabCase(strings.Join(tok.Path, "-"))
		if prefix != "" {
			name = prefix + "-" + name
		}
		index[path] = &tokenIndexEntry{Token: tok, Name: name}
	}
	return index
}

// findLightDarkGroup checks if a token is part of a light-dark group.
// Returns the group if found, nil otherwise.
//
// Detection rules:
// - A color token with a Reference field that points to a ".light" child
// - Must have both ".light" and ".dark" children
func findLightDarkGroup(tok *token.Token, index map[string]*tokenIndexEntry) *lightDarkGroup {
	if tok.Type != token.TypeColor {
		return nil
	}

	tokPath := strings.Join(tok.Path, ".")

	// Check if this token IS the root (has Reference pointing to light child)
	if tok.Reference != "" {
		lightPath := tokPath + ".light"
		darkPath := tokPath + ".dark"

		expectedRef := "{" + lightPath + "}"
		if tok.Reference == expectedRef {
			lightEntry, hasLight := index[lightPath]
			darkEntry, hasDark := index[darkPath]

			if hasLight && hasDark {
				return &lightDarkGroup{
					Root:  tok,
					Light: lightEntry.Token,
					Dark:  darkEntry.Token,
				}
			}
		}
	}

	// Check if this token is a light/dark child
	if len(tok.Path) < 2 {
		return nil
	}

	lastSegment := tok.Path[len(tok.Path)-1]
	if lastSegment != "light" && lastSegment != "dark" {
		return nil
	}

	parentPath := strings.Join(tok.Path[:len(tok.Path)-1], ".")
	rootEntry, hasRoot := index[parentPath]
	if !hasRoot || rootEntry.Token.Type != token.TypeColor {
		return nil
	}

	// Verify root references the light variant
	lightPath := parentPath + ".light"
	darkPath := parentPath + ".dark"
	expectedRef := "{" + lightPath + "}"

	if rootEntry.Token.Reference != expectedRef {
		return nil
	}

	lightEntry, hasLight := index[lightPath]
	darkEntry, hasDark := index[darkPath]

	if !hasLight || !hasDark {
		return nil
	}

	return &lightDarkGroup{
		Root:  rootEntry.Token,
		Light: lightEntry.Token,
		Dark:  darkEntry.Token,
	}
}

// isRootToken checks if the given token is the root of the light-dark group.
func isRootToken(tok *token.Token, group *lightDarkGroup) bool {
	return tok == group.Root
}

// buildLightDarkSnippet creates a snippet with light-dark() pattern.
func buildLightDarkSnippet(group *lightDarkGroup, name string, opts formatter.Options) Snippet {
	lightName := formatter.ToKebabCase(strings.Join(group.Light.Path, "-"))
	darkName := formatter.ToKebabCase(strings.Join(group.Dark.Path, "-"))
	if opts.Prefix != "" {
		lightName = opts.Prefix + "-" + lightName
		darkName = opts.Prefix + "-" + darkName
	}

	// Get resolved color values for fallbacks
	lightValue := getColorValue(group.Light)
	darkValue := getColorValue(group.Dark)

	// Build the light-dark() pattern with fallbacks
	var body string
	if lightValue != "" && darkValue != "" {
		body = "var(--" + name + ", light-dark(\n  var(--" + lightName + ", " + lightValue + "),\n  var(--" + darkName + ", " + darkValue + ")\n))"
	} else {
		body = "var(--" + name + ", light-dark(\n  var(--" + lightName + "),\n  var(--" + darkName + ")\n))"
	}

	prefixes := buildPrefixes(group.Root, name)

	snippet := Snippet{
		Scope:  "css,scss,less,stylus,postcss",
		Prefix: prefixes,
		Body:   []string{body},
	}

	if group.Root.Description != "" {
		snippet.Description = group.Root.Description
	}

	return snippet
}

// getColorValue extracts a CSS color value from a token.
func getColorValue(tok *token.Token) string {
	value := formatter.ResolvedValue(tok)

	if colorVal, err := common.ParseColorValue(value, tok.SchemaVersion); err == nil {
		return colorVal.ToCSS()
	}

	if s, ok := value.(string); ok {
		return s
	}

	return ""
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
			// Only use hex-like values as prefixes
			if hex, found := strings.CutPrefix(strColor.Value, "#"); found {
				return hex
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
