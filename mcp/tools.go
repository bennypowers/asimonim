/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	convertlib "bennypowers.dev/asimonim/convert"
	"bennypowers.dev/asimonim/resolver"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/token"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// validateInput is the input schema for the validate_tokens tool.
type validateInput struct {
	// Files are token file paths to validate. If empty, uses workspace config.
	Files []string `json:"files,omitempty"`
	// Strict fails on warnings (e.g., deprecated tokens).
	Strict bool `json:"strict,omitempty"`
}

// searchInput is the input schema for the search_tokens tool.
type searchInput struct {
	// Query is the search string (substring match or regex).
	Query string `json:"query"`
	// Regex treats the query as a regular expression.
	Regex bool `json:"regex,omitempty"`
	// NameOnly searches token names only.
	NameOnly bool `json:"name_only,omitempty"`
	// ValueOnly searches token values only.
	ValueOnly bool `json:"value_only,omitempty"`
	// Type filters results by token type (color, dimension, etc.).
	Type string `json:"type,omitempty"`
	// Group filters results by group/path prefix (e.g., "color.brand").
	Group string `json:"group,omitempty"`
}

// convertInput is the input schema for the convert_tokens tool.
type convertInput struct {
	// Format is the output format (css, scss, js, swift, android, dtcg, json, snippets).
	Format string `json:"format"`
	// Files are token file paths to convert. If empty, uses workspace config.
	Files []string `json:"files,omitempty"`
	// Prefix for output variable names.
	Prefix string `json:"prefix,omitempty"`
	// CSSSelector for custom properties (:root or :host).
	CSSSelector string `json:"css_selector,omitempty"`
	// Type filters tokens by type before conversion.
	Type string `json:"type,omitempty"`
	// Group filters tokens by group/path prefix before conversion.
	Group string `json:"group,omitempty"`
}

// tokenSummary is the JSON representation of a token in tool results.
type tokenSummary struct {
	Name            string `json:"name"`
	Path            string `json:"path"`
	Value           string `json:"value"`
	Type            string `json:"type,omitempty"`
	Description     string `json:"description,omitempty"`
	Deprecated      bool   `json:"deprecated,omitempty"`
	CSSVariableName string `json:"cssVariableName,omitempty"`
}

func newTokenSummary(tok *token.Token) tokenSummary {
	return tokenSummary{
		Name:            tok.Name,
		Path:            tok.DotPath(),
		Value:           tok.DisplayValue(),
		Type:            tok.Type,
		Description:     tok.Description,
		Deprecated:      tok.Deprecated,
		CSSVariableName: tok.CSSVariableName(),
	}
}

func (s *Server) setupTools() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "validate_tokens",
		Description: "Validate design token files for correctness, detect circular references, and report deprecated tokens.",
	}, s.handleValidate)

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "search_tokens",
		Description: "Search design tokens by name, value, description, or type with optional regex support.",
	}, s.handleSearch)

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "convert_tokens",
		Description: "Convert design tokens to CSS, SCSS, JavaScript, Swift, Android XML, or other formats.",
	}, s.handleConvert)
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

func errorResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
		IsError: true,
	}
}

func (s *Server) handleValidate(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input validateInput,
) (*mcp.CallToolResult, any, error) {
	parsed, err := parseWorkspaceTokens(s.fs, s.cfg, input.Files, s.cwd)
	if err != nil {
		return errorResult(fmt.Sprintf("Validation error: %v", err)), nil, nil
	}

	var sb strings.Builder
	var hasErrors, hasWarnings bool

	for _, src := range parsed.Sources {
		sb.WriteString(fmt.Sprintf("Source: %s\n", src.Source))

		graph := resolver.BuildDependencyGraph(src.Tokens)
		if cycle := graph.FindCycle(); cycle != nil {
			sb.WriteString(fmt.Sprintf("  ERROR: Circular reference: %v\n", cycle))
			hasErrors = true
			continue
		}

		deprecatedCount := 0
		for _, tok := range src.Tokens {
			if tok.Deprecated {
				deprecatedCount++
			}
		}

		sb.WriteString(fmt.Sprintf("  %d tokens, schema: %s\n", len(src.Tokens), parsed.Version))
		if deprecatedCount > 0 {
			hasWarnings = true
			sb.WriteString(fmt.Sprintf("  WARNING: %d deprecated token(s)\n", deprecatedCount))
		}
	}

	if hasErrors {
		sb.WriteString("\nValidation FAILED.\n")
		return errorResult(sb.String()), nil, nil
	}

	if input.Strict && hasWarnings {
		sb.WriteString("\nValidation FAILED (strict mode: warnings treated as errors).\n")
		return errorResult(sb.String()), nil, nil
	}

	sb.WriteString("\nAll files valid.\n")
	return textResult(sb.String()), nil, nil
}

func (s *Server) handleSearch(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input searchInput,
) (*mcp.CallToolResult, any, error) {
	if input.Query == "" {
		return errorResult("Error: query is required"), nil, nil
	}

	parsed, err := parseWorkspaceTokens(s.fs, s.cfg, nil, s.cwd)
	if err != nil {
		return errorResult(fmt.Sprintf("Error: %v", err)), nil, nil
	}

	var pattern *regexp.Regexp
	if input.Regex {
		pattern, err = regexp.Compile(input.Query)
		if err != nil {
			return errorResult(fmt.Sprintf("Invalid regex: %v", err)), nil, nil
		}
	}

	var matches []*token.Token
	for _, tok := range parsed.AllTokens {
		matched := false
		if input.NameOnly {
			matched = matchString(tok.Name, input.Query, pattern)
		} else if input.ValueOnly {
			matched = matchString(tok.DisplayValue(), input.Query, pattern) ||
				matchString(tok.Value, input.Query, pattern)
		} else {
			matched = matchString(tok.Name, input.Query, pattern) ||
				matchString(tok.DisplayValue(), input.Query, pattern) ||
				matchString(tok.Value, input.Query, pattern) ||
				matchString(tok.Type, input.Query, pattern) ||
				matchString(tok.Description, input.Query, pattern)
		}
		if matched {
			matches = append(matches, tok)
		}
	}

	matches = filterTokens(matches, input.Type, input.Group)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Name < matches[j].Name
	})

	summaries := make([]tokenSummary, len(matches))
	for i, tok := range matches {
		summaries[i] = newTokenSummary(tok)
	}

	data, err := json.Marshal(summaries)
	if err != nil {
		return errorResult(fmt.Sprintf("Error: failed to marshal results: %v", err)), nil, nil
	}

	return textResult(string(data)), nil, nil
}

func (s *Server) handleConvert(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input convertInput,
) (*mcp.CallToolResult, any, error) {
	if input.Format == "" {
		return errorResult("Error: format is required"), nil, nil
	}

	format, err := convertlib.ParseFormat(input.Format)
	if err != nil {
		return errorResult(fmt.Sprintf("Error: %v", err)), nil, nil
	}

	parsed, err := parseWorkspaceTokens(s.fs, s.cfg, input.Files, s.cwd)
	if err != nil {
		return errorResult(fmt.Sprintf("Error: %v", err)), nil, nil
	}

	tokens := filterTokens(parsed.AllTokens, input.Type, input.Group)

	outputSchema := parsed.Version
	if outputSchema == schema.Unknown {
		outputSchema = schema.Draft
	}

	opts := convertlib.Options{
		InputSchema:  parsed.Version,
		OutputSchema: outputSchema,
		Format:       format,
		Prefix:       input.Prefix,
		CSSSelector:  input.CSSSelector,
	}

	output, err := convertlib.FormatTokens(tokens, format, opts)
	if err != nil {
		return errorResult(fmt.Sprintf("Error formatting output: %v", err)), nil, nil
	}

	return textResult(string(output)), nil, nil
}

func matchString(s, query string, pattern *regexp.Regexp) bool {
	if pattern != nil {
		return pattern.MatchString(s)
	}
	return strings.Contains(strings.ToLower(s), strings.ToLower(query))
}
