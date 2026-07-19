package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"bennypowers.dev/asimonim/lsp"
	"bennypowers.dev/asimonim/lsp/methods/lifecycle"
	"bennypowers.dev/asimonim/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// TestServerInitialization tests the full server initialization flow
func TestServerInitialization(t *testing.T) {
	t.Run("Initialize with workspace root", func(t *testing.T) {
		server, err := lsp.NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		// Create temp workspace
		tmpDir := t.TempDir()
		rootURI := uri.File(tmpDir)

		// Initialize server
		initParams := &protocol.InitializeParams{}
		initParams.RootURI = &rootURI
		initParams.RootPath = protocol.NewNullable(tmpDir)
		initParams.ClientInfo = protocol.ClientInfo{
			Name:    "test-client",
			Version: protocol.NewOptional("1.0.0"),
		}

		req := types.NewRequestContext(server, context.Background())
		result, err := lifecycle.Initialize(req, initParams)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify server info and capabilities
		assert.Equal(t, "design-tokens-language-server", result.ServerInfo.Name)
		assert.NotNil(t, result.Capabilities.TextDocumentSync)
		assert.NotNil(t, result.Capabilities.HoverProvider)
		assert.NotNil(t, result.Capabilities.CompletionProvider)
		assert.NotNil(t, result.Capabilities.DefinitionProvider)
		assert.NotNil(t, result.Capabilities.ReferencesProvider)
		assert.NotNil(t, result.Capabilities.CodeActionProvider)
		assert.NotNil(t, result.Capabilities.ColorProvider)
		assert.NotNil(t, result.Capabilities.InlayHintProvider)
		assert.NotNil(t, result.Capabilities.SemanticTokensProvider)
	})

	t.Run("Initialize without workspace root", func(t *testing.T) {
		server, err := lsp.NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		initParams := &protocol.InitializeParams{}
		initParams.ClientInfo = protocol.ClientInfo{
			Name: "test-client",
		}

		req := types.NewRequestContext(server, context.Background())
		result, err := lifecycle.Initialize(req, initParams)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("Load tokens from workspace configuration", func(t *testing.T) {
		server, err := lsp.NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		// Create temp workspace with token file
		tmpDir := t.TempDir()
		tokensPath := filepath.Join(tmpDir, "tokens.json")
		tokens := `{
  "color": {
    "primary": {
      "$value": "#0000ff",
      "$type": "color"
    }
  }
}`
		err = os.WriteFile(tokensPath, []byte(tokens), 0o644)
		require.NoError(t, err)

		// Load token file directly (simulating what Initialized would do)
		err = server.LoadTokenFile(tokensPath, "")
		require.NoError(t, err)

		// Verify tokens were loaded
		assert.Equal(t, 1, server.TokenCount(), "Should load tokens from file")
	})
}

// TestServerShutdown tests the shutdown flow
func TestServerShutdown(t *testing.T) {
	server, err := lsp.NewServer()
	require.NoError(t, err)

	// Shutdown should not error
	req := types.NewRequestContext(server, context.Background())
	err = lifecycle.Shutdown(req)
	assert.NoError(t, err)

	// Multiple shutdowns should be safe
	req = types.NewRequestContext(server, context.Background())
	err = lifecycle.Shutdown(req)
	assert.NoError(t, err)
}

// TestSetTrace tests the setTrace notification
func TestSetTrace(t *testing.T) {
	server, err := lsp.NewServer()
	require.NoError(t, err)
	defer func() { _ = server.Close() }()

	traces := []protocol.TraceValue{
		protocol.TraceValueOff,
		protocol.TraceValueMessages,
		protocol.TraceValueVerbose,
	}
	for _, trace := range traces {
		t.Run(string(trace), func(t *testing.T) {
			req := types.NewRequestContext(server, context.Background())
			err := lifecycle.SetTrace(req, &protocol.SetTraceParams{
				Value: trace,
			})
			assert.NoError(t, err)
		})
	}
}

