package inlayhint

import (
	"testing"

	"bennypowers.dev/asimonim/lsp/internal/tokens"
	"bennypowers.dev/asimonim/lsp/testutil"
	"bennypowers.dev/asimonim/lsp/types"
	"github.com/bennypowers/glsp"
	protocol "github.com/bennypowers/glsp/protocol_3_17"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInlayHint_VarCallShowsResolvedValue(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "spacing.lg",
		Value: "24px",
		Type:  "dimension",
	}))

	uri := "file:///test.css"
	// padding: var(--spacing-lg);
	cssContent := `.box { padding: var(--spacing-lg); }`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	result, err := InlayHint(req, &protocol.InlayHintParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 100},
		},
	})

	require.NoError(t, err)
	require.Len(t, result, 1)

	// Hint positioned before closing paren of var()
	// `.box { padding: var(--spacing-lg); }`
	//                                   ^ position 32 (before closing paren)
	assert.Equal(t, uint32(0), result[0].Position.Line)
	assert.Equal(t, uint32(32), result[0].Position.Character)
	// Label is ", 24px" to look like a fallback value
	assert.Equal(t, ", 24px", result[0].Label)
	padLeft := true
	assert.Equal(t, &padLeft, result[0].PaddingLeft)
}

func TestInlayHint_UnknownTokenSkipped(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///test.css"
	cssContent := `.box { padding: var(--unknown-token); }`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	result, err := InlayHint(req, &protocol.InlayHintParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 100},
		},
	})

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestInlayHint_EmptyDocument(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///test.css"
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, ""))

	result, err := InlayHint(req, &protocol.InlayHintParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 0},
		},
	})

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestInlayHint_DisabledBySetting(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	disabled := false
	cfg := ctx.GetConfig()
	cfg.InlayHints = &disabled
	ctx.SetConfig(cfg)

	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "spacing.lg",
		Value: "24px",
		Type:  "dimension",
	}))

	uri := "file:///test.css"
	cssContent := `.box { padding: var(--spacing-lg); }`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	result, err := InlayHint(req, &protocol.InlayHintParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 100},
		},
	})

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestInlayHint_VarCallWithExistingFallback(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "spacing.lg",
		Value: "24px",
		Type:  "dimension",
	}))

	uri := "file:///test.css"
	// var() already has a fallback - no hint needed
	cssContent := `.box { padding: var(--spacing-lg, 16px); }`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	result, err := InlayHint(req, &protocol.InlayHintParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 100},
		},
	})

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestInlayHint_MultipleVarCalls(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "spacing.sm",
		Value: "8px",
		Type:  "dimension",
	}))
	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "spacing.lg",
		Value: "24px",
		Type:  "dimension",
	}))

	uri := "file:///test.css"
	cssContent := `.box {
  padding: var(--spacing-sm);
  margin: var(--spacing-lg);
}`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	result, err := InlayHint(req, &protocol.InlayHintParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 3, Character: 1},
		},
	})

	require.NoError(t, err)
	require.Len(t, result, 2)
}

func TestInlayHint_ColorToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
		Type:  "color",
	}))

	uri := "file:///test.css"
	cssContent := `.btn { color: var(--color-primary); }`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	result, err := InlayHint(req, &protocol.InlayHintParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 100},
		},
	})

	require.NoError(t, err)
	require.Len(t, result, 1)
	// color.primary DisplayValue = "#ff0000"
	assert.Equal(t, ", #ff0000", result[0].Label)
}

func TestInlayHint_HTMLDocument(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "spacing.lg",
		Value: "24px",
		Type:  "dimension",
	}))

	uri := "file:///test.html"
	htmlContent := `<style>.box { padding: var(--spacing-lg); }</style>`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "html", 1, htmlContent))

	result, err := InlayHint(req, &protocol.InlayHintParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 100},
		},
	})

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, ", 24px", result[0].Label)
}

func TestInlayHint_MissingDocument(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	result, err := InlayHint(req, &protocol.InlayHintParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.css"},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 0},
		},
	})

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestInlayHint_VarCallOutsideRange(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "spacing.sm",
		Value: "8px",
		Type:  "dimension",
	}))
	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "spacing.lg",
		Value: "24px",
		Type:  "dimension",
	}))

	uri := "file:///test.css"
	cssContent := `.a { padding: var(--spacing-sm); }
.b { margin: var(--spacing-lg); }`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	// Request only line 1 -- should exclude line 0 var() call
	result, err := InlayHint(req, &protocol.InlayHintParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 1, Character: 0},
			End:   protocol.Position{Line: 1, Character: 100},
		},
	})

	require.NoError(t, err)
	require.Len(t, result, 1)
	// spacing.lg on line 1, not spacing.sm on line 0
	assert.Equal(t, ", 24px", result[0].Label)
	assert.Equal(t, uint32(1), result[0].Position.Line)
}

func TestInlayHint_RangeExcludesBothLines(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "spacing.sm",
		Value: "8px",
		Type:  "dimension",
	}))

	uri := "file:///test.css"
	cssContent := `.a { padding: var(--spacing-sm); }`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	// Request range on a different line entirely
	result, err := InlayHint(req, &protocol.InlayHintParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 5, Character: 0},
			End:   protocol.Position{Line: 10, Character: 0},
		},
	})

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestInlayHint_UnsupportedLanguage(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///test.json"
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "json", 1, `{"foo": "bar"}`))

	result, err := InlayHint(req, &protocol.InlayHintParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 100},
		},
	})

	require.NoError(t, err)
	assert.Empty(t, result)
}
