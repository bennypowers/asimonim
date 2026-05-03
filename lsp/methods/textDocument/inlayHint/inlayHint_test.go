package inlayhint

import (
	"testing"

	"bennypowers.dev/asimonim/lsp/internal/tokens"
	"bennypowers.dev/asimonim/lsp/testutil"
	"bennypowers.dev/asimonim/lsp/types"
	"bennypowers.dev/asimonim/schema"
	fixtureutil "bennypowers.dev/asimonim/testutil"
	"github.com/bennypowers/glsp"
	protocol "github.com/bennypowers/glsp/protocol_3_17"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInlayHint_VarCallShowsResolvedValue(t *testing.T) {
	allTokens := fixtureutil.ParseFixtureTokens(t, "fixtures/v2025_10/all-color-spaces", schema.V2025_10)
	// spacing.small: {value: 4, unit: "px"} -> 4px
	spacingSmall := fixtureutil.TokenByPath(t, allTokens, "spacing.small")

	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)
	require.NoError(t, ctx.TokenManager().Add(spacingSmall))

	uri := "file:///test.css"
	// `.box { padding: var(--spacing-small); }`
	cssContent := `.box { padding: var(--spacing-small); }`
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
	// `.box { padding: var(--spacing-small); }`
	//                                      ^ position 35 (before closing paren)
	assert.Equal(t, uint32(0), result[0].Position.Line)
	assert.Equal(t, uint32(35), result[0].Position.Character)
	// spacing.small DisplayValue = "4px"
	assert.Equal(t, ", 4px", result[0].Label)
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
	allTokens := fixtureutil.ParseFixtureTokens(t, "fixtures/v2025_10/all-color-spaces", schema.V2025_10)
	spacingSmall := fixtureutil.TokenByPath(t, allTokens, "spacing.small")

	ctx := testutil.NewMockServerContext()
	disabled := false
	cfg := ctx.GetConfig()
	cfg.InlayHints = &disabled
	ctx.SetConfig(cfg)

	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)
	require.NoError(t, ctx.TokenManager().Add(spacingSmall))

	uri := "file:///test.css"
	cssContent := `.box { padding: var(--spacing-small); }`
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
	allTokens := fixtureutil.ParseFixtureTokens(t, "fixtures/v2025_10/all-color-spaces", schema.V2025_10)
	spacingSmall := fixtureutil.TokenByPath(t, allTokens, "spacing.small")

	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)
	require.NoError(t, ctx.TokenManager().Add(spacingSmall))

	uri := "file:///test.css"
	cssContent := `.box { padding: var(--spacing-small, 16px); }`
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
	allTokens := fixtureutil.ParseFixtureTokens(t, "fixtures/v2025_10/all-color-spaces", schema.V2025_10)
	// spacing.small: {value: 4, unit: "px"} -> 4px
	spacingSmall := fixtureutil.TokenByPath(t, allTokens, "spacing.small")
	// spacing.medium: {value: 1.5, unit: "rem"} -> 1.5rem
	spacingMedium := fixtureutil.TokenByPath(t, allTokens, "spacing.medium")

	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)
	require.NoError(t, ctx.TokenManager().Add(spacingSmall))
	require.NoError(t, ctx.TokenManager().Add(spacingMedium))

	uri := "file:///test.css"
	cssContent := `.box {
  padding: var(--spacing-small);
  margin: var(--spacing-medium);
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

	// spacing.small on line 1: ", 4px"
	assert.Equal(t, uint32(1), result[0].Position.Line)
	assert.Equal(t, ", 4px", result[0].Label)
	// spacing.medium on line 2: ", 1.5rem"
	assert.Equal(t, uint32(2), result[1].Position.Line)
	assert.Equal(t, ", 1.5rem", result[1].Label)
}

func TestInlayHint_ColorToken(t *testing.T) {
	allTokens := fixtureutil.ParseFixtureTokens(t, "fixtures/v2025_10/all-color-spaces", schema.V2025_10)
	// color.srgb-hex: structured color -> DisplayValue = "#FF6B36"
	colorHex := fixtureutil.TokenByPath(t, allTokens, "color.srgb-hex")

	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)
	require.NoError(t, ctx.TokenManager().Add(colorHex))

	uri := "file:///test.css"
	cssContent := `.btn { color: var(--color-srgb-hex); }`
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
	// color.srgb-hex DisplayValue = "#FF6B36"
	assert.Equal(t, ", #FF6B36", result[0].Label)
}

func TestInlayHint_HTMLDocument(t *testing.T) {
	allTokens := fixtureutil.ParseFixtureTokens(t, "fixtures/v2025_10/all-color-spaces", schema.V2025_10)
	// spacing.small: {value: 4, unit: "px"} -> 4px
	spacingSmall := fixtureutil.TokenByPath(t, allTokens, "spacing.small")

	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)
	require.NoError(t, ctx.TokenManager().Add(spacingSmall))

	uri := "file:///test.html"
	htmlContent := `<style>.box { padding: var(--spacing-small); }</style>`
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
	// spacing.small DisplayValue = "4px"
	assert.Equal(t, ", 4px", result[0].Label)
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
