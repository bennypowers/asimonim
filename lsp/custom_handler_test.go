package lsp

import (
	"bennypowers.dev/asimonim/lsp/types"
	"encoding/json"
	"testing"

	"bennypowers.dev/asimonim/lsp/internal/documents"
	"bennypowers.dev/asimonim/lsp/internal/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/bennypowers/glsp"
	protocol "github.com/bennypowers/glsp/protocol_3_17"
)

func TestCustomHandler_InitializeInterception(t *testing.T) {
	server := &Server{
		documents:   documents.NewManager(),
		tokens:      tokens.NewManager(),
		config:      types.ServerConfig{},
		loadedFiles: make(map[string]*TokenFileOptions),
	}

	t.Run("detects pull diagnostics capability from initialize params", func(t *testing.T) {
		initHandler := &protocol.Handler{}
		initHandler.Initialize = func(ctx *glsp.Context, params *protocol.InitializeParams) (any, error) {
			return protocol.InitializeResult{
				Capabilities: protocol.ServerCapabilities{},
			}, nil
		}

		h := &CustomHandler{
			Handler: initHandler,
			server:  server,
		}

		paramsJSON := []byte(`{
			"capabilities": {
				"textDocument": {
					"diagnostic": {"dynamicRegistration": true}
				}
			}
		}`)

		ctx := &glsp.Context{
			Method: "initialize",
			Params: paramsJSON,
		}

		result, validMethod, _, err := h.Handle(ctx)
		assert.True(t, validMethod)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		cap := server.ClientDiagnosticCapability()
		require.NotNil(t, cap, "pull diagnostics capability should be detected")
		assert.True(t, *cap, "pull diagnostics should be true")
	})

	t.Run("no pull diagnostics when client lacks capability", func(t *testing.T) {
		server.clientDiagnosticCapability = nil

		initHandler := &protocol.Handler{}
		initHandler.Initialize = func(ctx *glsp.Context, params *protocol.InitializeParams) (any, error) {
			return protocol.InitializeResult{
				Capabilities: protocol.ServerCapabilities{},
			}, nil
		}

		h := &CustomHandler{
			Handler: initHandler,
			server:  server,
		}

		paramsJSON := []byte(`{"capabilities": {"textDocument": {}}}`)

		ctx := &glsp.Context{
			Method: "initialize",
			Params: paramsJSON,
		}

		_, _, _, err := h.Handle(ctx)
		assert.NoError(t, err)

		cap := server.ClientDiagnosticCapability()
		require.NotNil(t, cap)
		assert.False(t, *cap, "pull diagnostics should be false without diagnostic capability")
	})

	t.Run("passes non-initialize methods through to base handler", func(t *testing.T) {
		handler := &protocol.Handler{}
		handler.SetInitialized(true)

		h := &CustomHandler{
			Handler: handler,
			server:  server,
		}

		params := protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
				Position:     protocol.Position{Line: 0, Character: 0},
			},
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		ctx := &glsp.Context{
			Method: "textDocument/hover",
			Params: paramsJSON,
		}

		_, validMethod, _, _ := h.Handle(ctx)
		assert.False(t, validMethod, "No hover handler set, so method should not be valid")
	})
}

func TestDetectPullDiagnosticsSupport(t *testing.T) {
	t.Run("returns true when diagnostic capability present", func(t *testing.T) {
		params := []byte(`{"capabilities": {"textDocument": {"diagnostic": {"dynamicRegistration": true}}}}`)
		assert.True(t, DetectPullDiagnosticsSupport(params))
	})

	t.Run("returns false when diagnostic capability absent", func(t *testing.T) {
		params := []byte(`{"capabilities": {"textDocument": {}}}`)
		assert.False(t, DetectPullDiagnosticsSupport(params))
	})

	t.Run("returns false for invalid JSON", func(t *testing.T) {
		assert.False(t, DetectPullDiagnosticsSupport([]byte(`{invalid`)))
	})

	t.Run("returns false when textDocument is null", func(t *testing.T) {
		params := []byte(`{"capabilities": {"textDocument": null}}`)
		assert.False(t, DetectPullDiagnosticsSupport(params))
	})
}
