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
	"encoding/json"
	"sort"
	"testing"

	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/testutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// connectSession connects a client to server over an in-memory transport.
// Roots, if any, are registered before connecting so the server sees them
// when it calls ListRoots.
func connectSession(t *testing.T, server *mcp.Server, roots ...*mcp.Root) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "register-test-client",
		Version: "1.0.0",
	}, nil)
	if len(roots) > 0 {
		client.AddRoots(roots...)
	}
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = clientSession.Close() })

	return clientSession
}

func listToolNames(t *testing.T, ctx context.Context, session *mcp.ClientSession) []string {
	t.Helper()
	result, err := session.ListTools(ctx, nil)
	require.NoError(t, err)
	names := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
	}
	sort.Strings(names)
	return names
}

func listResourceURIs(t *testing.T, ctx context.Context, session *mcp.ClientSession) []string {
	t.Helper()
	result, err := session.ListResources(ctx, nil)
	require.NoError(t, err)
	uris := make([]string, 0, len(result.Resources))
	for _, resource := range result.Resources {
		uris = append(uris, resource.URI)
	}
	sort.Strings(uris)
	return uris
}

func listResourceTemplateURIs(t *testing.T, ctx context.Context, session *mcp.ClientSession) []string {
	t.Helper()
	result, err := session.ListResourceTemplates(ctx, nil)
	require.NoError(t, err)
	uris := make([]string, 0, len(result.ResourceTemplates))
	for _, template := range result.ResourceTemplates {
		uris = append(uris, template.URITemplate)
	}
	sort.Strings(uris)
	return uris
}

func newEmbeddingServer() *mcp.Server {
	return mcp.NewServer(&mcp.Implementation{
		Name:    "embedding-server",
		Version: "1.0.0",
	}, nil)
}

func TestRegister_ToolParity(t *testing.T) {
	ctx := context.Background()

	baselineSession := connectSession(t, newTestServer(t, "fixtures/draft/simple").server)
	baseline := listToolNames(t, ctx, baselineSession)

	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
	cfg := &config.Config{Files: []config.FileSpec{{Path: "/test/tokens.json"}}}
	server := newEmbeddingServer()
	Register(server, mfs, cfg, "/test")
	registered := listToolNames(t, ctx, connectSession(t, server))

	assert.Equal(t, baseline, registered)
	assert.Equal(t, []string{"convert_tokens", "search_tokens", "validate_tokens"}, registered)
}

func TestRegister_ResourceParity(t *testing.T) {
	ctx := context.Background()

	baselineSession := connectSession(t, newTestServer(t, "fixtures/draft/simple").server)
	baselineResources := listResourceURIs(t, ctx, baselineSession)
	baselineTemplates := listResourceTemplateURIs(t, ctx, baselineSession)

	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
	cfg := &config.Config{Files: []config.FileSpec{{Path: "/test/tokens.json"}}}
	server := newEmbeddingServer()
	Register(server, mfs, cfg, "/test")
	session := connectSession(t, server)

	assert.Equal(t, baselineResources, listResourceURIs(t, ctx, session))
	assert.Equal(t, []string{"asimonim://config", "asimonim://tokens"}, listResourceURIs(t, ctx, session))
	assert.Equal(t, baselineTemplates, listResourceTemplateURIs(t, ctx, session))
	assert.Equal(t, []string{
		"asimonim://token/{+rest}",
		"asimonim://tokens/{+source}",
	}, listResourceTemplateURIs(t, ctx, session))
}

func TestRegister_CallSearchTool(t *testing.T) {
	ctx := context.Background()

	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
	cfg := &config.Config{Files: []config.FileSpec{{Path: "/test/tokens.json"}}}
	server := newEmbeddingServer()
	Register(server, mfs, cfg, "/test")
	session := connectSession(t, server)

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "search_tokens",
		Arguments: map[string]any{
			"query":     "primary",
			"name_only": true,
		},
	})
	require.NoError(t, err)
	require.False(t, result.IsError, "unexpected tool error: %s", resultText(t, result))

	var tokens []tokenSummary
	require.NoError(t, json.Unmarshal([]byte(resultText(t, result)), &tokens))
	require.Len(t, tokens, 1)
	assert.Equal(t, "color-primary", tokens[0].Name)
	assert.Equal(t, "#FF6B35", tokens[0].Value)
}

func TestRegister_ReadTokensResource(t *testing.T) {
	ctx := context.Background()

	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
	cfg := &config.Config{Files: []config.FileSpec{{Path: "/test/tokens.json"}}}
	server := newEmbeddingServer()
	Register(server, mfs, cfg, "/test")
	session := connectSession(t, server)

	result, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "asimonim://tokens"})
	require.NoError(t, err)
	require.Len(t, result.Contents, 1)
	assert.Equal(t, "asimonim://tokens", result.Contents[0].URI)

	var sources []sourceSummary
	require.NoError(t, json.Unmarshal([]byte(result.Contents[0].Text), &sources))
	require.Len(t, sources, 1)
	assert.Equal(t, "/test/tokens.json", sources[0].Source)
	assert.Equal(t, 5, sources[0].TokenCount)
}

func TestRegister_ResolvesConfigFromSessionRoots(t *testing.T) {
	ctx := context.Background()

	// cfg is nil and cwd does not contain a config, so the only way this
	// succeeds is by resolving roots through the embedding server's session.
	server := newEmbeddingServer()
	s := Register(server, projectFS(), nil, "/elsewhere")
	require.NotNil(t, s)
	session := connectSession(t, server, &mcp.Root{URI: "file:///project"})

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search_tokens",
		Arguments: map[string]any{"query": "primary"},
	})
	require.NoError(t, err)
	require.False(t, result.IsError, "unexpected tool error: %s", resultText(t, result))

	var tokens []tokenSummary
	require.NoError(t, json.Unmarshal([]byte(resultText(t, result)), &tokens))
	require.Len(t, tokens, 1)
	assert.Equal(t, "color-primary", tokens[0].Name)
}
