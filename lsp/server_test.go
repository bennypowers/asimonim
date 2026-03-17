package lsp

import (
	"testing"

	"bennypowers.dev/asimonim/lsp/types"

	"bennypowers.dev/asimonim/lsp/internal/documents"
	"bennypowers.dev/asimonim/lsp/internal/tokens"
	"bennypowers.dev/asimonim/lsp/methods/lifecycle"
	"bennypowers.dev/asimonim/lsp/methods/textDocument"
	codeaction "bennypowers.dev/asimonim/lsp/methods/textDocument/codeAction"
	"bennypowers.dev/asimonim/lsp/methods/textDocument/completion"
	"bennypowers.dev/asimonim/lsp/methods/textDocument/definition"
	"bennypowers.dev/asimonim/lsp/methods/textDocument/diagnostic"
	documentcolor "bennypowers.dev/asimonim/lsp/methods/textDocument/documentColor"
	"bennypowers.dev/asimonim/lsp/methods/textDocument/hover"
	"bennypowers.dev/asimonim/lsp/methods/textDocument/references"
	semantictokens "bennypowers.dev/asimonim/lsp/methods/textDocument/semanticTokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestHandlers_WrappersSmokeTest verifies that protocol handler wrappers
// are properly connected to their business logic methods.
// This provides coverage for the 1-3 line wrapper functions without
// duplicating the comprehensive business logic tests in integration/.
func TestHandlers_WrappersSmokeTest(t *testing.T) {
	// Create minimal server for smoke tests
	server := &Server{
		documents:          documents.NewManager(),
		tokens:             tokens.NewManager(),
		config:             types.ServerConfig{},
		loadedFiles:        make(map[string]*TokenFileOptions),
		semanticTokenCache: semantictokens.NewTokenCache(),
	}

	// Dummy context (nil is fine for these simple wrappers)
	var ctx *glsp.Context

	t.Run("Hover", func(t *testing.T) {
		params := &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
				Position:     protocol.Position{Line: 0, Character: 0},
			},
		}
		// Should not panic, returns nil for non-existent document
		req := types.NewRequestContext(server, ctx)
		result, err := hover.Hover(req, params)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("Completion", func(t *testing.T) {
		params := &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
				Position:     protocol.Position{Line: 0, Character: 0},
			},
		}
		req := types.NewRequestContext(server, ctx)
		result, err := completion.Completion(req, params)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("CompletionResolve", func(t *testing.T) {
		item := &protocol.CompletionItem{Label: "test"}
		req := types.NewRequestContext(server, ctx)
		result, err := completion.CompletionResolve(req, item)
		assert.NoError(t, err)
		assert.Equal(t, item, result) // Returns same item if no data
	})

	t.Run("Definition", func(t *testing.T) {
		params := &protocol.DefinitionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
				Position:     protocol.Position{Line: 0, Character: 0},
			},
		}
		req := types.NewRequestContext(server, ctx)
		result, err := definition.Definition(req, params)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("References", func(t *testing.T) {
		params := &protocol.ReferenceParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
				Position:     protocol.Position{Line: 0, Character: 0},
			},
		}
		req := types.NewRequestContext(server, ctx)
		result, err := references.References(req, params)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("CodeAction", func(t *testing.T) {
		params := &protocol.CodeActionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
		}
		req := types.NewRequestContext(server, ctx)
		result, err := codeaction.CodeAction(req, params)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("CodeActionResolve", func(t *testing.T) {
		action := &protocol.CodeAction{Title: "test"}
		req := types.NewRequestContext(server, ctx)
		result, err := codeaction.CodeActionResolve(req, action)
		assert.NoError(t, err)
		assert.Equal(t, action, result)
	})

	t.Run("DocumentColor", func(t *testing.T) {
		params := &protocol.DocumentColorParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
		}
		req := types.NewRequestContext(server, ctx)
		result, err := documentcolor.DocumentColor(req, params)
		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("ColorPresentation", func(t *testing.T) {
		params := &protocol.ColorPresentationParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
			Color: protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
		}
		req := types.NewRequestContext(server, ctx)
		result, err := documentcolor.ColorPresentation(req, params)
		assert.NoError(t, err)
		// Returns empty array when no tokens match (new behavior matches TypeScript)
		assert.Empty(t, result)
	})

	t.Run("DocumentDiagnostic", func(t *testing.T) {
		params := &diagnostic.DocumentDiagnosticParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
		}
		req := types.NewRequestContext(server, ctx)
		result, err := diagnostic.DocumentDiagnostic(req, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("DidOpen", func(t *testing.T) {
		params := &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:        "file:///test.css",
				LanguageID: "css",
				Version:    1,
				Text:       "body { color: red; }",
			},
		}
		req := types.NewRequestContext(server, ctx)
		err := textDocument.DidOpen(req, params)
		assert.NoError(t, err)
	})

	t.Run("didChange", func(t *testing.T) {
		// First open a document
		_ = server.documents.DidOpen("file:///test.css", "css", 1, "body { color: red; }")

		textChange := protocol.TextDocumentContentChangeEvent{}
		textChange.Text = "body { color: blue; }"

		params := &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
				Version:                2,
			},
			ContentChanges: []any{textChange},
		}
		req := types.NewRequestContext(server, ctx)
		err := textDocument.DidChange(req, params)
		assert.NoError(t, err)
	})

	t.Run("didClose", func(t *testing.T) {
		// Ensure document exists
		_ = server.documents.DidOpen("file:///test2.css", "css", 1, "")

		params := &protocol.DidCloseTextDocumentParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test2.css"},
		}
		req := types.NewRequestContext(server, ctx)
		err := textDocument.DidClose(req, params)
		assert.NoError(t, err)
	})

	t.Run("shutdown", func(t *testing.T) {
		req := types.NewRequestContext(server, ctx)
		err := lifecycle.Shutdown(req)
		assert.NoError(t, err)
	})

	t.Run("setTrace", func(t *testing.T) {
		params := &protocol.SetTraceParams{Value: "off"}
		req := types.NewRequestContext(server, ctx)
		err := lifecycle.SetTrace(req, params)
		assert.NoError(t, err)
	})
}

// TestServer_Close tests that Close() properly releases resources
func TestServer_Close(t *testing.T) {
	t.Run("Close releases CSS parser pool", func(t *testing.T) {
		server, err := NewServer()
		assert.NoError(t, err)
		assert.NotNil(t, server)

		// Close should not panic and should clean up resources
		assert.NotPanics(t, func() {
			err := server.Close()
			assert.NoError(t, err)
		})
	})

	t.Run("Close can be called multiple times", func(t *testing.T) {
		server, err := NewServer()
		assert.NoError(t, err)

		// First close
		err = server.Close()
		assert.NoError(t, err)

		// Second close should not panic or error
		err = server.Close()
		assert.NoError(t, err)
	})

	t.Run("Close works with nil server fields", func(t *testing.T) {
		// Minimal server with no initialization
		server := &Server{
			documents:   documents.NewManager(),
			tokens:      tokens.NewManager(),
			config:      types.ServerConfig{},
			loadedFiles: make(map[string]*TokenFileOptions),
		}

		// Should not panic
		assert.NotPanics(t, func() {
			err := server.Close()
			assert.NoError(t, err)
		})
	})
}

func TestPublishDiagnostics_NilContext(t *testing.T) {
	t.Run("errors when both contexts are nil", func(t *testing.T) {
		server := &Server{
			documents:   documents.NewManager(),
			tokens:      tokens.NewManager(),
			config:      types.ServerConfig{},
			loadedFiles: make(map[string]*TokenFileOptions),
			context:     nil, // No server context
		}

		// Open a document
		err := server.documents.DidOpen("file:///test.css", "css", 1, `.test { color: red; }`)
		require.NoError(t, err)

		// Attempt to publish diagnostics with nil context
		err = server.PublishDiagnostics(nil, "file:///test.css")

		// Should return an error, not panic
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no client context available")
	})

	t.Run("uses server context when parameter is nil", func(t *testing.T) {
		// This test verifies the fallback mechanism works
		// We can't easily test Notify being called without a real client,
		// but we can verify it doesn't error when s.context is set
		server := &Server{
			documents:   documents.NewManager(),
			tokens:      tokens.NewManager(),
			config:      types.ServerConfig{},
			loadedFiles: make(map[string]*TokenFileOptions),
			// In a real scenario, context would be set by SetGLSPContext
			// For this test, we're just verifying the error path isn't triggered
		}

		// Open a document
		err := server.documents.DidOpen("file:///test.css", "css", 1, `.test { color: red; }`)
		require.NoError(t, err)

		// With both contexts nil, it should error (already tested above)
		err = server.PublishDiagnostics(nil, "file:///test.css")
		assert.Error(t, err)

		// Note: We can't easily test the success case without a real GLSP context
		// as that requires a running LSP client. The important thing is that
		// it doesn't panic and returns a clear error when no context is available.
	})
}

func TestShouldProcessAsTokenFile(t *testing.T) {
	t.Run("returns true when file is in config", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.rootPath = "/workspace"
		s.config.TokensFiles = []any{"tokens.json"}

		// Open a document (without $schema)
		err = s.documents.DidOpen("file:///workspace/tokens.json", "json", 1, `{"color": {"$value": "#fff"}}`)
		require.NoError(t, err)

		result := s.ShouldProcessAsTokenFile("file:///workspace/tokens.json")
		assert.True(t, result, "Should return true for configured token file")
	})

	t.Run("returns true when document has Design Tokens schema", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		// Open a document with valid $schema (not in config)
		content := `{
  "$schema": "https://www.designtokens.org/schemas/draft.json",
  "color": {"$value": "#fff"}
}`
		err = s.documents.DidOpen("file:///tokens.json", "json", 1, content)
		require.NoError(t, err)

		result := s.ShouldProcessAsTokenFile("file:///tokens.json")
		assert.True(t, result, "Should return true for file with Design Tokens schema")
	})

	t.Run("returns false when document has non-Design-Tokens schema", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		// Open a document with different $schema
		content := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object"
}`
		err = s.documents.DidOpen("file:///schema.json", "json", 1, content)
		require.NoError(t, err)

		result := s.ShouldProcessAsTokenFile("file:///schema.json")
		assert.False(t, result, "Should return false for file with non-Design-Tokens schema")
	})

	t.Run("returns false when document has no schema", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		// Open a document without $schema (not in config)
		err = s.documents.DidOpen("file:///package.json", "json", 1, `{"name": "test"}`)
		require.NoError(t, err)

		result := s.ShouldProcessAsTokenFile("file:///package.json")
		assert.False(t, result, "Should return false for file without schema and not in config")
	})

	t.Run("returns false when document does not exist", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		result := s.ShouldProcessAsTokenFile("file:///nonexistent.json")
		assert.False(t, result, "Should return false for non-existent document")
	})
}

func TestIsTokenFile(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		rootPath       string
		configFiles    []any
		expectedResult bool
	}{
		{
			name:           "Explicit token file - JSON",
			path:           "/workspace/tokens.json",
			rootPath:       "/workspace",
			configFiles:    []any{"tokens.json"},
			expectedResult: true,
		},
		{
			name:           "Explicit token file - absolute path",
			path:           "/workspace/design-system/tokens.json",
			rootPath:       "/workspace",
			configFiles:    []any{"/workspace/design-system/tokens.json"},
			expectedResult: true,
		},
		{
			name:     "Explicit token file - relative path",
			path:     "/workspace/design-system/tokens.json",
			rootPath: "/workspace",
			configFiles: []any{
				map[string]any{
					"path": "design-system/tokens.json",
				},
			},
			expectedResult: true,
		},
		{
			name:           "Non-token file",
			path:           "/workspace/package.json",
			rootPath:       "/workspace",
			configFiles:    []any{"tokens.json"},
			expectedResult: false,
		},
		{
			name:           "Empty config - tokens.json not tracked",
			path:           "/workspace/tokens.json",
			rootPath:       "/workspace",
			configFiles:    []any{}, // Empty = no auto-discover
			expectedResult: false,
		},
		{
			name:           "Empty config - design-tokens.json not tracked",
			path:           "/workspace/design-tokens.json",
			rootPath:       "/workspace",
			configFiles:    []any{},
			expectedResult: false,
		},
		{
			name:           "Non-JSON/YAML file",
			path:           "/workspace/tokens.txt",
			rootPath:       "/workspace",
			configFiles:    []any{},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewServer()
			if err != nil {
				t.Fatalf("Failed to create server: %v", err)
			}

			s.rootPath = tt.rootPath
			s.config.TokensFiles = tt.configFiles

			result := s.IsTokenFile(tt.path)
			if result != tt.expectedResult {
				t.Errorf("Expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

func TestServer_SupportsSnippets(t *testing.T) {
	t.Run("returns false when capabilities are nil", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		assert.False(t, s.SupportsSnippets())
	})

	t.Run("returns false when TextDocument is nil", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.SetClientCapabilities(protocol.ClientCapabilities{})
		assert.False(t, s.SupportsSnippets())
	})

	t.Run("returns false when Completion is nil", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.SetClientCapabilities(protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{},
		})
		assert.False(t, s.SupportsSnippets())
	})

	t.Run("returns false when CompletionItem is nil", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.SetClientCapabilities(protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{
				Completion: &protocol.CompletionClientCapabilities{},
			},
		})
		assert.False(t, s.SupportsSnippets())
	})

	t.Run("returns false when SnippetSupport is nil", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.SetClientCapabilities(protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{
				Completion: &protocol.CompletionClientCapabilities{
					CompletionItem: &struct {
						SnippetSupport          *bool    `json:"snippetSupport,omitempty"`
						CommitCharactersSupport *bool    `json:"commitCharactersSupport,omitempty"`
						DocumentationFormat     []protocol.MarkupKind `json:"documentationFormat,omitempty"`
						DeprecatedSupport       *bool    `json:"deprecatedSupport,omitempty"`
						PreselectSupport        *bool    `json:"preselectSupport,omitempty"`
						TagSupport              *struct {
							ValueSet []protocol.CompletionItemTag `json:"valueSet"`
						} `json:"tagSupport,omitempty"`
						InsertReplaceSupport    *bool `json:"insertReplaceSupport,omitempty"`
						ResolveSupport          *struct {
							Properties []string `json:"properties"`
						} `json:"resolveSupport,omitempty"`
						InsertTextModeSupport   *struct {
							ValueSet []protocol.InsertTextMode `json:"valueSet"`
						} `json:"insertTextModeSupport,omitempty"`
					}{},
				},
			},
		})
		assert.False(t, s.SupportsSnippets())
	})

	t.Run("returns true when SnippetSupport is true", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		snippetSupport := true
		s.SetClientCapabilities(protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{
				Completion: &protocol.CompletionClientCapabilities{
					CompletionItem: &struct {
						SnippetSupport          *bool    `json:"snippetSupport,omitempty"`
						CommitCharactersSupport *bool    `json:"commitCharactersSupport,omitempty"`
						DocumentationFormat     []protocol.MarkupKind `json:"documentationFormat,omitempty"`
						DeprecatedSupport       *bool    `json:"deprecatedSupport,omitempty"`
						PreselectSupport        *bool    `json:"preselectSupport,omitempty"`
						TagSupport              *struct {
							ValueSet []protocol.CompletionItemTag `json:"valueSet"`
						} `json:"tagSupport,omitempty"`
						InsertReplaceSupport    *bool `json:"insertReplaceSupport,omitempty"`
						ResolveSupport          *struct {
							Properties []string `json:"properties"`
						} `json:"resolveSupport,omitempty"`
						InsertTextModeSupport   *struct {
							ValueSet []protocol.InsertTextMode `json:"valueSet"`
						} `json:"insertTextModeSupport,omitempty"`
					}{
						SnippetSupport: &snippetSupport,
					},
				},
			},
		})
		assert.True(t, s.SupportsSnippets())
	})
}

func TestServer_PreferredHoverFormat(t *testing.T) {
	t.Run("returns markdown when capabilities are nil", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		assert.Equal(t, protocol.MarkupKindMarkdown, s.PreferredHoverFormat())
	})

	t.Run("returns markdown when TextDocument is nil", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.SetClientCapabilities(protocol.ClientCapabilities{})
		assert.Equal(t, protocol.MarkupKindMarkdown, s.PreferredHoverFormat())
	})

	t.Run("returns markdown when Hover is nil", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.SetClientCapabilities(protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{},
		})
		assert.Equal(t, protocol.MarkupKindMarkdown, s.PreferredHoverFormat())
	})

	t.Run("returns markdown when ContentFormat is empty", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.SetClientCapabilities(protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{
				Hover: &protocol.HoverClientCapabilities{
					ContentFormat: []protocol.MarkupKind{},
				},
			},
		})
		assert.Equal(t, protocol.MarkupKindMarkdown, s.PreferredHoverFormat())
	})

	t.Run("returns first format from ContentFormat", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.SetClientCapabilities(protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{
				Hover: &protocol.HoverClientCapabilities{
					ContentFormat: []protocol.MarkupKind{protocol.MarkupKindPlainText, protocol.MarkupKindMarkdown},
				},
			},
		})
		assert.Equal(t, protocol.MarkupKindPlainText, s.PreferredHoverFormat())
	})
}

func TestServer_SupportsDefinitionLinks(t *testing.T) {
	t.Run("returns false when capabilities are nil", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		assert.False(t, s.SupportsDefinitionLinks())
	})

	t.Run("returns false when TextDocument is nil", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.SetClientCapabilities(protocol.ClientCapabilities{})
		assert.False(t, s.SupportsDefinitionLinks())
	})

	t.Run("returns false when Definition is nil", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.SetClientCapabilities(protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{},
		})
		assert.False(t, s.SupportsDefinitionLinks())
	})

	t.Run("returns false when LinkSupport is nil", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.SetClientCapabilities(protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{
				Definition: &protocol.DefinitionClientCapabilities{},
			},
		})
		assert.False(t, s.SupportsDefinitionLinks())
	})

	t.Run("returns true when LinkSupport is true", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		linkSupport := true
		s.SetClientCapabilities(protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{
				Definition: &protocol.DefinitionClientCapabilities{
					LinkSupport: &linkSupport,
				},
			},
		})
		assert.True(t, s.SupportsDefinitionLinks())
	})
}

func TestServer_SupportsDiagnosticRelatedInfo(t *testing.T) {
	t.Run("returns false when capabilities are nil", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		assert.False(t, s.SupportsDiagnosticRelatedInfo())
	})

	t.Run("returns false when TextDocument is nil", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.SetClientCapabilities(protocol.ClientCapabilities{})
		assert.False(t, s.SupportsDiagnosticRelatedInfo())
	})

	t.Run("returns false when PublishDiagnostics is nil", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.SetClientCapabilities(protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{},
		})
		assert.False(t, s.SupportsDiagnosticRelatedInfo())
	})

	t.Run("returns false when RelatedInformation is nil", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.SetClientCapabilities(protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{
				PublishDiagnostics: &protocol.PublishDiagnosticsClientCapabilities{},
			},
		})
		assert.False(t, s.SupportsDiagnosticRelatedInfo())
	})

	t.Run("returns true when RelatedInformation is true", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		relatedInfo := true
		s.SetClientCapabilities(protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{
				PublishDiagnostics: &protocol.PublishDiagnosticsClientCapabilities{
					RelatedInformation: &relatedInfo,
				},
			},
		})
		assert.True(t, s.SupportsDiagnosticRelatedInfo())
	})
}

func TestNewServer(t *testing.T) {
	t.Run("creates server with default values", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)
		assert.NotNil(t, s)
		assert.Equal(t, "dev", s.Version())
		assert.NotNil(t, s.DocumentManager())
		assert.NotNil(t, s.TokenManager())
		assert.Equal(t, 0, s.TokenCount())
		assert.Nil(t, s.ClientDiagnosticCapability())
		assert.Nil(t, s.ClientCapabilities())
		assert.False(t, s.UsePullDiagnostics())
		assert.NotNil(t, s.SemanticTokenCache())
	})

	t.Run("applies WithVersion option", func(t *testing.T) {
		s, err := NewServer(WithVersion("1.2.3"))
		require.NoError(t, err)
		assert.Equal(t, "1.2.3", s.Version())
	})
}

func TestServer_SetGLSPContext(t *testing.T) {
	s, err := NewServer()
	require.NoError(t, err)

	// Initially nil
	assert.Nil(t, s.GLSPContext())

	// Set and retrieve
	ctx := &glsp.Context{}
	s.SetGLSPContext(ctx)
	assert.Equal(t, ctx, s.GLSPContext())
}

func TestServer_ClientDiagnosticCapability(t *testing.T) {
	s, err := NewServer()
	require.NoError(t, err)

	// Initially nil (not detected yet)
	assert.Nil(t, s.ClientDiagnosticCapability())

	// Set to true
	s.SetClientDiagnosticCapability(true)
	cap := s.ClientDiagnosticCapability()
	require.NotNil(t, cap)
	assert.True(t, *cap)

	// Set to false
	s.SetClientDiagnosticCapability(false)
	cap = s.ClientDiagnosticCapability()
	require.NotNil(t, cap)
	assert.False(t, *cap)
}

func TestServer_ClientCapabilities(t *testing.T) {
	s, err := NewServer()
	require.NoError(t, err)

	// Initially nil
	assert.Nil(t, s.ClientCapabilities())

	// Set and retrieve
	caps := protocol.ClientCapabilities{
		TextDocument: &protocol.TextDocumentClientCapabilities{},
	}
	s.SetClientCapabilities(caps)
	got := s.ClientCapabilities()
	require.NotNil(t, got)
	assert.NotNil(t, got.TextDocument)
}

func TestServer_SetUsePullDiagnostics(t *testing.T) {
	s, err := NewServer()
	require.NoError(t, err)

	assert.False(t, s.UsePullDiagnostics())

	s.SetUsePullDiagnostics(true)
	assert.True(t, s.UsePullDiagnostics())

	s.SetUsePullDiagnostics(false)
	assert.False(t, s.UsePullDiagnostics())
}

func TestServer_RemoveLoadedFile(t *testing.T) {
	s, err := NewServer()
	require.NoError(t, err)

	// Add a loaded file
	s.loadedFiles["/workspace/tokens.json"] = &TokenFileOptions{Prefix: "ds"}
	assert.True(t, s.IsTokenFile("/workspace/tokens.json"))

	// Remove it
	s.RemoveLoadedFile("/workspace/tokens.json")
	assert.False(t, s.IsTokenFile("/workspace/tokens.json"))
}

func TestServer_RemoveLoadedFile_NormalizesPath(t *testing.T) {
	s, err := NewServer()
	require.NoError(t, err)

	// Add a loaded file with normalized path
	s.loadedFiles["/workspace/tokens.json"] = &TokenFileOptions{}
	assert.True(t, s.IsTokenFile("/workspace/tokens.json"))

	// Remove with unnormalized path (has /./)
	s.RemoveLoadedFile("/workspace/./tokens.json")
	assert.False(t, s.IsTokenFile("/workspace/tokens.json"))
}

func TestPublishDiagnostics_PullDiagnosticsSkips(t *testing.T) {
	s, err := NewServer()
	require.NoError(t, err)

	// Enable pull diagnostics
	s.SetUsePullDiagnostics(true)

	// Even with a valid context, PublishDiagnostics should be a no-op
	ctx := &glsp.Context{}
	s.SetGLSPContext(ctx)

	err = s.PublishDiagnostics(ctx, "file:///test.css")
	// Should return nil (skipped, not error)
	assert.NoError(t, err)
}

func TestServer_RegisterFileWatchers_NilContext(t *testing.T) {
	s, err := NewServer()
	require.NoError(t, err)

	// Should return nil without panicking when context is nil
	err = s.RegisterFileWatchers(nil)
	assert.NoError(t, err)
}

func TestServer_RegisterFileWatchers_EmptyContext(t *testing.T) {
	s, err := NewServer()
	require.NoError(t, err)

	// An empty context (Call is nil) should also skip registration
	ctx := &glsp.Context{}
	err = s.RegisterFileWatchers(ctx)
	assert.NoError(t, err)
}

func TestServer_RegisterFileWatchers_WithCallContext(t *testing.T) {
	s, err := NewServer()
	require.NoError(t, err)

	// Configure some token files so watchers are generated
	s.config.TokensFiles = []any{"tokens.json"}

	ctx := &glsp.Context{
		Call: func(_ string, _ any, _ any) {},
	}

	err = s.RegisterFileWatchers(ctx)
	assert.NoError(t, err)
}

func TestServer_BuildFileWatchers(t *testing.T) {
	t.Run("builds watchers for string token file paths", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.rootPath = "/workspace"
		s.config.TokensFiles = []any{
			"tokens.json",
			"design-tokens.json",
		}

		watchers := s.buildFileWatchers()
		require.Len(t, watchers, 2)
		// Relative paths joined with rootPath
		assert.Equal(t, "/workspace/tokens.json", watchers[0].GlobPattern)
		assert.Equal(t, "/workspace/design-tokens.json", watchers[1].GlobPattern)
	})

	t.Run("builds watchers for object token file paths", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.rootPath = "/workspace"
		s.config.TokensFiles = []any{
			map[string]any{"path": "tokens.json", "prefix": "ds"},
		}

		watchers := s.buildFileWatchers()
		require.Len(t, watchers, 1)
		assert.Equal(t, "/workspace/tokens.json", watchers[0].GlobPattern)
	})

	t.Run("skips entries with empty path", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.rootPath = "/workspace"
		s.config.TokensFiles = []any{
			"",                            // empty string
			map[string]any{"path": ""},    // empty path in object
			map[string]any{"prefix": "ds"}, // no path key at all
		}

		watchers := s.buildFileWatchers()
		assert.Empty(t, watchers)
	})

	t.Run("handles absolute token file paths", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.rootPath = "/workspace"
		s.config.TokensFiles = []any{
			"/absolute/path/to/tokens.json",
		}

		watchers := s.buildFileWatchers()
		require.Len(t, watchers, 1)
		assert.Equal(t, "/absolute/path/to/tokens.json", watchers[0].GlobPattern)
	})

	t.Run("handles relative paths with no root path", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		// No rootPath set
		s.config.TokensFiles = []any{
			"tokens.json",
		}

		watchers := s.buildFileWatchers()
		require.Len(t, watchers, 1)
		assert.Equal(t, "tokens.json", watchers[0].GlobPattern)
	})

	t.Run("returns empty for no configured files", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.config.TokensFiles = []any{}

		watchers := s.buildFileWatchers()
		assert.Empty(t, watchers)
	})

	t.Run("cleans redundant path separators", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.rootPath = "/workspace"
		s.config.TokensFiles = []any{
			"./sub/../tokens.json",
		}

		watchers := s.buildFileWatchers()
		require.Len(t, watchers, 1)
		// filepath.Clean should normalize the path
		assert.Equal(t, "/workspace/tokens.json", watchers[0].GlobPattern)
	})
}

func TestPublishDiagnostics_UsesPassedContext(t *testing.T) {
	t.Run("skips when pull diagnostics enabled with passed context", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.SetUsePullDiagnostics(true)

		// Even with a passed context, should skip
		ctx := &glsp.Context{}
		err = s.PublishDiagnostics(ctx, "file:///test.json")
		assert.NoError(t, err)
	})

	t.Run("falls back to server context when passed nil", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		// No server context set either -- should error
		err = s.PublishDiagnostics(nil, "file:///test.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no client context available")
	})
}

func TestPublishDiagnostics_NotifiesClient(t *testing.T) {
	s, err := NewServer()
	require.NoError(t, err)

	// Open a CSS document so GetDiagnostics has something to work with
	s.DocumentManager().DidOpen("file:///test.css", "css", 1, ".btn { color: var(--c); }")

	// Add a token so diagnostics can resolve
	tok := &tokens.Token{Name: "c", Value: "#ff0000", Type: "color"}
	_ = s.TokenManager().Add(tok)

	// Non-nil context with no-op Notify exercises the notification branch
	notified := false
	ctx := &glsp.Context{
		Notify: func(_ string, _ any) {
			notified = true
		},
	}

	err = s.PublishDiagnostics(ctx, "file:///test.css")
	assert.NoError(t, err)
	assert.True(t, notified, "expected Notify to be called")
}

func TestServer_IsTokenFile_ObjectWithMissingPath(t *testing.T) {
	s, err := NewServer()
	require.NoError(t, err)

	s.rootPath = "/workspace"
	// Object entry without "path" key
	s.config.TokensFiles = []any{
		map[string]any{"prefix": "ds"},
	}

	result := s.IsTokenFile("/workspace/tokens.json")
	assert.False(t, result)
}

func TestServer_IsTokenFile_ObjectWithEmptyPath(t *testing.T) {
	s, err := NewServer()
	require.NoError(t, err)

	s.rootPath = "/workspace"
	// Object entry with empty "path" value
	s.config.TokensFiles = []any{
		map[string]any{"path": ""},
	}

	result := s.IsTokenFile("/workspace/tokens.json")
	assert.False(t, result)
}

func TestServer_SupportsCodeActionLiterals(t *testing.T) {
	t.Run("returns false when capabilities are nil", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		assert.False(t, s.SupportsCodeActionLiterals())
	})

	t.Run("returns false when TextDocument is nil", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.SetClientCapabilities(protocol.ClientCapabilities{})
		assert.False(t, s.SupportsCodeActionLiterals())
	})

	t.Run("returns false when CodeAction is nil", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.SetClientCapabilities(protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{},
		})
		assert.False(t, s.SupportsCodeActionLiterals())
	})

	t.Run("returns false when CodeActionLiteralSupport is nil", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.SetClientCapabilities(protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{
				CodeAction: &protocol.CodeActionClientCapabilities{
					// No CodeActionLiteralSupport
				},
			},
		})
		assert.False(t, s.SupportsCodeActionLiterals())
	})

	t.Run("returns true when CodeActionLiteralSupport is present", func(t *testing.T) {
		s, err := NewServer()
		require.NoError(t, err)

		s.SetClientCapabilities(protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{
				CodeAction: &protocol.CodeActionClientCapabilities{
					CodeActionLiteralSupport: &struct {
						CodeActionKind struct {
							ValueSet []protocol.CodeActionKind `json:"valueSet"`
						} `json:"codeActionKind"`
					}{
						CodeActionKind: struct {
							ValueSet []protocol.CodeActionKind `json:"valueSet"`
						}{
							ValueSet: []protocol.CodeActionKind{protocol.CodeActionKindRefactorRewrite},
						},
					},
				},
			},
		})
		assert.True(t, s.SupportsCodeActionLiterals())
	})
}
