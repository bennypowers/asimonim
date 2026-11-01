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
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/internal/mapfs"
	"bennypowers.dev/asimonim/testutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	allToolNames       = []string{toolConvert, toolSearch, toolValidate}
	allResourceURIs    = []string{resourceConfig, resourceTokens}
	allTemplateURIs    = []string{templateTokenDetail, templateTokensBySource}
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

// addExistingTool pre-registers a tool that returns marker, so tests can tell
// it apart from asimonim's handlers.
func addExistingTool(t *testing.T, server *mcp.Server, name, marker string) {
	t.Helper()
	mcp.AddTool(server, &mcp.Tool{
		Name:        name,
		Description: "pre-existing handler",
	}, func(context.Context, *mcp.CallToolRequest, existingSearchInput) (*mcp.CallToolResult, any, error) {
		return textResult(marker), nil, nil
	})
}

// addExistingSearchTool pre-registers a search_tokens handler that returns
// marker.
func addExistingSearchTool(t *testing.T, server *mcp.Server, marker string) {
	t.Helper()
	addExistingTool(t, server, toolSearch, marker)
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

	// Also pin the literal surface so the baseline is not only self-referential.
	assert.Equal(t, allToolNames, listToolNames(t, ctx, session))
	assert.Equal(t, allResourceURIs, listResourceURIs(t, ctx, session))
	assert.Equal(t, allTemplateURIs, listResourceTemplateURIs(t, ctx, session))
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
		URI:      resourceTokens,
		Name:     "existing",
		MIMEType: "application/json",
	}, func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{}, nil
	})

	s, err := Register(server, fixtureMFS(t), fixtureTokenConfig, "/test", WithConflictPolicy(ConflictError))
	require.Error(t, err)
	assert.Nil(t, s)
	assert.Contains(t, err.Error(), resourceTokens)

	tools, resources, templates := connectAndList(t, server)
	assert.Empty(t, tools, "validation must stop before other categories register")
	assert.Equal(t, []string{resourceTokens}, resources)
	assert.Empty(t, templates)
}

func TestRegister_ConflictErrorDetectsTemplateConflict(t *testing.T) {
	server := newEmbeddingServer()
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: templateTokensBySource,
		Name:        "existing",
		MIMEType:    "application/json",
	}, func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{}, nil
	})

	s, err := Register(server, fixtureMFS(t), fixtureTokenConfig, "/test", WithConflictPolicy(ConflictError))
	require.Error(t, err)
	assert.Nil(t, s)
	assert.Contains(t, err.Error(), templateTokensBySource)

	tools, resources, templates := connectAndList(t, server)
	assert.Empty(t, tools)
	assert.Empty(t, resources)
	assert.Equal(t, []string{templateTokensBySource}, templates)
}

func TestRegister_ConflictSkipKeepsExistingResource(t *testing.T) {
	server := newEmbeddingServer()
	server.AddResource(&mcp.Resource{
		URI:      resourceTokens,
		Name:     "existing",
		MIMEType: "application/json",
	}, func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{URI: req.Params.URI, Text: "existing-handler"}},
		}, nil
	})

	mustRegister(t, server, fixtureMFS(t), fixtureTokenConfig, "/test", WithConflictPolicy(ConflictSkip))

	session := connectSession(t, server)
	result, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: resourceTokens})
	require.NoError(t, err)
	require.Len(t, result.Contents, 1)
	assert.Equal(t, "existing-handler", result.Contents[0].Text)

	_, resources, templates := connectAndList(t, server)
	assert.Equal(t, allResourceURIs, resources)
	assert.Equal(t, allTemplateURIs, templates)
}

func TestRegister_ExcludedConflictDoesNotFail(t *testing.T) {
	ctx := context.Background()

	server := newEmbeddingServer()
	addExistingSearchTool(t, server, "existing-handler")

	// The conflicting tool is filtered out, so conflict detection must not
	// see it and ConflictError must not fire.
	mustRegister(t, server, fixtureMFS(t), fixtureTokenConfig, "/test",
		WithTools(Selection{Exclude: []string{toolSearch}}),
		WithConflictPolicy(ConflictError),
	)

	session := connectSession(t, server)
	assert.Equal(t, allToolNames, listToolNames(t, ctx, session))
	assert.Equal(t, "existing-handler", callExistingSearch(t, session))
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

func TestRegister_OptionCapturesSelectionAtConstruction(t *testing.T) {
	sel := Selection{Include: []string{toolSearch}}
	opt := WithTools(sel)
	// Mutating the caller's slice after constructing the option must not
	// change what the option selects.
	sel.Include[0] = toolConvert

	server := newEmbeddingServer()
	mustRegister(t, server, fixtureMFS(t), fixtureTokenConfig, "/test", opt)

	tools, _, _ := connectAndList(t, server)
	assert.Equal(t, []string{toolSearch}, tools)
}

func TestRegister_UnknownConflictPolicyIsError(t *testing.T) {
	server := newEmbeddingServer()
	s, err := Register(server, fixtureMFS(t), fixtureTokenConfig, "/test", WithConflictPolicy(ConflictPolicy(99)))
	require.Error(t, err)
	assert.Nil(t, s)
	assert.Contains(t, err.Error(), "conflict policy")

	tools, resources, templates := connectAndList(t, server)
	assert.Empty(t, tools)
	assert.Empty(t, resources)
	assert.Empty(t, templates)
}

func TestRegister_NilOptionIsIgnored(t *testing.T) {
	server := newEmbeddingServer()
	s, err := Register(server, fixtureMFS(t), fixtureTokenConfig, "/test", nil)
	require.NoError(t, err)
	require.NotNil(t, s)

	tools, resources, templates := connectAndList(t, server)
	assert.Equal(t, allToolNames, tools)
	assert.Equal(t, allResourceURIs, resources)
	assert.Equal(t, allTemplateURIs, templates)
}

func TestRegister_SelectionIsCaseSensitive(t *testing.T) {
	t.Run("exclude with wrong case keeps the tool", func(t *testing.T) {
		server := newEmbeddingServer()
		mustRegister(t, server, fixtureMFS(t), fixtureTokenConfig, "/test",
			WithTools(Selection{Exclude: []string{"SEARCH_TOKENS"}}),
		)

		tools, _, _ := connectAndList(t, server)
		assert.Equal(t, allToolNames, tools)
	})

	t.Run("include with wrong case is unknown", func(t *testing.T) {
		server := newEmbeddingServer()
		_, err := Register(server, fixtureMFS(t), fixtureTokenConfig, "/test",
			WithTools(Selection{Include: []string{"Search_Tokens"}}),
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Search_Tokens")
	})
}

func TestRegister_ConflictProbePaginates(t *testing.T) {
	// PageSize 1 forces the probe to follow cursors. Tools list in sorted
	// order, so search_tokens lands on the second page.
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "paged-server",
		Version: "1.0.0",
	}, &mcp.ServerOptions{PageSize: 1})
	for _, name := range []string{"aaa_tool", toolSearch, "zzz_tool"} {
		addExistingTool(t, server, name, "existing-handler")
	}

	s, err := Register(server, fixtureMFS(t), fixtureTokenConfig, "/test", WithConflictPolicy(ConflictError))
	require.Error(t, err)
	assert.Nil(t, s)
	assert.Contains(t, err.Error(), toolSearch)
}

func TestRegister_ConflictProbePaginatesAllCategories(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "paged-server",
		Version: "1.0.0",
	}, &mcp.ServerOptions{PageSize: 1})

	// Sort each category so the conflicting item is never on the first page.
	addExistingTool(t, server, "aaa_tool", "existing-handler")
	addExistingSearchTool(t, server, "existing-handler")

	emptyResource := func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{}, nil
	}
	server.AddResource(&mcp.Resource{URI: "asimonim://aaa", Name: "aaa", MIMEType: "application/json"}, emptyResource)
	server.AddResource(&mcp.Resource{URI: resourceTokens, Name: "existing", MIMEType: "application/json"}, emptyResource)

	server.AddResourceTemplate(&mcp.ResourceTemplate{URITemplate: "asimonim://aaa/{+x}", Name: "aaa", MIMEType: "application/json"}, emptyResource)
	server.AddResourceTemplate(&mcp.ResourceTemplate{URITemplate: templateTokenDetail, Name: "existing", MIMEType: "application/json"}, emptyResource)

	s, err := Register(server, fixtureMFS(t), fixtureTokenConfig, "/test", WithConflictPolicy(ConflictError))
	require.Error(t, err)
	assert.Nil(t, s)
	assert.Contains(t, err.Error(), toolSearch)
	assert.Contains(t, err.Error(), resourceTokens)
	assert.Contains(t, err.Error(), templateTokenDetail)
}

func TestRegister_ConflictProbeFailureIsReported(t *testing.T) {
	server := newEmbeddingServer()
	server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method == "initialize" {
				return nil, errors.New("rejected probe client")
			}
			return next(ctx, method, req)
		}
	})

	_, err := Register(server, fixtureMFS(t), fixtureTokenConfig, "/test", WithConflictPolicy(ConflictError))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "inspect server before registering")
	assert.Contains(t, err.Error(), "rejected probe client")
}

func TestRegister_ProbeDoesNotBlockOnStuckHandler(t *testing.T) {
	oldTimeout := probeTimeout
	probeTimeout = 100 * time.Millisecond
	t.Cleanup(func() { probeTimeout = oldTimeout })

	release := make(chan struct{})
	t.Cleanup(func() { close(release) })

	server := newEmbeddingServer()
	server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method == "tools/list" {
				// Deliberately ignore ctx and block until the test ends.
				<-release
			}
			return next(ctx, method, req)
		}
	})

	mfs := fixtureMFS(t)
	done := make(chan error, 1)
	go func() {
		_, err := Register(server, mfs, fixtureTokenConfig, "/test", WithConflictPolicy(ConflictSkip))
		done <- err
	}()

	select {
	case err := <-done:
		require.Error(t, err)
		assert.Contains(t, err.Error(), "inspect server before registering")
	case <-time.After(5 * time.Second):
		t.Fatal("Register did not return after the probe deadline")
	}
}

func TestRegister_EmptySelectionSkipsProbe(t *testing.T) {
	var initializeCalls atomic.Int32
	server := newEmbeddingServer()
	server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method == "initialize" {
				initializeCalls.Add(1)
			}
			return next(ctx, method, req)
		}
	})

	// Nothing is selected, so there is no conflict to detect and no reason to
	// touch the host server.
	mustRegister(t, server, fixtureMFS(t), fixtureTokenConfig, "/test",
		WithTools(Selection{Exclude: allToolNames}),
		WithResources(Selection{Exclude: allResourceURIs}),
		WithTemplates(Selection{Exclude: allTemplateURIs}),
		WithConflictPolicy(ConflictError),
	)
	assert.Zero(t, initializeCalls.Load(), "empty selection must not probe the server")

	tools, resources, templates := connectAndList(t, server)
	assert.Empty(t, tools)
	assert.Empty(t, resources)
	assert.Empty(t, templates)
}

func TestRegister_ProbeListsOnlyNeededCategories(t *testing.T) {
	var resourceListCalls, templateListCalls atomic.Int32
	server := newEmbeddingServer()
	server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			switch method {
			case "resources/list":
				resourceListCalls.Add(1)
			case "resources/templates/list":
				templateListCalls.Add(1)
			}
			return next(ctx, method, req)
		}
	})

	// Only tools are selected, so the probe must not list resources.
	mustRegister(t, server, fixtureMFS(t), fixtureTokenConfig, "/test",
		WithTools(Selection{Include: []string{toolSearch}}),
		WithResources(Selection{Exclude: allResourceURIs}),
		WithTemplates(Selection{Exclude: allTemplateURIs}),
		WithConflictPolicy(ConflictSkip),
	)
	assert.Zero(t, resourceListCalls.Load(), "probe must not list unneeded resources")
	assert.Zero(t, templateListCalls.Load(), "probe must not list unneeded templates")

	session := connectSession(t, server)
	assert.Equal(t, []string{toolSearch}, listToolNames(t, context.Background(), session))
}

func TestRegister_ConflictProbeRunsHostHooksOnlyForNonDefaultPolicies(t *testing.T) {
	var initialized atomic.Int32
	newHookedServer := func() *mcp.Server {
		return mcp.NewServer(&mcp.Implementation{
			Name:    "hooked-server",
			Version: "1.0.0",
		}, &mcp.ServerOptions{
			InitializedHandler: func(context.Context, *mcp.InitializedRequest) {
				initialized.Add(1)
			},
		})
	}

	replaceServer := newHookedServer()
	mustRegister(t, replaceServer, fixtureMFS(t), fixtureTokenConfig, "/test")
	assert.Zero(t, initialized.Load(), "ConflictReplace must not probe the server")

	skipServer := newHookedServer()
	mustRegister(t, skipServer, fixtureMFS(t), fixtureTokenConfig, "/test", WithConflictPolicy(ConflictSkip))
	assert.Equal(t, int32(1), initialized.Load(), "ConflictSkip probes the server once")
}

func TestRegister_ConflictProbeWithLiveSession(t *testing.T) {
	ctx := context.Background()

	server := newEmbeddingServer()
	liveSession := connectSession(t, server)
	addExistingSearchTool(t, server, "existing-handler")

	mustRegister(t, server, fixtureMFS(t), fixtureTokenConfig, "/test", WithConflictPolicy(ConflictSkip))

	assert.Equal(t, allToolNames, listToolNames(t, ctx, liveSession))
	assert.Equal(t, "existing-handler", callExistingSearch(t, liveSession))
}
