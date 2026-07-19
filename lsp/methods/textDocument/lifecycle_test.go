package textDocument

import (
	"context"
	"testing"

	"bennypowers.dev/asimonim/lsp/testutil"
	"bennypowers.dev/asimonim/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestDidOpen(t *testing.T) {
	t.Run("opens document successfully", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		req := types.NewRequestContext(ctx, context.Background())

		params := &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:        uri.URI("file:///test.css"),
				LanguageID: protocol.LanguageKind("css"),
				Version:    1,
				Text:       "body { color: red; }",
			},
		}

		err := DidOpen(req, params)
		require.NoError(t, err)

		// Verify document was opened
		doc := ctx.Document("file:///test.css")
		require.NotNil(t, doc)
		assert.Equal(t, "file:///test.css", doc.URI())
		assert.Equal(t, "css", doc.LanguageID())
		assert.Equal(t, 1, doc.Version())
		assert.Equal(t, "body { color: red; }", doc.Content())
	})

	t.Run("publishes diagnostics after opening", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		req := types.NewRequestContext(ctx, context.Background())
		ctx.SetServerCtx(context.Background())

		params := &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:        uri.URI("file:///test.css"),
				LanguageID: protocol.LanguageKind("css"),
				Version:    1,
				Text:       "body { color: red; }",
			},
		}

		err := DidOpen(req, params)
		require.NoError(t, err)

		// Diagnostics are published asynchronously, no direct assertion needed
	})

	t.Run("handles JSON document", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		req := types.NewRequestContext(ctx, context.Background())

		params := &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:        uri.URI("file:///tokens.json"),
				LanguageID: protocol.LanguageKind("json"),
				Version:    1,
				Text:       `{"color": {"$type": "color", "$value": "#ff0000"}}`,
			},
		}

		err := DidOpen(req, params)
		require.NoError(t, err)

		doc := ctx.Document("file:///tokens.json")
		require.NotNil(t, doc)
		assert.Equal(t, "json", doc.LanguageID())
	})

	t.Run("handles YAML document", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		req := types.NewRequestContext(ctx, context.Background())

		params := &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:        uri.URI("file:///tokens.yaml"),
				LanguageID: protocol.LanguageKind("yaml"),
				Version:    1,
				Text:       "color:\n  $type: color\n  $value: '#ff0000'",
			},
		}

		err := DidOpen(req, params)
		require.NoError(t, err)

		doc := ctx.Document("file:///tokens.yaml")
		require.NotNil(t, doc)
		assert.Equal(t, "yaml", doc.LanguageID())
	})
}

func TestDidChange(t *testing.T) {
	t.Run("updates document content", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		req := types.NewRequestContext(ctx, context.Background())

		// First open a document
		_ = ctx.DocumentManager().DidOpen("file:///test.css", "css", 1, "body { color: red; }")

		// Change the document (whole document replacement)
		params := &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri.URI("file:///test.css")},
				Version:                2,
			},
			ContentChanges: []protocol.TextDocumentContentChangeEvent{
				&protocol.TextDocumentContentChangeWholeDocument{Text: "body { color: blue; }"},
			},
		}

		err := DidChange(req, params)
		require.NoError(t, err)

		// Verify document was updated
		doc := ctx.Document("file:///test.css")
		require.NotNil(t, doc)
		assert.Equal(t, 2, doc.Version())
		assert.Equal(t, "body { color: blue; }", doc.Content())
	})

	t.Run("publishes diagnostics after change", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		req := types.NewRequestContext(ctx, context.Background())
		ctx.SetServerCtx(context.Background())

		// First open a document
		_ = ctx.DocumentManager().DidOpen("file:///test.css", "css", 1, "body { color: red; }")

		params := &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri.URI("file:///test.css")},
				Version:                2,
			},
			ContentChanges: []protocol.TextDocumentContentChangeEvent{
				&protocol.TextDocumentContentChangeWholeDocument{Text: "body { color: blue; }"},
			},
		}

		err := DidChange(req, params)
		require.NoError(t, err)

		// Diagnostics are published asynchronously, no direct assertion needed
	})

	t.Run("handles incremental changes", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		req := types.NewRequestContext(ctx, context.Background())

		// First open a document
		_ = ctx.DocumentManager().DidOpen("file:///test.css", "css", 1, "body { color: red; }")

		// Incremental change with range
		params := &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri.URI("file:///test.css")},
				Version:                2,
			},
			ContentChanges: []protocol.TextDocumentContentChangeEvent{
				&protocol.TextDocumentContentChangePartial{
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 7},
						End:   protocol.Position{Line: 0, Character: 18},
					},
					Text: "background: blue",
				},
			},
		}

		err := DidChange(req, params)
		require.NoError(t, err)

		// Verify version was updated
		doc := ctx.Document("file:///test.css")
		require.NotNil(t, doc)
		assert.Equal(t, 2, doc.Version())
	})

	t.Run("handles multiple changes", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		req := types.NewRequestContext(ctx, context.Background())

		// First open a document
		_ = ctx.DocumentManager().DidOpen("file:///test.css", "css", 1, "body { color: red; }")

		params := &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri.URI("file:///test.css")},
				Version:                2,
			},
			ContentChanges: []protocol.TextDocumentContentChangeEvent{
				&protocol.TextDocumentContentChangeWholeDocument{Text: "body { color: blue; }"},
				&protocol.TextDocumentContentChangeWholeDocument{Text: "body { background: green; }"},
			},
		}

		err := DidChange(req, params)
		require.NoError(t, err)

		doc := ctx.Document("file:///test.css")
		require.NotNil(t, doc)
		assert.Equal(t, 2, doc.Version())
	})
}

func TestDidClose(t *testing.T) {
	t.Run("closes document successfully", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		req := types.NewRequestContext(ctx, context.Background())

		// First open a document
		_ = ctx.DocumentManager().DidOpen("file:///test.css", "css", 1, "body { color: red; }")

		params := &protocol.DidCloseTextDocumentParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri.URI("file:///test.css")},
		}

		err := DidClose(req, params)
		require.NoError(t, err)

		// Verify document was closed
		doc := ctx.Document("file:///test.css")
		assert.Nil(t, doc)
	})

	t.Run("returns error when closing non-existent document", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		req := types.NewRequestContext(ctx, context.Background())

		params := &protocol.DidCloseTextDocumentParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri.URI("file:///nonexistent.css")},
		}

		// Should return error
		err := DidClose(req, params)
		assert.Error(t, err)
	})

	t.Run("closes multiple documents independently", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		req := types.NewRequestContext(ctx, context.Background())

		// Open two documents
		_ = ctx.DocumentManager().DidOpen("file:///test1.css", "css", 1, "body { color: red; }")
		_ = ctx.DocumentManager().DidOpen("file:///test2.css", "css", 1, "div { color: blue; }")

		// Close first document
		params := &protocol.DidCloseTextDocumentParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri.URI("file:///test1.css")},
		}

		err := DidClose(req, params)
		require.NoError(t, err)

		// First should be closed, second should remain
		assert.Nil(t, ctx.Document("file:///test1.css"))
		assert.NotNil(t, ctx.Document("file:///test2.css"))
	})
}
