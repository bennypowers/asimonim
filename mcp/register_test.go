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
	"fmt"
	"sort"
	"testing"

	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/fs"
	"bennypowers.dev/asimonim/internal/mapfs"
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

// normalizedTools returns each tool's full JSON definition keyed by name, so
// parity assertions cover schemas and descriptions, not just names.
func normalizedTools(t *testing.T, ctx context.Context, session *mcp.ClientSession) map[string]string {
	t.Helper()
	result, err := session.ListTools(ctx, nil)
	require.NoError(t, err)
	tools := make(map[string]string, len(result.Tools))
	for _, tool := range result.Tools {
		data, err := json.Marshal(tool)
		require.NoError(t, err)
		tools[tool.Name] = string(data)
	}
	return tools
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

// requiredToolInputs returns the sorted required input fields from a tool's
// marshaled JSON definition.
func requiredToolInputs(t *testing.T, toolJSON string) []string {
	t.Helper()
	var tool struct {
		InputSchema struct {
			Required []string `json:"required"`
		} `json:"inputSchema"`
	}
	require.NoError(t, json.Unmarshal([]byte(toolJSON), &tool))
	sort.Strings(tool.InputSchema.Required)
	return tool.InputSchema.Required
}

func newEmbeddingServer() *mcp.Server {
	return mcp.NewServer(&mcp.Implementation{
		Name:    "embedding-server",
		Version: "1.0.0",
	}, nil)
}

// mustRegister registers asimonim's surface on server and fails the test if
// registration returns an error.
func mustRegister(t *testing.T, server *mcp.Server, filesystem fs.FileSystem, cfg *config.Config, cwd string, opts ...RegisterOption) *Server {
	t.Helper()
	s, err := Register(server, filesystem, cfg, cwd, opts...)
	require.NoError(t, err)
	return s
}

func TestRegister_ToolParity(t *testing.T) {
	ctx := context.Background()

	baselineSession := connectSession(t, newTestServer(t, "fixtures/draft/simple").server)
	baseline := listToolNames(t, ctx, baselineSession)
	baselineTools := normalizedTools(t, ctx, baselineSession)

	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
	cfg := &config.Config{Files: []config.FileSpec{{Path: "/test/tokens.json"}}}
	server := newEmbeddingServer()
	mustRegister(t, server, mfs, cfg, "/test")
	session := connectSession(t, server)
	registered := listToolNames(t, ctx, session)
	registeredTools := normalizedTools(t, ctx, session)

	assert.Equal(t, baseline, registered)
	assert.Equal(t, baselineTools, registeredTools)
	assert.Equal(t, []string{"query"}, requiredToolInputs(t, registeredTools["search_tokens"]))
	assert.Equal(t, []string{"format"}, requiredToolInputs(t, registeredTools["convert_tokens"]))
	assert.Empty(t, requiredToolInputs(t, registeredTools["validate_tokens"]))
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
	mustRegister(t, server, mfs, cfg, "/test")
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
	mustRegister(t, server, mfs, cfg, "/test")
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
	mustRegister(t, server, mfs, cfg, "/test")
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

func TestRegister_ReadResourceTemplate(t *testing.T) {
	ctx := context.Background()

	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
	cfg := &config.Config{Files: []config.FileSpec{{Path: "/test/tokens.json"}}}
	server := newEmbeddingServer()
	mustRegister(t, server, mfs, cfg, "/test")
	session := connectSession(t, server)

	t.Run("tokens by source", func(t *testing.T) {
		result, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
			URI: "asimonim://tokens//test/tokens.json",
		})
		require.NoError(t, err)
		require.Len(t, result.Contents, 1)

		var tokens []tokenSummary
		require.NoError(t, json.Unmarshal([]byte(result.Contents[0].Text), &tokens))
		assert.Len(t, tokens, 5)
	})

	t.Run("token detail", func(t *testing.T) {
		result, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
			URI: "asimonim://token//test/tokens.json/color/primary",
		})
		require.NoError(t, err)
		require.Len(t, result.Contents, 1)

		var detail tokenDetail
		require.NoError(t, json.Unmarshal([]byte(result.Contents[0].Text), &detail))
		assert.Equal(t, "color-primary", detail.Name)
		assert.Equal(t, "color", detail.Type)
	})
}

func TestRegister_FallsBackToCwdWithoutRoots(t *testing.T) {
	ctx := context.Background()

	// No roots are advertised, so config resolution must fall back to cwd.
	server := newEmbeddingServer()
	mustRegister(t, server, cwdFS(), nil, "/mycwd")
	session := connectSession(t, server)

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search_tokens",
		Arguments: map[string]any{"query": "small"},
	})
	require.NoError(t, err)
	require.False(t, result.IsError, "unexpected tool error: %s", resultText(t, result))

	var tokens []tokenSummary
	require.NoError(t, json.Unmarshal([]byte(resultText(t, result)), &tokens))
	require.Len(t, tokens, 1)
	assert.Equal(t, "spacing-small", tokens[0].Name)
}

func TestRegister_RejectsNilServer(t *testing.T) {
	s, err := Register(nil, testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test"), nil, "/test")
	require.Error(t, err)
	assert.Nil(t, s)

	// The method form returns an error too, matching cem.
	err = (&Server{}).Register(nil)
	require.Error(t, err)
}

func TestRegister_RunHasNoTransport(t *testing.T) {
	server := newEmbeddingServer()
	s := mustRegister(t, server, testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test"), nil, "/test")
	require.NotNil(t, s)

	require.Error(t, s.Run(context.Background()))
}

func TestRegister_ResolvesConfigFromSessionRoots(t *testing.T) {
	ctx := context.Background()

	// cfg is nil and cwd does not contain a config, so the only way this
	// succeeds is by resolving roots through the embedding server's session.
	server := newEmbeddingServer()
	s := mustRegister(t, server, projectFS(), nil, "/elsewhere")
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

// twoProjectFS returns a filesystem with two independent workspaces.
func twoProjectFS() *mapfs.MapFileSystem {
	mfs := mapfs.New()
	projects := []struct {
		root  string
		name  string
		value string
	}{
		{"/project-a", "alpha", "#FF0000"},
		{"/project-b", "beta", "#0000FF"},
	}
	for _, project := range projects {
		mfs.AddFile(project.root+"/.config/design-tokens.yaml",
			"files:\n  - "+project.root+"/tokens.json\n", 0644)
		mfs.AddFile(project.root+"/tokens.json", fmt.Sprintf(`{
			"color": {
				"%s": {
					"$type": "color",
					"$value": "%s"
				}
			}
		}`, project.name, project.value), 0644)
	}
	return mfs
}

func TestRegister_ResolvesPerSessionRoots(t *testing.T) {
	ctx := context.Background()

	// A single embedding server serves two clients with different roots.
	// Each session must resolve its own workspace.
	server := newEmbeddingServer()
	mustRegister(t, server, twoProjectFS(), nil, "/elsewhere")

	sessionA := connectSession(t, server, &mcp.Root{URI: "file:///project-a"})
	sessionB := connectSession(t, server, &mcp.Root{URI: "file:///project-b"})

	assertSearchFinds(t, ctx, sessionA, "alpha")
	assertSearchFinds(t, ctx, sessionB, "beta")
}

func assertSearchFinds(t *testing.T, ctx context.Context, session *mcp.ClientSession, name string) {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search_tokens",
		Arguments: map[string]any{"query": name},
	})
	require.NoError(t, err)
	require.False(t, result.IsError, "unexpected tool error: %s", resultText(t, result))

	var tokens []tokenSummary
	require.NoError(t, json.Unmarshal([]byte(resultText(t, result)), &tokens))
	require.Len(t, tokens, 1)
	assert.Equal(t, "color-"+name, tokens[0].Name)
}
