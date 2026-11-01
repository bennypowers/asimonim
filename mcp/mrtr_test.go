/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

//lint:file-ignore SA1019 roots/list is deprecated by SEP-2577 but still supported

//nolint:staticcheck // roots/list is deprecated by SEP-2577 but still supported
package mcp

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"

	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/internal/mapfs"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Workspace provenance constants mirrored here so the tests fail to compile
// only for genuinely missing API, not for renamed internal identifiers.
const (
	testWorkspaceMetaKey  = "asimonim/workspace"
	testInputRequestRoots = "asimonim/roots"

	testSourceExplicitConfig = "explicit-config"
	testSourceHost           = "host"
	testSourceRoots          = "roots"
	testSourceCwd            = "cwd"

	testProtocol20250618 = "2025-06-18"
	testProtocol20251125 = "2025-11-25"
	testProtocol20260728 = "2026-07-28"
)

var testProtocols = []string{testProtocol20250618, testProtocol20251125, testProtocol20260728}

// clientSetup describes the client half of a protocol matrix row.
type clientSetup struct {
	protocol       string
	roots          []*mcp.Root
	advertiseRoots bool
	disableMRTR    bool
}

// rootsCountingTransport counts roots/list requests arriving at the client.
// On legacy revisions the server issues a real roots/list request. On
// 2026-07-28 the client fulfills the request locally from its own roots list,
// so this counter stays at zero there.
type rootsCountingTransport struct {
	inner mcp.Transport
	calls *atomic.Int32
}

func (t *rootsCountingTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	conn, err := t.inner.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &rootsCountingConn{Connection: conn, calls: t.calls}, nil
}

type rootsCountingConn struct {
	mcp.Connection
	calls *atomic.Int32
}

func (c *rootsCountingConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	msg, err := c.Connection.Read(ctx)
	if req, ok := msg.(*jsonrpc.Request); ok && req.Method == "roots/list" {
		c.calls.Add(1)
	}
	return msg, err
}

// connectMatrixClient connects a client to server using an explicit protocol
// version and returns the session plus a counter of roots/list requests
// observed on the wire.
func connectMatrixClient(t *testing.T, server *mcp.Server, setup clientSetup) (*mcp.ClientSession, *atomic.Int32) {
	t.Helper()
	ctx := context.Background()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })

	var calls atomic.Int32
	counting := &rootsCountingTransport{inner: clientTransport, calls: &calls}

	opts := &mcp.ClientOptions{}
	if !setup.advertiseRoots {
		opts.Capabilities = &mcp.ClientCapabilities{}
	}
	if setup.disableMRTR {
		opts.MultiRoundTrip = &mcp.MultiRoundTripOptions{Disabled: true}
	}

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "mrtr-test-client",
		Version: "1.0.0",
	}, opts)
	for _, root := range setup.roots {
		client.AddRoots(root)
	}

	clientSession, err := client.Connect(ctx, counting, &mcp.ClientSessionOptions{ProtocolVersion: setup.protocol})
	require.NoError(t, err)
	t.Cleanup(func() { _ = clientSession.Close() })

	return clientSession, &calls
}

func workspaceMeta(t *testing.T, meta mcp.Meta) map[string]any {
	t.Helper()
	raw, ok := meta.GetMeta()[testWorkspaceMetaKey]
	require.True(t, ok, "result is missing %q in _meta: %#v", testWorkspaceMetaKey, meta.GetMeta())
	value, ok := raw.(map[string]any)
	require.True(t, ok, "%q should be a JSON object, got %T", testWorkspaceMetaKey, raw)
	return value
}

func callSearch(t *testing.T, ctx context.Context, session *mcp.ClientSession, query string) *mcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search_tokens",
		Arguments: map[string]any{"query": query},
	})
	require.NoError(t, err)
	return result
}

func searchNames(t *testing.T, result *mcp.CallToolResult) []string {
	t.Helper()
	require.False(t, result.IsError, "unexpected tool error: %s", resultText(t, result))
	var tokens []tokenSummary
	require.NoError(t, json.Unmarshal([]byte(resultText(t, result)), &tokens))
	names := make([]string, len(tokens))
	for i, tok := range tokens {
		names[i] = tok.Name
	}
	return names
}

// cwdWithProjectFS adds a cwd workspace to a filesystem that also has
// /project, so a test can distinguish roots resolution from the cwd fallback.
func cwdWithProjectFS() *mapfs.MapFileSystem {
	mfs := projectFS()
	mfs.AddFile("/mycwd/.config/design-tokens.yaml", "files:\n  - /mycwd/tokens.json\n", 0644)
	mfs.AddFile("/mycwd/tokens.json", `{
		"spacing": {
			"small": {
				"$type": "dimension",
				"$value": "4px"
			}
		}
	}`, 0644)
	return mfs
}

func TestRootsProtocolMatrix(t *testing.T) {
	ctx := context.Background()

	for _, protocol := range testProtocols {
		t.Run(protocol, func(t *testing.T) {
			t.Run("capability absent", func(t *testing.T) {
				server := newEmbeddingServer()
				mustRegister(t, server, cwdFS(), nil, "/mycwd")
				session, rootsCalls := connectMatrixClient(t, server, clientSetup{
					protocol:       protocol,
					advertiseRoots: false,
				})

				result := callSearch(t, ctx, session, "small")
				require.False(t, result.NeedsInput(), "must not request input when the client does not advertise roots")
				assert.Equal(t, []string{"spacing-small"}, searchNames(t, result))

				meta := workspaceMeta(t, result.Meta)
				assert.Equal(t, testSourceCwd, meta["source"])
				assert.Equal(t, "/mycwd", meta["root"])
				assert.NotContains(t, meta, "warning")
				assert.Zero(t, rootsCalls.Load(), "no roots/list traffic without the capability")
			})

			t.Run("capability present with a root", func(t *testing.T) {
				server := newEmbeddingServer()
				mustRegister(t, server, projectFS(), nil, "/elsewhere")
				session, _ := connectMatrixClient(t, server, clientSetup{
					protocol:       protocol,
					advertiseRoots: true,
					roots:          []*mcp.Root{{URI: "file:///project"}},
				})

				result := callSearch(t, ctx, session, "primary")
				assert.Equal(t, []string{"color-primary"}, searchNames(t, result))

				meta := workspaceMeta(t, result.Meta)
				assert.Equal(t, testSourceRoots, meta["source"])
				assert.Equal(t, "/project", meta["root"])
				assert.NotContains(t, meta, "warning")
			})

			t.Run("capability present with empty roots", func(t *testing.T) {
				server := newEmbeddingServer()
				mustRegister(t, server, cwdFS(), nil, "/mycwd")
				session, _ := connectMatrixClient(t, server, clientSetup{
					protocol:       protocol,
					advertiseRoots: true,
				})

				result := callSearch(t, ctx, session, "small")
				assert.Equal(t, []string{"spacing-small"}, searchNames(t, result))

				meta := workspaceMeta(t, result.Meta)
				assert.Equal(t, testSourceCwd, meta["source"])
				assert.Equal(t, "/mycwd", meta["root"])
				warning, _ := meta["warning"].(string)
				assert.NotEmpty(t, warning, "empty roots must warn rather than fall back silently")
			})
		})
	}
}

func TestRootsLegacyWireTraffic(t *testing.T) {
	ctx := context.Background()

	for _, protocol := range []string{testProtocol20250618, testProtocol20251125} {
		t.Run(protocol, func(t *testing.T) {
			server := newEmbeddingServer()
			mustRegister(t, server, projectFS(), nil, "/elsewhere")
			session, rootsCalls := connectMatrixClient(t, server, clientSetup{
				protocol:       protocol,
				advertiseRoots: true,
				roots:          []*mcp.Root{{URI: "file:///project"}},
			})

			assert.Equal(t, []string{"color-primary"}, searchNames(t, callSearch(t, ctx, session, "primary")))
			assert.EqualValues(t, 1, rootsCalls.Load(), "legacy revisions resolve roots with exactly one roots/list request")
		})
	}
}

func TestMRTRMalformedRetryFallsBackToCwd(t *testing.T) {
	ctx := context.Background()

	t.Run("missing response", func(t *testing.T) {
		server := newEmbeddingServer()
		mustRegister(t, server, cwdWithProjectFS(), nil, "/mycwd")
		session, _ := connectMatrixClient(t, server, clientSetup{
			protocol:       testProtocol20260728,
			advertiseRoots: true,
			roots:          []*mcp.Root{{URI: "file:///project"}},
			disableMRTR:    true,
		})

		first := callSearch(t, ctx, session, "small")
		require.True(t, first.NeedsInput(), "first call should ask for roots")
		require.Contains(t, first.InputRequests, testInputRequestRoots)

		retry, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:           "search_tokens",
			Arguments:      map[string]any{"query": "small"},
			InputResponses: mcp.InputResponseMap{},
			RequestState:   first.RequestState,
		})
		require.NoError(t, err)
		require.False(t, retry.NeedsInput(), "a malformed retry must not loop")
		assert.Equal(t, []string{"spacing-small"}, searchNames(t, retry))

		meta := workspaceMeta(t, retry.Meta)
		assert.Equal(t, testSourceCwd, meta["source"])
		assert.Equal(t, "/mycwd", meta["root"])
		assert.NotEmpty(t, meta["warning"])

		again := callSearch(t, ctx, session, "small")
		require.False(t, again.NeedsInput(), "resolution is cached; no second retry")
	})

	t.Run("wrong response type", func(t *testing.T) {
		server := newEmbeddingServer()
		mustRegister(t, server, cwdWithProjectFS(), nil, "/mycwd")
		session, _ := connectMatrixClient(t, server, clientSetup{
			protocol:       testProtocol20260728,
			advertiseRoots: true,
			roots:          []*mcp.Root{{URI: "file:///project"}},
			disableMRTR:    true,
		})

		first := callSearch(t, ctx, session, "small")
		require.True(t, first.NeedsInput())

		retry, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "search_tokens",
			Arguments: map[string]any{"query": "small"},
			InputResponses: mcp.InputResponseMap{
				testInputRequestRoots: &mcp.ElicitResult{Action: "accept"},
			},
			RequestState: first.RequestState,
		})
		require.NoError(t, err)
		require.False(t, retry.NeedsInput(), "a wrong-typed retry must not loop")
		assert.Equal(t, []string{"spacing-small"}, searchNames(t, retry))

		meta := workspaceMeta(t, retry.Meta)
		assert.Equal(t, testSourceCwd, meta["source"])
		assert.NotEmpty(t, meta["warning"])
	})
}

func TestMRTRResourceReadUsesRoots(t *testing.T) {
	ctx := context.Background()

	server := newEmbeddingServer()
	mustRegister(t, server, projectFS(), nil, "/elsewhere")
	session, _ := connectMatrixClient(t, server, clientSetup{
		protocol:       testProtocol20260728,
		advertiseRoots: true,
		roots:          []*mcp.Root{{URI: "file:///project"}},
	})

	result, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "asimonim://config"})
	require.NoError(t, err)
	require.False(t, result.NeedsInput())
	require.Len(t, result.Contents, 1)
	assert.Contains(t, result.Contents[0].Text, "/project/tokens.json")

	meta := workspaceMeta(t, result.Meta)
	assert.Equal(t, testSourceRoots, meta["source"])
	assert.Equal(t, "/project", meta["root"])
}

func TestMRTRResourceMalformedRetryFallsBackToCwd(t *testing.T) {
	ctx := context.Background()

	server := newEmbeddingServer()
	mustRegister(t, server, cwdWithProjectFS(), nil, "/mycwd")
	session, _ := connectMatrixClient(t, server, clientSetup{
		protocol:       testProtocol20260728,
		advertiseRoots: true,
		roots:          []*mcp.Root{{URI: "file:///project"}},
		disableMRTR:    true,
	})

	first, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "asimonim://config"})
	require.NoError(t, err)
	require.True(t, first.NeedsInput(), "resource reads must request roots on 2026-07-28")

	retry, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
		URI:            "asimonim://config",
		InputResponses: mcp.InputResponseMap{},
		RequestState:   first.RequestState,
	})
	require.NoError(t, err)
	require.False(t, retry.NeedsInput(), "a malformed resource retry must not loop")
	require.Len(t, retry.Contents, 1)
	assert.Contains(t, retry.Contents[0].Text, "/mycwd/tokens.json")

	meta := workspaceMeta(t, retry.Meta)
	assert.Equal(t, testSourceCwd, meta["source"])
	assert.NotEmpty(t, meta["warning"])
}

func TestMRTRTwoSessionsDifferentRoots(t *testing.T) {
	ctx := context.Background()

	server := newEmbeddingServer()
	mustRegister(t, server, twoProjectFS(), nil, "/elsewhere")

	sessionA, _ := connectMatrixClient(t, server, clientSetup{
		protocol:       testProtocol20260728,
		advertiseRoots: true,
		roots:          []*mcp.Root{{URI: "file:///project-a"}},
	})
	sessionB, _ := connectMatrixClient(t, server, clientSetup{
		protocol:       testProtocol20260728,
		advertiseRoots: true,
		roots:          []*mcp.Root{{URI: "file:///project-b"}},
	})

	resultA := callSearch(t, ctx, sessionA, "alpha")
	assert.Equal(t, []string{"color-alpha"}, searchNames(t, resultA))
	assert.Equal(t, "/project-a", workspaceMeta(t, resultA.Meta)["root"])

	resultB := callSearch(t, ctx, sessionB, "beta")
	assert.Equal(t, []string{"color-beta"}, searchNames(t, resultB))
	assert.Equal(t, "/project-b", workspaceMeta(t, resultB.Meta)["root"])
}

func TestMRTRNewServerAndRegisterParity(t *testing.T) {
	ctx := context.Background()

	t.Run("NewServer", func(t *testing.T) {
		s := NewServer(projectFS(), nil, "/elsewhere")
		session, _ := connectMatrixClient(t, s.server, clientSetup{
			protocol:       testProtocol20260728,
			advertiseRoots: true,
			roots:          []*mcp.Root{{URI: "file:///project"}},
		})

		result := callSearch(t, ctx, session, "primary")
		assert.Equal(t, []string{"color-primary"}, searchNames(t, result))
		meta := workspaceMeta(t, result.Meta)
		assert.Equal(t, testSourceRoots, meta["source"])
		assert.Equal(t, "/project", meta["root"])
	})

	t.Run("Register", func(t *testing.T) {
		server := newEmbeddingServer()
		mustRegister(t, server, projectFS(), nil, "/elsewhere")
		session, _ := connectMatrixClient(t, server, clientSetup{
			protocol:       testProtocol20260728,
			advertiseRoots: true,
			roots:          []*mcp.Root{{URI: "file:///project"}},
		})

		result := callSearch(t, ctx, session, "primary")
		assert.Equal(t, []string{"color-primary"}, searchNames(t, result))
		meta := workspaceMeta(t, result.Meta)
		assert.Equal(t, testSourceRoots, meta["source"])
		assert.Equal(t, "/project", meta["root"])
	})
}

func TestMRTROneRoundTripPerSession(t *testing.T) {
	ctx := context.Background()

	server := newEmbeddingServer()
	mustRegister(t, server, projectFS(), nil, "/elsewhere")
	session, _ := connectMatrixClient(t, server, clientSetup{
		protocol:       testProtocol20260728,
		advertiseRoots: true,
		roots:          []*mcp.Root{{URI: "file:///project"}},
		disableMRTR:    true,
	})

	first := callSearch(t, ctx, session, "primary")
	require.True(t, first.NeedsInput(), "the first call performs the only roots round trip")

	retry, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search_tokens",
		Arguments: map[string]any{"query": "primary"},
		InputResponses: mcp.InputResponseMap{
			testInputRequestRoots: &mcp.ListRootsResult{Roots: []*mcp.Root{{URI: "file:///project"}}},
		},
		RequestState: first.RequestState,
	})
	require.NoError(t, err)
	require.False(t, retry.NeedsInput())
	assert.Equal(t, []string{"color-primary"}, searchNames(t, retry))

	validate, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "validate_tokens",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	require.False(t, validate.NeedsInput())

	convert, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "convert_tokens",
		Arguments: map[string]any{"format": "css"},
	})
	require.NoError(t, err)
	require.False(t, convert.NeedsInput())

	resource, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "asimonim://config"})
	require.NoError(t, err)
	require.False(t, resource.NeedsInput())

	for _, result := range []*mcp.CallToolResult{retry, validate, convert} {
		assert.Equal(t, testSourceRoots, workspaceMeta(t, result.Meta)["source"])
	}
	assert.Equal(t, testSourceRoots, workspaceMeta(t, resource.Meta)["source"])
}

func TestWithWorkspaceWinsOverClientRoots(t *testing.T) {
	ctx := context.Background()

	mfs := twoProjectFS()
	mfs.AddFile("/mycwd/.config/design-tokens.yaml", "files:\n  - /mycwd/tokens.json\n", 0644)
	mfs.AddFile("/mycwd/tokens.json", `{
		"cwd": {
			"token": {
				"$type": "color",
				"$value": "#00FF00"
			}
		}
	}`, 0644)

	for _, protocol := range testProtocols {
		t.Run(protocol, func(t *testing.T) {
			server := newEmbeddingServer()
			mustRegister(t, server, mfs, nil, "/mycwd", WithWorkspace("/project-a"))
			session, rootsCalls := connectMatrixClient(t, server, clientSetup{
				protocol:       protocol,
				advertiseRoots: true,
				roots:          []*mcp.Root{{URI: "file:///project-b"}},
				disableMRTR:    true,
			})

			result := callSearch(t, ctx, session, "alpha")
			require.False(t, result.NeedsInput(), "host workspace must not require a roots round trip")
			assert.Equal(t, []string{"color-alpha"}, searchNames(t, result))
			meta := workspaceMeta(t, result.Meta)
			assert.Equal(t, testSourceHost, meta["source"])
			assert.Equal(t, "/project-a", meta["root"])
			assert.Zero(t, rootsCalls.Load(), "host workspace must not query roots")
		})
	}
}

func TestWithWorkspaceRejectsExplicitConfig(t *testing.T) {
	server := newEmbeddingServer()
	cfg := &config.Config{Files: []config.FileSpec{{Path: "/project/tokens.json"}}}

	_, err := Register(server, projectFS(), cfg, "/elsewhere", WithWorkspace("/project"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WithWorkspace")

	_, err = Register(server, projectFS(), nil, "/elsewhere", WithWorkspace("  "))
	require.Error(t, err)
}

func TestExplicitConfigMeta(t *testing.T) {
	ctx := context.Background()

	server := newEmbeddingServer()
	cfg := &config.Config{Files: []config.FileSpec{{Path: "/project/tokens.json"}}}
	mustRegister(t, server, projectFS(), cfg, "/elsewhere")
	session, rootsCalls := connectMatrixClient(t, server, clientSetup{
		protocol:       testProtocol20260728,
		advertiseRoots: true,
		roots:          []*mcp.Root{{URI: "file:///project"}},
		disableMRTR:    true,
	})

	result := callSearch(t, ctx, session, "primary")
	require.False(t, result.NeedsInput())
	assert.Equal(t, []string{"color-primary"}, searchNames(t, result))
	meta := workspaceMeta(t, result.Meta)
	assert.Equal(t, testSourceExplicitConfig, meta["source"])
	assert.Equal(t, "/elsewhere", meta["root"])
	assert.Zero(t, rootsCalls.Load())
}

func TestRootsResolutionWarningWithoutSession(t *testing.T) {
	mfs := cwdFS()
	s := NewServer(mfs, nil, "/mycwd")
	s.listRoots = func(_ context.Context) ([]*mcp.Root, error) {
		return nil, assert.AnError
	}

	ws := s.workspaceForRequest(context.Background(), nil)
	assert.Equal(t, testSourceCwd, ws.source)
	assert.Equal(t, "/mycwd", ws.rootDir)
	assert.Contains(t, ws.warning, assert.AnError.Error())
}

func TestWorkspaceMetaWarningIsOmittedWhenEmpty(t *testing.T) {
	ws := workspace{rootDir: "/project", source: testSourceRoots}
	meta := ws.meta()
	assert.Equal(t, "/project", meta["root"])
	assert.Equal(t, testSourceRoots, meta["source"])
	_, hasWarning := meta["warning"]
	assert.False(t, hasWarning)
}
