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
	"strings"

	"bennypowers.dev/asimonim/token"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// sourceSummary is the JSON representation of a token source.
type sourceSummary struct {
	Source     string `json:"source"`
	TokenCount int   `json:"tokenCount"`
}

// tokenDetail is the full JSON representation of an individual token.
type tokenDetail struct {
	Name              string   `json:"name"`
	Path              string   `json:"path"`
	Value             string   `json:"value"`
	Type              string   `json:"type,omitempty"`
	Description       string   `json:"description,omitempty"`
	Deprecated        bool     `json:"deprecated,omitempty"`
	DeprecationMsg    string   `json:"deprecationMessage,omitempty"`
	CSSVariableName   string   `json:"cssVariableName,omitempty"`
	CSSSyntax         string   `json:"cssSyntax,omitempty"`
	IsResolved        bool     `json:"isResolved,omitempty"`
	ResolutionChain   []string `json:"resolutionChain,omitempty"`
}

func newTokenDetail(tok *token.Token) tokenDetail {
	return tokenDetail{
		Name:            tok.Name,
		Path:            tok.DotPath(),
		Value:           tok.DisplayValue(),
		Type:            tok.Type,
		Description:     tok.Description,
		Deprecated:      tok.Deprecated,
		DeprecationMsg:  tok.DeprecationMessage,
		CSSVariableName: tok.CSSVariableName(),
		CSSSyntax:       tok.CSSSyntax(),
		IsResolved:      tok.IsResolved,
		ResolutionChain: tok.ResolutionChain,
	}
}

func (s *Server) setupResources() {
	// asimonim://tokens - list available token sources
	s.server.AddResource(&mcp.Resource{
		URI:         "asimonim://tokens",
		Name:        "tokens",
		Description: "Lists available design token sources in the workspace with token counts.",
		MIMEType:    "application/json",
	}, s.handleTokenSources)

	// asimonim://config - workspace configuration
	s.server.AddResource(&mcp.Resource{
		URI:         "asimonim://config",
		Name:        "config",
		Description: "Workspace design token configuration (files, resolvers, schema, prefix).",
		MIMEType:    "application/json",
	}, s.handleConfig)

	// asimonim://tokens/{+source} - all tokens for a source
	s.server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "asimonim://tokens/{+source}",
		Name:        "tokens-by-source",
		Description: "All design tokens from a specific source (file or package).",
		MIMEType:    "application/json",
	}, s.handleTokensBySource)

	// asimonim://token/{+rest} - individual token detail (rest = source/path...)
	s.server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "asimonim://token/{+rest}",
		Name:        "token-detail",
		Description: "Detailed information about an individual design token by source and path.",
		MIMEType:    "application/json",
	}, s.handleTokenDetail)
}

func (s *Server) handleTokenSources(
	_ context.Context,
	req *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	if err := validateResourceRequest(req); err != nil {
		return nil, err
	}

	parsed, err := parseWorkspaceTokens(s.fs, s.cfg, nil, s.cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tokens: %w", err)
	}

	summaries := make([]sourceSummary, len(parsed.Sources))
	for i, src := range parsed.Sources {
		summaries[i] = sourceSummary{
			Source:     src.Source,
			TokenCount: len(src.Tokens),
		}
	}

	data, err := json.MarshalIndent(summaries, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sources: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

func (s *Server) handleConfig(
	_ context.Context,
	req *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	if err := validateResourceRequest(req); err != nil {
		return nil, err
	}

	cfgData := map[string]any{
		"prefix":    s.cfg.Prefix,
		"schema":    s.cfg.Schema,
		"files":     s.cfg.FilePaths(),
		"resolvers": s.cfg.Resolvers,
	}

	data, err := json.MarshalIndent(cfgData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

func (s *Server) handleTokensBySource(
	_ context.Context,
	req *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	if err := validateResourceRequest(req); err != nil {
		return nil, err
	}

	// Extract source from URI: asimonim://tokens/{+source}
	source, ok := extractURISuffix(req.Params.URI, "asimonim://tokens/")
	if !ok {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	parsed, err := parseWorkspaceTokens(s.fs, s.cfg, nil, s.cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tokens: %w", err)
	}

	for _, src := range parsed.Sources {
		if src.Source == source {
			summaries := make([]tokenSummary, len(src.Tokens))
			for i, tok := range src.Tokens {
				summaries[i] = newTokenSummary(tok)
			}

			data, err := json.MarshalIndent(summaries, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal tokens: %w", err)
			}

			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     string(data),
				}},
			}, nil
		}
	}

	return nil, mcp.ResourceNotFoundError(req.Params.URI)
}

func (s *Server) handleTokenDetail(
	_ context.Context,
	req *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	if err := validateResourceRequest(req); err != nil {
		return nil, err
	}

	// Extract rest from URI: asimonim://token/{+rest}
	rest, ok := extractURISuffix(req.Params.URI, "asimonim://token/")
	if !ok {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	parsed, err := parseWorkspaceTokens(s.fs, s.cfg, nil, s.cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tokens: %w", err)
	}

	// Try to find the token by matching source + path
	for _, src := range parsed.Sources {
		prefix := src.Source + "/"
		if !strings.HasPrefix(rest, prefix) {
			continue
		}
		tokenPath := strings.TrimPrefix(rest, prefix)
		// Convert slash-separated path to dot-separated for matching
		dotPath := strings.ReplaceAll(tokenPath, "/", ".")

		for _, tok := range src.Tokens {
			if tok.DotPath() == dotPath {
				detail := newTokenDetail(tok)
				data, err := json.MarshalIndent(detail, "", "  ")
				if err != nil {
					return nil, fmt.Errorf("failed to marshal token: %w", err)
				}

				return &mcp.ReadResourceResult{
					Contents: []*mcp.ResourceContents{{
						URI:      req.Params.URI,
						MIMEType: "application/json",
						Text:     string(data),
					}},
				}, nil
			}
		}
	}

	return nil, mcp.ResourceNotFoundError(req.Params.URI)
}

// validateResourceRequest checks that the request has valid params.
func validateResourceRequest(req *mcp.ReadResourceRequest) error {
	if req == nil || req.Params == nil || req.Params.URI == "" {
		return fmt.Errorf("invalid resource request: missing URI")
	}
	return nil
}

// extractURISuffix extracts the suffix after a URI prefix.
func extractURISuffix(uri, prefix string) (string, bool) {
	if !strings.HasPrefix(uri, prefix) {
		return "", false
	}
	suffix := strings.TrimPrefix(uri, prefix)
	if suffix == "" {
		return "", false
	}
	return suffix, true
}
