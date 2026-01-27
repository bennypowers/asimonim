/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package snippets provides editor snippet formatting for design tokens.
package snippets

import (
	"encoding/json"
	"fmt"
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
		name := buildTokenName(tok.Path, opts.Prefix)

		// Check if this token is part of a light-dark group
		if group := findLightDarkGroup(tok, tokenIndex); group != nil {
			// Only emit the combined snippet for the root token
			if isRootToken(tok, group) {
				rootName := getRootName(group, opts.Prefix)
				snippet := buildLightDarkSnippet(group, rootName, opts)
				snippetMap[rootName] = snippet
			}
			// Skip individual snippets for light/dark children
			continue
		}

		snippet := buildSnippet(tok, name, opts)
		snippetMap[name] = snippet
	}

	return json.MarshalIndent(snippetMap, "", "  ")
}

const textMatePlistHeader = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<array>
`

const textMateSnippetTemplate = `  <dict>
    <key>name</key>
    <string>%s</string>
    <key>tabTrigger</key>
    <string>%s</string>
    <key>content</key>
    <string>%s</string>
    <key>scope</key>
    <string>source.css, source.scss</string>
  </dict>
`

// formatTextMate outputs TextMate plist snippets format.
func (f *Formatter) formatTextMate(tokens []*token.Token, opts formatter.Options) ([]byte, error) {
	var sb strings.Builder
	sb.WriteString(textMatePlistHeader)

	sorted := formatter.SortTokens(tokens)

	// Build token index for light-dark detection
	tokenIndex := buildTokenIndex(sorted, opts.Prefix)

	for _, tok := range sorted {
		name := buildTokenName(tok.Path, opts.Prefix)

		// Check if this token is part of a light-dark group
		if group := findLightDarkGroup(tok, tokenIndex); group != nil {
			// Only emit the combined snippet for the root token
			if isRootToken(tok, group) {
				rootName := getRootName(group, opts.Prefix)
				lightName := buildTokenName(group.Light.Path, opts.Prefix)
				darkName := buildTokenName(group.Dark.Path, opts.Prefix)
				lightValue := getColorValue(group.Light)
				darkValue := getColorValue(group.Dark)
				body := buildLightDarkBody(rootName, lightName, darkName, lightValue, darkValue)
				fmt.Fprintf(&sb, textMateSnippetTemplate, rootName, rootName, body)
			}
			// Skip individual snippets for light/dark children
			continue
		}

		cssVar := fmt.Sprintf("var(--%s)", name)
		fmt.Fprintf(&sb, textMateSnippetTemplate, name, name, cssVar)
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
		name := buildTokenName(tok.Path, opts.Prefix)

		// Check if this token is part of a light-dark group
		if group := findLightDarkGroup(tok, tokenIndex); group != nil {
			// Only emit the combined snippet for the root token
			if isRootToken(tok, group) {
				rootName := getRootName(group, opts.Prefix)
				snippet := buildZedLightDarkSnippet(group, rootName, opts)
				snippetMap[rootName] = snippet
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
	snippet := ZedSnippet{
		Prefix: name,
		Body:   []string{fmt.Sprintf("var(--%s)", name)},
	}

	if tok.Description != "" {
		snippet.Description = tok.Description
	}

	return snippet
}

// buildZedLightDarkSnippet creates a Zed snippet with light-dark() pattern.
func buildZedLightDarkSnippet(group *lightDarkGroup, name string, opts formatter.Options) ZedSnippet {
	lightName := buildTokenName(group.Light.Path, opts.Prefix)
	darkName := buildTokenName(group.Dark.Path, opts.Prefix)

	// Get resolved color values for fallbacks
	lightValue := getColorValue(group.Light)
	darkValue := getColorValue(group.Dark)

	body := buildLightDarkBody(name, lightName, darkName, lightValue, darkValue)

	snippet := ZedSnippet{
		Prefix: name,
		Body:   []string{body},
	}

	// Use description from real root if available, otherwise from light token
	if group.Root != group.Light && group.Root.Description != "" {
		snippet.Description = group.Root.Description
	} else if group.Light.Description != "" {
		snippet.Description = group.Light.Description
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

// buildTokenName creates a CSS custom property name from a token path.
func buildTokenName(path []string, prefix string) string {
	name := formatter.ToKebabCase(strings.Join(path, "-"))
	if prefix != "" {
		return fmt.Sprintf("%s-%s", prefix, name)
	}
	return name
}

// buildTokenIndex creates a map from token path to token for efficient lookup.
func buildTokenIndex(tokens []*token.Token, prefix string) map[string]*tokenIndexEntry {
	index := make(map[string]*tokenIndexEntry, len(tokens))
	for _, tok := range tokens {
		path := strings.Join(tok.Path, ".")
		name := buildTokenName(tok.Path, prefix)
		index[path] = &tokenIndexEntry{Token: tok, Name: name}
	}
	return index
}

// buildLightDarkBody creates the CSS light-dark() function body.
func buildLightDarkBody(name, lightName, darkName, lightValue, darkValue string) string {
	if lightValue != "" && darkValue != "" {
		return fmt.Sprintf(
			"var(--%s, light-dark(\n  var(--%s, %s),\n  var(--%s, %s)\n))",
			name, lightName, lightValue, darkName, darkValue,
		)
	}
	return fmt.Sprintf(
		"var(--%s, light-dark(\n  var(--%s),\n  var(--%s)\n))",
		name, lightName, darkName,
	)
}

// findLightDarkGroup checks if a token is part of a light-dark group.
// Returns the group if found, nil otherwise.
//
// Detection rules (convention-based):
// - A color token ending in ".light" that has a sibling ".dark" token
// - A color token with a Reference field that points to a ".light" child
func findLightDarkGroup(tok *token.Token, index map[string]*tokenIndexEntry) *lightDarkGroup {
	if tok.Type != token.TypeColor {
		return nil
	}

	tokPath := strings.Join(tok.Path, ".")

	// Check if this token IS the root (has Reference pointing to light child)
	if tok.Reference != "" {
		lightPath := fmt.Sprintf("%s.light", tokPath)
		darkPath := fmt.Sprintf("%s.dark", tokPath)

		expectedRef := fmt.Sprintf("{%s}", lightPath)
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

	// Check if this token is a light/dark child (convention-based detection)
	if len(tok.Path) < 2 {
		return nil
	}

	lastSegment := tok.Path[len(tok.Path)-1]
	if lastSegment != "light" && lastSegment != "dark" {
		return nil
	}

	// Build sibling paths
	parentPath := strings.Join(tok.Path[:len(tok.Path)-1], ".")
	lightPath := fmt.Sprintf("%s.light", parentPath)
	darkPath := fmt.Sprintf("%s.dark", parentPath)

	lightEntry, hasLight := index[lightPath]
	darkEntry, hasDark := index[darkPath]

	if !hasLight || !hasDark {
		return nil
	}

	// Check if there's a root token (optional - may not exist if using $root syntax)
	rootEntry, hasRoot := index[parentPath]

	return &lightDarkGroup{
		Root:  rootTokenOrLight(rootEntry, lightEntry, hasRoot),
		Light: lightEntry.Token,
		Dark:  darkEntry.Token,
	}
}

// rootTokenOrLight returns the root token if it exists, otherwise the light token.
// This allows light-dark detection to work even when the parser doesn't emit a root token.
func rootTokenOrLight(rootEntry, lightEntry *tokenIndexEntry, hasRoot bool) *token.Token {
	if hasRoot {
		return rootEntry.Token
	}
	return lightEntry.Token
}

// isRootToken checks if the given token is the root of the light-dark group.
// When there's no explicit root token, the light token is used as surrogate.
func isRootToken(tok *token.Token, group *lightDarkGroup) bool {
	if tok == group.Root {
		return true
	}
	// When Root == Light, we're using light as surrogate root
	// Only emit when processing the light token
	if group.Root == group.Light {
		return tok == group.Light
	}
	return false
}

// getRootName returns the CSS custom property name for the root of a light-dark group.
// When using a surrogate root (light token), derives the parent name.
func getRootName(group *lightDarkGroup, prefix string) string {
	// If there's a real root token (different from light), use its path
	if group.Root != group.Light {
		return buildTokenName(group.Root.Path, prefix)
	}
	// Derive parent name from light token's path (remove ".light" suffix)
	if len(group.Light.Path) > 1 {
		parentPath := group.Light.Path[:len(group.Light.Path)-1]
		return buildTokenName(parentPath, prefix)
	}
	return buildTokenName(group.Light.Path, prefix)
}

// buildLightDarkSnippet creates a snippet with light-dark() pattern.
func buildLightDarkSnippet(group *lightDarkGroup, name string, opts formatter.Options) Snippet {
	lightName := buildTokenName(group.Light.Path, opts.Prefix)
	darkName := buildTokenName(group.Dark.Path, opts.Prefix)

	// Get resolved color values for fallbacks
	lightValue := getColorValue(group.Light)
	darkValue := getColorValue(group.Dark)

	body := buildLightDarkBody(name, lightName, darkName, lightValue, darkValue)
	// Use name-only prefixes for combined snippets (no hex values)
	prefixes := buildNamePrefixes(name)

	snippet := Snippet{
		Scope:  "css,scss,less,stylus,postcss",
		Prefix: prefixes,
		Body:   []string{body},
	}

	// Use description from real root if available, otherwise from light token
	if group.Root != group.Light && group.Root.Description != "" {
		snippet.Description = group.Root.Description
	} else if group.Light.Description != "" {
		snippet.Description = group.Light.Description
	}

	return snippet
}

// buildNamePrefixes generates prefix array without color hex values.
func buildNamePrefixes(name string) []string {
	prefixes := []string{name}

	camelName := formatter.ToCamelCase(name)
	if camelName != name {
		prefixes = append(prefixes, camelName)
	}

	underscoreName := strings.ReplaceAll(name, "-", "_")
	if underscoreName != name {
		prefixes = append(prefixes, underscoreName)
	}

	return prefixes
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
	prefixes := buildPrefixes(tok, name)

	snippet := Snippet{
		Scope:  "css,scss,less,stylus,postcss",
		Prefix: prefixes,
		Body:   []string{fmt.Sprintf("var(--%s)", name)},
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
