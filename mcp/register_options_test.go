/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

package mcp

import (
	"context"
	"testing"

	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/internal/mapfs"
	"bennypowers.dev/asimonim/testutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	allToolNames       = []string{"convert_tokens", "search_tokens", "validate_tokens"}
	allResourceURIs    = []string{"asimonim://config", "asimonim://tokens"}
	allTemplateURIs    = []string{"asimonim://token/{+rest}", "asimonim://tokens/{+source}"}
	fixtureTokenConfig = &config.Config{Files: []config.FileSpec{{Path: "/test/tokens.json"}}}
)

func fixtureMFS(t *testing.T) *mapfs.MapFileSystem {
	t.Helper()
	return testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
}

// connectAndList connects a client to server and returns the tool names,
// resource URIs, and resource template URIs it exposes.
func connectAndList(t *testing.T, server *mcp.Server) (tools, resources, templates []string) {
	t.Helper()
	ctx := context.Background()
	session := connectSession(t, server)
	return listToolNames(t, ctx, session), listResourceURIs(t, ctx, session), listResourceTemplateURIs(t, ctx, session)
}

// existingSearchInput is the input schema for the pre-registered handler used
// by conflict tests.
type existingSearchInput struct {
	Query string `json:"query"`
}

// addExistingSearchTool pre-registers a search_tokens handler that returns
// marker, so tests can tell it apart from asimonim's handler.
func addExistingSearchTool(t *testing.T, server *mcp.Server, marker string) {
	t.Helper()
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_tokens",
		Description: "pre-existing search handler",
	}, func(context.Context, *mcp.CallToolRequest, existingSearchInput) (*mcp.CallToolResult, any, error) {
		return textResult(marker), nil, nil
	})
}

func callExistingSearch(t *testing.T, session *mcp.ClientSession) string {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "search_tokens",
		Arguments: map[string]any{"query": "primary"},
	})
	require.NoError(t, err)
	return resultText(t, result)
}

func TestRegister_DefaultOptionsPreserveSurface(t *testing.T) {
	ctx := context.Background()

	baselineSession := connectSession(t, newTestServer(t, "fixtures/draft/simple").server)
	baselineTools := normalizedTools(t, ctx, baselineSession)
	baselineResources := listResourceURIs(t, ctx, baselineSession)
	baselineTemplates := listResourceTemplateURIs(t, ctx, baselineSession)

	server := newEmbeddingServer()
	mustRegister(t, server, fixtureMFS(t), fixtureTokenConfig, "/test",
		WithTools(Selection{}),
		WithResources(Selection{}),
		WithTemplates(Selection{}),
		WithConflictPolicy(ConflictReplace),
	)
	session := connectSession(t, server)

	assert.Equal(t, baselineTools, normalizedTools(t, ctx, session))
	assert.Equal(t, baselineResources, listResourceURIs(t, ctx, session))
	assert.Equal(t, baselineTemplates, listResourceTemplateURIs(t, ctx, session))
}

func TestRegister_ToolSelection(t *testing.T) {
	tests := []struct {
		name string
		opts []RegisterOption
		want []string
	}{
		{
			name: "include one",
			opts: []RegisterOption{WithTools(Selection{Include: []string{"search_tokens"}})},
			want: []string{"search_tokens"},
		},
		{
			name: "exclude one",
			opts: []RegisterOption{WithTools(Selection{Exclude: []string{"search_tokens"}})},
			want: []string{"convert_tokens", "validate_tokens"},
		},
		{
			name: "exclude wins over include",
			opts: []RegisterOption{WithTools(Selection{
				Include: []string{"convert_tokens", "search_tokens", "validate_tokens"},
				Exclude: []string{"search_tokens"},
			})},
			want: []string{"convert_tokens", "validate_tokens"},
		},
		{
			name: "exclude everything",
			opts: []RegisterOption{WithTools(Selection{Exclude: allToolNames})},
			want: []string{},
		},
		{
			name: "include multiple",
			opts: []RegisterOption{WithTools(Selection{Include: []string{"search_tokens", "convert_tokens"}})},
			want: []string{"convert_tokens", "search_tokens"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newEmbeddingServer()
			mustRegister(t, server, fixtureMFS(t), fixtureTokenConfig, "/test", tt.opts...)

			tools, resources, templates := connectAndList(t, server)
			assert.Equal(t, tt.want, tools)
			// Tool filtering must not touch the resource surface.
			assert.Equal(t, allResourceURIs, resources)
			assert.Equal(t, allTemplateURIs, templates)
		})
	}
}

func TestRegister_ResourceSelection(t *testing.T) {
	tests := []struct {
		name string
		opts []RegisterOption
		want []string
	}{
		{
			name: "include one",
			opts: []RegisterOption{WithResources(Selection{Include: []string{"asimonim://tokens"}})},
			want: []string{"asimonim://tokens"},
		},
		{
			name: "exclude one",
			opts: []RegisterOption{WithResources(Selection{Exclude: []string{"asimonim://tokens"}})},
			want: []string{"asimonim://config"},
		},
		{
			name: "exclude wins over include",
			opts: []RegisterOption{WithResources(Selection{
				Include: allResourceURIs,
				Exclude: []string{"asimonim://tokens"},
			})},
			want: []string{"asimonim://config"},
		},
		{
			name: "exclude everything",
			opts: []RegisterOption{WithResources(Selection{Exclude: allResourceURIs})},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newEmbeddingServer()
			mustRegister(t, server, fixtureMFS(t), fixtureTokenConfig, "/test", tt.opts...)

			tools, resources, templates := connectAndList(t, server)
			assert.Equal(t, allToolNames, tools)
			assert.Equal(t, tt.want, resources)
			assert.Equal(t, allTemplateURIs, templates)
		})
	}
}

func TestRegister_ResourceSelectionRemovesHandler(t *testing.T) {
	server := newEmbeddingServer()
	mustRegister(t, server, fixtureMFS(t), fixtureTokenConfig, "/test",
		WithResources(Selection{Exclude: []string{"asimonim://tokens"}}),
	)
	session := connectSession(t, server)

	_, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: "asimonim://tokens"})
	require.Error(t, err)

	result, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: "asimonim://config"})
	require.NoError(t, err)
	require.Len(t, result.Contents, 1)
}

func TestRegister_TemplateSelection(t *testing.T) {
	tests := []struct {
		name string
		opts []RegisterOption
		want []string
	}{
		{
			name: "include one",
			opts: []RegisterOption{WithTemplates(Selection{Include: []string{"asimonim://token/{+rest}"}})},
			want: []string{"asimonim://token/{+rest}"},
		},
		{
			name: "exclude one",
			opts: []RegisterOption{WithTemplates(Selection{Exclude: []string{"asimonim://token/{+rest}"}})},
			want: []string{"asimonim://tokens/{+source}"},
		},
		{
			name: "exclude wins over include",
			opts: []RegisterOption{WithTemplates(Selection{
				Include: allTemplateURIs,
				Exclude: []string{"asimonim://tokens/{+source}"},
			})},
			want: []string{"asimonim://token/{+rest}"},
		},
		{
			name: "exclude everything",
			opts: []RegisterOption{WithTemplates(Selection{Exclude: allTemplateURIs})},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newEmbeddingServer()
			mustRegister(t, server, fixtureMFS(t), fixtureTokenConfig, "/test", tt.opts...)

			tools, resources, templates := connectAndList(t, server)
			assert.Equal(t, allToolNames, tools)
			assert.Equal(t, allResourceURIs, resources)
			assert.Equal(t, tt.want, templates)
		})
	}
}

func TestRegister_UnknownIncludeIsError(t *testing.T) {
	tests := []struct {
		name   string
		opts   []RegisterOption
		absent string
	}{
		{
			name:   "tools",
			opts:   []RegisterOption{WithTools(Selection{Include: []string{"no_such_tool"}})},
			absent: "no_such_tool",
		},
		{
			name:   "resources",
			opts:   []RegisterOption{WithResources(Selection{Include: []string{"asimonim://nope"}})},
			absent: "asimonim://nope",
		},
		{
			name:   "templates",
			opts:   []RegisterOption{WithTemplates(Selection{Include: []string{"asimonim://nope/{+rest}"}})},
			absent: "asimonim://nope/{+rest}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newEmbeddingServer()
			s, err := Register(server, fixtureMFS(t), fixtureTokenConfig, "/test", tt.opts...)
			require.Error(t, err)
			assert.Nil(t, s)
			assert.Contains(t, err.Error(), tt.absent)

			// Validation happens before registration, so nothing was mounted.
			tools, resources, templates := connectAndList(t, server)
			assert.Empty(t, tools)
			assert.Empty(t, resources)
			assert.Empty(t, templates)
		})
	}
}

func TestRegister_UnknownExcludeIsAccepted(t *testing.T) {
	server := newEmbeddingServer()
	mustRegister(t, server, fixtureMFS(t), fixtureTokenConfig, "/test",
		WithTools(Selection{Exclude: []string{"no_such_tool"}}),
		WithResources(Selection{Exclude: []string{"asimonim://nope"}}),
		WithTemplates(Selection{Exclude: []string{"asimonim://nope/{+rest}"}}),
	)

	tools, resources, templates := connectAndList(t, server)
	assert.Equal(t, allToolNames, tools)
	assert.Equal(t, allResourceURIs, resources)
	assert.Equal(t, allTemplateURIs, templates)
}

func TestRegister_EmptySelectionEntryIsError(t *testing.T) {
	tests := []struct {
		name string
		opts []RegisterOption
	}{
		{"tools include", []RegisterOption{WithTools(Selection{Include: []string{""}})}},
		{"tools exclude", []RegisterOption{WithTools(Selection{Exclude: []string{""}})}},
		{"resources include", []RegisterOption{WithResources(Selection{Include: []string{""}})}},
		{"resources exclude", []RegisterOption{WithResources(Selection{Exclude: []string{""}})}},
		{"templates include", []RegisterOption{WithTemplates(Selection{Include: []string{""}})}},
		{"templates exclude", []RegisterOption{WithTemplates(Selection{Exclude: []string{""}})}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newEmbeddingServer()
			_, err := Register(server, fixtureMFS(t), fixtureTokenConfig, "/test", tt.opts...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "empty")

			tools, resources, templates := connectAndList(t, server)
			assert.Empty(t, tools)
			assert.Empty(t, resources)
			assert.Empty(t, templates)
		})
	}
}

func TestRegister_RepeatedFiltersAccumulate(t *testing.T) {
	server := newEmbeddingServer()
	mustRegister(t, server, fixtureMFS(t), fixtureTokenConfig, "/test",
		WithTools(Selection{Include: []string{"search_tokens"}}),
		WithTools(Selection{Include: []string{"convert_tokens"}}),
		WithResources(Selection{Include: []string{"asimonim://config"}}),
		WithResources(Selection{Include: []string{"asimonim://tokens"}}),
		WithTemplates(Selection{Exclude: []string{"asimonim://token/{+rest}"}}),
		WithTemplates(Selection{Exclude: []string{"asimonim://tokens/{+source}"}}),
	)

	tools, resources, templates := connectAndList(t, server)
	assert.Equal(t, []string{"convert_tokens", "search_tokens"}, tools)
	assert.Equal(t, allResourceURIs, resources)
	assert.Empty(t, templates)
}

func TestRegister_FilteredToolAbsentFromListAndCallFails(t *testing.T) {
	ctx := context.Background()

	server := newEmbeddingServer()
	mustRegister(t, server, fixtureMFS(t), fixtureTokenConfig, "/test",
		WithTools(Selection{Include: []string{"validate_tokens"}}),
	)
	session := connectSession(t, server)

	assert.Equal(t, []string{"validate_tokens"}, listToolNames(t, ctx, session))

	_, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search_tokens",
		Arguments: map[string]any{"query": "primary"},
	})
	require.Error(t, err)

	_, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "convert_tokens",
		Arguments: map[string]any{"format": "css"},
	})
	require.Error(t, err)

	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "validate_tokens"})
	require.NoError(t, err)
	require.False(t, result.IsError, "unexpected tool error: %s", resultText(t, result))
}

func TestRegister_ConflictErrorLeavesServerUnchanged(t *testing.T) {
	ctx := context.Background()

	server := newEmbeddingServer()
	addExistingSearchTool(t, server, "existing-handler")

	s, err := Register(server, fixtureMFS(t), fixtureTokenConfig, "/test", WithConflictPolicy(ConflictError))
	require.Error(t, err)
	assert.Nil(t, s)
	assert.Contains(t, err.Error(), "search_tokens")

	session := connectSession(t, server)
	assert.Equal(t, []string{"search_tokens"}, listToolNames(t, ctx, session))
	assert.Empty(t, listResourceURIs(t, ctx, session))
	assert.Empty(t, listResourceTemplateURIs(t, ctx, session))
	assert.Equal(t, "existing-handler", callExistingSearch(t, session))
}

func TestRegister_ConflictSkipKeepsExisting(t *testing.T) {
	ctx := context.Background()

	server := newEmbeddingServer()
	addExistingSearchTool(t, server, "existing-handler")

	mustRegister(t, server, fixtureMFS(t), fixtureTokenConfig, "/test", WithConflictPolicy(ConflictSkip))

	session := connectSession(t, server)
	assert.Equal(t, allToolNames, listToolNames(t, ctx, session))
	assert.Equal(t, allResourceURIs, listResourceURIs(t, ctx, session))
	assert.Equal(t, allTemplateURIs, listResourceTemplateURIs(t, ctx, session))
	assert.Equal(t, "existing-handler", callExistingSearch(t, session))
}

func TestRegister_ConflictReplaceKeepsDefaultBehavior(t *testing.T) {
	ctx := context.Background()

	server := newEmbeddingServer()
	addExistingSearchTool(t, server, "existing-handler")

	mustRegister(t, server, fixtureMFS(t), fixtureTokenConfig, "/test")

	session := connectSession(t, server)
	assert.Equal(t, allToolNames, listToolNames(t, ctx, session))
	assert.Contains(t, callExistingSearch(t, session), "color-primary")
	assert.NotEqual(t, "existing-handler", callExistingSearch(t, session))
}

func TestRegister_ConflictPolicyLastValueWins(t *testing.T) {
	server := newEmbeddingServer()
	addExistingSearchTool(t, server, "existing-handler")

	// ConflictError would fail, but the later ConflictSkip replaces it.
	mustRegister(t, server, fixtureMFS(t), fixtureTokenConfig, "/test",
		WithConflictPolicy(ConflictError),
		WithConflictPolicy(ConflictSkip),
	)

	session := connectSession(t, server)
	assert.Equal(t, "existing-handler", callExistingSearch(t, session))
}

func TestRegister_ConflictErrorDetectsResourceConflict(t *testing.T) {
	server := newEmbeddingServer()
	server.AddResource(&mcp.Resource{
		URI:      "asimonim://tokens",
		Name:     "existing",
		MIMEType: "application/json",
	}, func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{}, nil
	})

	s, err := Register(server, fixtureMFS(t), fixtureTokenConfig, "/test", WithConflictPolicy(ConflictError))
	require.Error(t, err)
	assert.Nil(t, s)
	assert.Contains(t, err.Error(), "asimonim://tokens")

	_, resources, _ := connectAndList(t, server)
	assert.Equal(t, []string{"asimonim://tokens"}, resources)
}

func TestRegister_ExplicitConfigSharedAcrossSessions(t *testing.T) {
	server := newEmbeddingServer()
	cfg := &config.Config{Files: []config.FileSpec{{Path: "/project/tokens.json"}}}
	mustRegister(t, server, projectFS(), cfg, "/elsewhere")

	// Roots point at unrelated workspaces, but an explicit config is shared.
	sessionA := connectSession(t, server, &mcp.Root{URI: "file:///project-a"})
	sessionB := connectSession(t, server, &mcp.Root{URI: "file:///project-b"})

	assertSearchFinds(t, context.Background(), sessionA, "primary")
	assertSearchFinds(t, context.Background(), sessionB, "primary")
}
