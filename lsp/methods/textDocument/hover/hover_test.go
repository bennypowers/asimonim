package hover

import (
	"flag"
	"os"
	"testing"

	asimonim "bennypowers.dev/asimonim/parser"
	"bennypowers.dev/asimonim/lsp/internal/parser/css"
	tokens "bennypowers.dev/asimonim/lsp/internal/tokens"
	"bennypowers.dev/asimonim/lsp/testutil"
	"bennypowers.dev/asimonim/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var update = flag.Bool("update", false, "update golden files")

// TestIsPositionInRange tests the isPositionInRange function with half-open range semantics [start, end)
func TestIsPositionInRange(t *testing.T) {
	tests := []struct {
		name     string
		pos      protocol.Position
		r        css.Range
		expected bool
	}{
		{
			name: "position at start boundary - included",
			pos:  protocol.Position{Line: 0, Character: 5},
			r: css.Range{
				Start: css.Position{Line: 0, Character: 5},
				End:   css.Position{Line: 0, Character: 10},
			},
			expected: true,
		},
		{
			name: "position at end boundary - excluded",
			pos:  protocol.Position{Line: 0, Character: 10},
			r: css.Range{
				Start: css.Position{Line: 0, Character: 5},
				End:   css.Position{Line: 0, Character: 10},
			},
			expected: false,
		},
		{
			name: "position inside range",
			pos:  protocol.Position{Line: 0, Character: 7},
			r: css.Range{
				Start: css.Position{Line: 0, Character: 5},
				End:   css.Position{Line: 0, Character: 10},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPositionInRange(tt.pos, tt.r)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHover_CSSVariableReference(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a token
	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:        "color.primary",
		Value:       "#ff0000",
		Type:        "color",
		Description: "Primary brand color",
		FilePath:    "tokens.json",
	}))

	// Open a CSS document with var() call
	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	// Hover over --color-primary in var() call
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 24},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)

	// Assert hover content
	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok, "Contents should be MarkupContent")
	assert.Contains(t, content.Value, "--color-primary")
	assert.Contains(t, content.Value, "#ff0000")
	assert.Contains(t, content.Value, "color")
	assert.Contains(t, content.Value, "Primary brand color")
	assert.Contains(t, content.Value, "tokens.json")

	// Assert Range is present for var() calls
	require.NotNil(t, hover.Range, "Range should be present for var() call")
}

func TestHover_DeprecatedToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add deprecated token
	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:               "color.old-primary",
		Value:              "#cc0000",
		Type:               "color",
		Deprecated:         true,
		DeprecationMessage: "Use color.primary instead",
	}))

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-old-primary); }`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 28},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)

	// Assert deprecation warning
	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content.Value, "DEPRECATED")
	assert.Contains(t, content.Value, "Use color.primary instead")
}

func TestHover_UnknownToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///test.css"
	cssContent := `.button { color: var(--unknown-token); }`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 28},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover, "Should show 'unknown token' message for var() calls with unknown tokens")

	// Assert unknown token message
	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content.Value, "Unknown token")
	assert.Contains(t, content.Value, "--unknown-token")

	// Assert Range is present for unknown token (consistency with known tokens)
	require.NotNil(t, hover.Range, "Range should be present for unknown token var() call")
}

func TestHover_VarCallWithFallback(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "spacing.large",
		Value: "2rem",
		Type:  "dimension",
	}))

	uri := "file:///test.css"
	cssContent := `.card { padding: var(--spacing-large, 1rem); }`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	// Hover over the token name in var() call with fallback
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 28},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)

	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content.Value, "--spacing-large")
	assert.Contains(t, content.Value, "2rem")
}

func TestHover_NestedVarCalls(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
		Type:  "color",
	}))

	uri := "file:///test.css"
	// Nested var() - hover should work on the inner one
	cssContent := `.element { background: linear-gradient(var(--color-primary), white); }`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 47},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)

	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content.Value, "--color-primary")
}

func TestHover_VarCallOutsideCursorRange(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
	}))

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	// Hover on "color:" property, not in var() range
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 12},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, hover, "Should not show hover outside var() range")
}

func TestHover_VariableDeclaration(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a token
	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:        "color.primary",
		Value:       "#ff0000",
		Type:        "color",
		Description: "Primary brand color",
	}))

	uri := "file:///test.css"
	cssContent := `:root { --color-primary: #ff0000; }`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	// Hover over variable declaration (on the property name)
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)

	// Assert hover content for declaration
	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content.Value, "--color-primary")
	assert.Contains(t, content.Value, "#ff0000")
	assert.Contains(t, content.Value, "Primary brand color")

	// Assert Range is present and covers only the property name
	require.NotNil(t, hover.Range, "Range should be present for known token declaration")
	assert.Equal(t, uint32(0), hover.Range.Start.Line)
	assert.Equal(t, uint32(8), hover.Range.Start.Character) // Start of --color-primary (first dash)
	assert.Equal(t, uint32(0), hover.Range.End.Line)
	assert.Equal(t, uint32(23), hover.Range.End.Character) // End of --color-primary (just before colon)
}

func TestHover_VariableDeclaration_UnknownToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///test.css"
	// --local-var is not a known design token, just a local CSS custom property
	cssContent := `:root { --local-var: blue; }`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	// Hover over unknown variable declaration
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, hover, "Should not show hover for unknown token declaration (local CSS var)")
}

func TestHover_VariableDeclaration_OnValue(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a token
	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
		Type:  "color",
	}))

	uri := "file:///test.css"
	cssContent := `:root { --color-primary: #ff0000; }`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	// Hover on the value side (RHS) - should not trigger hover
	// Character 25 is on "#ff0000"
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 25},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, hover, "Should not show hover when cursor is on value side (RHS)")
}

func TestHover_VariableDeclaration_Boundaries(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
	}))

	uri := "file:///test.css"
	cssContent := `:root { --color-primary: #ff0000; }`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	tests := []struct {
		name      string
		character uint32
		expectHit bool
	}{
		{"before property name (space)", 7, false},
		{"at start of property name (first dash)", 8, true},
		{"middle of property name", 15, true},
		{"near end of property name", 22, true},
		{"at end boundary (colon) - excluded", 23, false},
		{"after property name (space after colon)", 24, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hover, err := Hover(req, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: tt.character},
				},
			})

			require.NoError(t, err)
			if tt.expectHit {
				assert.NotNil(t, hover, "Expected hover at character %d", tt.character)
			} else {
				assert.Nil(t, hover, "Expected no hover at character %d", tt.character)
			}
		})
	}
}

func TestHover_VariableDeclaration_WithPrefix(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a token with prefix
	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:        "color.primary",
		Value:       "#0000ff",
		Type:        "color",
		Description: "Blue color",
		Prefix:      "ds",
	}))

	uri := "file:///test.css"
	// Token with prefix: --ds-color-primary
	cssContent := `:root { --ds-color-primary: #0000ff; }`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 12},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)

	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content.Value, "--ds-color-primary")
	assert.Contains(t, content.Value, "#0000ff")
	assert.Contains(t, content.Value, "Blue color")
}

func TestHover_VariableDeclaration_MultipleInSameBlock(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
		Type:  "color",
	}))
	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.secondary",
		Value: "#00ff00",
		Type:  "color",
	}))

	uri := "file:///test.css"
	cssContent := `:root {
  --color-primary: #ff0000;
  --color-secondary: #00ff00;
}`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	// Test first declaration
	hover1, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 1, Character: 5},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, hover1)
	content1, ok := hover1.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content1.Value, "--color-primary")

	// Test second declaration
	hover2, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 2, Character: 5},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, hover2)
	content2, ok := hover2.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content2.Value, "--color-secondary")
}

func TestHover_InvalidPosition(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
	}))

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	// Hover outside var() call
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 5},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, hover)
}

func TestHover_NonCSSDocument(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///test.json"
	jsonContent := `{"color": {"$value": "#ff0000"}}`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "json", 1, jsonContent))

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, hover)
}

func TestHover_DocumentNotFound(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.css"},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, hover)
}

// TestHover_NestedVarInFallback tests hovering over nested var() calls in fallback position
// This is the RHDS pattern: var(--local, var(--design-token, fallback))
func TestHover_NestedVarInFallback(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add design tokens (not the local variables)
	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:        "color-text-primary",
		Value:       "#000000",
		Type:        "color",
		Description: "Primary text color",
	}))
	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:        "color-surface-lightest",
		Value:       "#ffffff",
		Type:        "color",
		Description: "Lightest surface color",
	}))

	uri := "file:///test.css"
	// RHDS pattern: local variable with design token fallback
	// The outer var(--_local, ...) has a nested var(--design-token, fallback)
	cssContent := `.card {
  color: var(--_local-color, var(--color-text-primary, #000000));
  background: var(--_card-background, var(--color-surface-lightest, #ffffff));
}`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	t.Run("hover over inner token in nested fallback", func(t *testing.T) {
		// Hover over --color-text-primary (the inner/nested var)
		// Line 1, character 40 is approximately over --color-text-primary
		hover, err := Hover(req, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 1, Character: 40},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, hover, "Should find hover for inner token")

		content, ok := hover.Contents.(protocol.MarkupContent)
		require.True(t, ok)

		// Should show info for the INNER token, not the outer --_local-color
		assert.Contains(t, content.Value, "--color-text-primary", "Should show inner token name")
		assert.Contains(t, content.Value, "#000000", "Should show inner token value")
		assert.Contains(t, content.Value, "Primary text color", "Should show inner token description")
		assert.NotContains(t, content.Value, "Unknown token", "Should not report as unknown")
		assert.NotContains(t, content.Value, "--_local-color", "Should not show outer local variable")
	})

	t.Run("hover over outer local variable", func(t *testing.T) {
		// Hover over --_local-color (the outer var, which is a local variable)
		// Line 1, character 18 is approximately over --_local-color
		hover, err := Hover(req, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 1, Character: 18},
			},
		})

		require.NoError(t, err)
		// The outer var is a local CSS variable, not a design token.
		// It may return an "unknown token" hover or nil — either is acceptable.
		// What matters is that it does NOT show the inner token's information.
		if hover != nil {
			content, ok := hover.Contents.(protocol.MarkupContent)
			if ok {
				assert.NotContains(t, content.Value, "--color-text-primary", "Should not show inner token")
				assert.NotContains(t, content.Value, "Primary text color", "Should not show inner token description")
			}
		}
	})

	t.Run("hover over second nested var in same document", func(t *testing.T) {
		// Hover over --color-surface-lightest (line 2)
		// Line 2, character 50 is approximately over --color-surface-lightest
		hover, err := Hover(req, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 2, Character: 50},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, hover, "Should find hover for second inner token")

		content, ok := hover.Contents.(protocol.MarkupContent)
		require.True(t, ok)

		assert.Contains(t, content.Value, "--color-surface-lightest", "Should show correct token name")
		assert.Contains(t, content.Value, "#ffffff", "Should show correct token value")
		assert.Contains(t, content.Value, "Lightest surface color", "Should show correct token description")
		assert.NotContains(t, content.Value, "--_card-background", "Should not show outer local variable")
	})
}

func TestHover_ContentFormat(t *testing.T) {
	t.Run("returns markdown when client prefers it", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		ctx.SetPreferredHoverFormat(protocol.MarkupKindMarkdown)
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
			Name:        "color.primary",
			Value:       "#ff0000",
			Type:        "color",
			Description: "Primary brand color",
		}))

		uri := "file:///test.css"
		cssContent := `.button { color: var(--color-primary); }`
		require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

		hover, err := Hover(req, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 0, Character: 24},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, hover)

		content, ok := hover.Contents.(protocol.MarkupContent)
		require.True(t, ok)
		assert.Equal(t, protocol.MarkupKindMarkdown, content.Kind)
		assert.Contains(t, content.Value, "**Value (CSS)**") // Markdown formatting
	})

	t.Run("returns plaintext when client only supports plaintext", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		ctx.SetPreferredHoverFormat(protocol.MarkupKindPlainText)
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
			Name:        "color.primary",
			Value:       "#ff0000",
			Type:        "color",
			Description: "Primary brand color",
		}))

		uri := "file:///test.css"
		cssContent := `.button { color: var(--color-primary); }`
		require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

		hover, err := Hover(req, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 0, Character: 24},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, hover)

		content, ok := hover.Contents.(protocol.MarkupContent)
		require.True(t, ok)
		assert.Equal(t, protocol.MarkupKindPlainText, content.Kind)
		assert.NotContains(t, content.Value, "**") // No markdown formatting
		assert.Contains(t, content.Value, "Value (CSS):") // Plaintext formatting
	})

	t.Run("defaults to markdown when no preference", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		// Don't set format preference - test default behavior
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
			Name:  "color.primary",
			Value: "#ff0000",
		}))

		uri := "file:///test.css"
		cssContent := `.button { color: var(--color-primary); }`
		require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

		hover, err := Hover(req, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 0, Character: 24},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, hover)

		content, ok := hover.Contents.(protocol.MarkupContent)
		require.True(t, ok)
		assert.Equal(t, protocol.MarkupKindMarkdown, content.Kind)
	})
}

// ============================================================================
// HTML/JS Hover Tests
// ============================================================================

func TestHover_HTMLStyleTag(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
		Type:  "color",
	}))

	uri := "file:///test.html"
	// <style>.button { color: var(--color-primary); }</style>
	//        0         1         2         3         4
	//        0123456789012345678901234567890123456789012345678
	content := `<style>.button { color: var(--color-primary); }</style>`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "html", 1, content))

	// Character 30 is inside var(--color-primary)
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 30},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)
	mc, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, mc.Value, "--color-primary")

	require.NotNil(t, hover.Range, "Range should be present for var() call in HTML")
	assert.Equal(t, uint32(0), hover.Range.Start.Line)
}

func TestHover_JSCSSTemplate(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "spacing.small",
		Value: "8px",
		Type:  "dimension",
	}))

	uri := "file:///test.js"
	content := "const s = css`\n  .card { padding: var(--spacing-small); }\n`;"
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "javascript", 1, content))

	// Character 30 is inside var(--spacing-small) on line 1
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 1, Character: 30},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)
	mc, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, mc.Value, "--spacing-small")

	require.NotNil(t, hover.Range, "Range should be present for var() call in JS template")
	assert.Equal(t, uint32(1), hover.Range.Start.Line)
}

func TestHover_TSXCSSTemplate(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "spacing.small",
		Value: "8px",
		Type:  "dimension",
	}))

	uri := "file:///test.tsx"
	content := "const s = css`\n  .card { padding: var(--spacing-small); }\n`;"
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "typescriptreact", 1, content))

	// Character 30 is inside var(--spacing-small) on line 1
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 1, Character: 30},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)
	mc, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, mc.Value, "--spacing-small")

	require.NotNil(t, hover.Range, "Range should be present for var() call in TSX template")
	assert.Equal(t, uint32(1), hover.Range.Start.Line)
}

// ============================================================================
// JSON/YAML Token Reference Hover Tests
// ============================================================================

func TestHover_CurlyBraceReference_JSON(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add tokens
	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:        "color-primary",
		Value:       "#ff0000",
		Type:        "color",
		Description: "Primary brand color",
		FilePath:    "tokens.json",
	}))

	// Open a JSON document with curly brace reference
	uri := "file:///tokens.json"
	jsonContent := `{
  "color": {
    "secondary": {
      "$value": "{color.primary}"
    }
  }
}`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "json", 1, jsonContent))

	// Hover over {color.primary} - position inside the reference
	// Line 3: `      "$value": "{color.primary}"`
	// Character 20 is inside "color.primary"
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 3, Character: 20},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)

	// Assert hover content
	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok, "Contents should be MarkupContent")
	assert.Contains(t, content.Value, "--color-primary")
	assert.Contains(t, content.Value, "#ff0000")
	assert.Contains(t, content.Value, "Primary brand color")

	// Assert Range is present
	require.NotNil(t, hover.Range, "Range should be present for token reference")
}

func TestHover_CurlyBraceReference_YAML(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add tokens
	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:        "color-accent-base",
		Value:       "#0066cc",
		Type:        "color",
		Description: "Base accent color",
	}))

	// Open a YAML document with curly brace reference
	uri := "file:///tokens.yaml"
	yamlContent := `color:
  button:
    background:
      $value: "{color.accent.base}"`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "yaml", 1, yamlContent))

	// Hover over {color.accent.base}
	// Line 3: `      $value: "{color.accent.base}"`
	// Character 20 is inside "color.accent.base"
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 3, Character: 20},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)

	// Assert hover content
	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok, "Contents should be MarkupContent")
	assert.Contains(t, content.Value, "--color-accent-base")
	assert.Contains(t, content.Value, "#0066cc")
	assert.Contains(t, content.Value, "Base accent color")
}

func TestHover_JSONPointerReference(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add tokens
	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:        "spacing-large",
		Value:       "2rem",
		Type:        "dimension",
		Description: "Large spacing unit",
	}))

	// Open a JSON document with $ref (JSON Pointer reference)
	uri := "file:///tokens.json"
	jsonContent := `{
  "padding": {
    "card": {
      "$ref": "#/spacing/large"
    }
  }
}`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "json", 1, jsonContent))

	// Hover over "#/spacing/large"
	// Line 3: `      "$ref": "#/spacing/large"`
	// Character 20 is inside "spacing/large"
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 3, Character: 20},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)

	// Assert hover content
	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok, "Contents should be MarkupContent")
	assert.Contains(t, content.Value, "--spacing-large")
	assert.Contains(t, content.Value, "2rem")
	assert.Contains(t, content.Value, "Large spacing unit")
}

func TestHover_TokenReference_UnknownToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Open a JSON document with reference to unknown token
	uri := "file:///tokens.json"
	jsonContent := `{
  "color": {
    "alias": {
      "$value": "{unknown.token}"
    }
  }
}`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "json", 1, jsonContent))

	// Hover over {unknown.token}
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 3, Character: 20},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover, "Should show 'unknown token' message")

	// Assert unknown token message
	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content.Value, "Unknown token")
}

func TestHover_TokenReference_NoReferenceAtPosition(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Open a JSON document
	uri := "file:///tokens.json"
	jsonContent := `{
  "color": {
    "primary": {
      "$value": "#ff0000"
    }
  }
}`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "json", 1, jsonContent))

	// Hover over a position without a token reference
	// Line 3: `      "$value": "#ff0000"`
	// Position on "$value" key, not on a reference
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 3, Character: 10},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, hover, "Should not show hover when not on a reference")
}

func TestHover_TokenReference_DeprecatedToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add deprecated token
	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:               "color-old-primary",
		Value:              "#cc0000",
		Type:               "color",
		Deprecated:         true,
		DeprecationMessage: "Use color.primary instead",
	}))

	uri := "file:///tokens.yaml"
	yamlContent := `color:
  alias:
    $value: "{color.old.primary}"`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "yaml", 1, yamlContent))

	// Hover over the deprecated token reference
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 2, Character: 18},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)

	// Assert deprecation warning
	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content.Value, "DEPRECATED")
	assert.Contains(t, content.Value, "Use color.primary instead")
}

// parseTokensFile parses a DTCG token file and returns tokens indexed by CSS variable name.
func parseTokensFile(t *testing.T, path string) map[string]*tokens.Token {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	p := asimonim.NewJSONParser()
	toks, err := p.Parse(data, asimonim.Options{})
	require.NoError(t, err)
	require.NotEmpty(t, toks)

	byName := make(map[string]*tokens.Token, len(toks))
	for _, tok := range toks {
		byName[tok.Name] = tok
	}

	// Manually resolve the alias token for testing the resolved-value code path.
	if alias, ok := byName["color-alias"]; ok {
		if primary, ok := byName["color-primary"]; ok {
			alias.IsResolved = true
			alias.ResolvedValue = primary.RawValue
			alias.Type = primary.Type
		}
	}

	return byName
}

// TestHover_PlaintextUnknownToken tests the plaintext template for unknown tokens
func TestHover_PlaintextUnknownToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.SetPreferredHoverFormat(protocol.MarkupKindPlainText)
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///test.css"
	cssContent := `.button { color: var(--unknown-token); }`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	// unknown token with plaintext format
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 28},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)

	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Equal(t, protocol.MarkupKindPlainText, content.Kind)
	assert.Contains(t, content.Value, "Unknown token")
	assert.NotContains(t, content.Value, "**") // no markdown formatting
}

// TestHover_IsTokenFile_JSONC tests that jsonc files are recognized as token files
func TestHover_IsTokenFile(t *testing.T) {
	tests := []struct {
		name     string
		langID   string
		expected bool
	}{
		{"json", "json", true},
		{"jsonc", "jsonc", true},
		{"yaml", "yaml", true},
		{"css", "css", false},
		{"html", "html", false},
		{"javascript", "javascript", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isTokenFile(tt.langID))
		})
	}
}

// TestCalculateRangeSize tests calculateRangeSize for single-line and multi-line ranges
func TestCalculateRangeSize(t *testing.T) {
	t.Run("single line range", func(t *testing.T) {
		r := css.Range{
			Start: css.Position{Line: 0, Character: 5},
			End:   css.Position{Line: 0, Character: 25},
		}
		// single line: just character difference = 20
		assert.Equal(t, 20, calculateRangeSize(r))
	})

	t.Run("multi-line range", func(t *testing.T) {
		r := css.Range{
			Start: css.Position{Line: 0, Character: 10},
			End:   css.Position{Line: 2, Character: 5},
		}
		// multi-line: 2*10000 + (5-10) = 19995
		size := calculateRangeSize(r)
		assert.Greater(t, size, 10000, "Multi-line range should be large")
	})
}

// TestExtractColorDetails tests extractColorDetails with various edge cases
func TestExtractColorDetails(t *testing.T) {
	t.Run("non-color type returns nil", func(t *testing.T) {
		token := &tokens.Token{
			Name: "spacing.small",
			Type: "dimension",
			RawValue: map[string]any{
				"colorSpace": "srgb",
				"components": []any{1.0, 0.0, 0.0},
			},
		}
		assert.Nil(t, extractColorDetails(token))
	})

	t.Run("string value returns nil", func(t *testing.T) {
		// draft schema: color value is a string, not a map
		token := &tokens.Token{
			Name:     "color.simple",
			Type:     "color",
			RawValue: "#ff0000",
		}
		assert.Nil(t, extractColorDetails(token))
	})

	t.Run("missing colorSpace returns nil", func(t *testing.T) {
		token := &tokens.Token{
			Name: "color.bad",
			Type: "color",
			RawValue: map[string]any{
				"components": []any{1.0, 0.0, 0.0},
			},
		}
		assert.Nil(t, extractColorDetails(token))
	})

	t.Run("missing components returns nil", func(t *testing.T) {
		token := &tokens.Token{
			Name: "color.bad",
			Type: "color",
			RawValue: map[string]any{
				"colorSpace": "srgb",
			},
		}
		assert.Nil(t, extractColorDetails(token))
	})

	t.Run("uses resolved value when available", func(t *testing.T) {
		token := &tokens.Token{
			Name:       "color.alias",
			Type:       "color",
			IsResolved: true,
			ResolvedValue: map[string]any{
				"colorSpace": "display-p3",
				"components": []any{0.5, 0.3, 0.1},
				"alpha":      0.8,
			},
		}
		cd := extractColorDetails(token)
		require.NotNil(t, cd)
		assert.Equal(t, "display-p3", cd.ColorSpace)
		assert.Equal(t, "0.5, 0.3, 0.1", cd.Components)
		assert.Equal(t, "0.8", cd.Alpha)
	})

	t.Run("color with hex field", func(t *testing.T) {
		token := &tokens.Token{
			Name: "color.brand",
			Type: "color",
			RawValue: map[string]any{
				"colorSpace": "srgb",
				"components": []any{1.0, 0.0, 0.0},
				"hex":        "#ff0000",
			},
		}
		cd := extractColorDetails(token)
		require.NotNil(t, cd)
		assert.Equal(t, "#ff0000", cd.Hex)
	})
}

// TestFormatComponents tests formatComponents with various component types
func TestFormatComponents(t *testing.T) {
	t.Run("float64 components", func(t *testing.T) {
		result := formatComponents([]any{1.0, 0.5, 0.0})
		assert.Equal(t, "1, 0.5, 0", result)
	})

	t.Run("string components like none", func(t *testing.T) {
		result := formatComponents([]any{0.5, 0.3, "none"})
		assert.Equal(t, "0.5, 0.3, none", result)
	})

	t.Run("other types use default formatting", func(t *testing.T) {
		// integer component (not float64 or string)
		result := formatComponents([]any{42, true})
		assert.Equal(t, "42, true", result)
	})
}

// TestFindInnermostVarCall tests finding the innermost var() call at a position
func TestFindInnermostVarCall(t *testing.T) {
	t.Run("returns nil for empty list", func(t *testing.T) {
		result := findInnermostVarCall(protocol.Position{Line: 0, Character: 5}, nil)
		assert.Nil(t, result)
	})

	t.Run("returns nil when no var call contains position", func(t *testing.T) {
		varCalls := []*css.VarCall{
			{
				TokenName: "--color-primary",
				Range: css.Range{
					Start: css.Position{Line: 0, Character: 10},
					End:   css.Position{Line: 0, Character: 30},
				},
			},
		}
		result := findInnermostVarCall(protocol.Position{Line: 0, Character: 5}, varCalls)
		assert.Nil(t, result)
	})

	t.Run("returns innermost when nested", func(t *testing.T) {
		// outer: var(--outer, var(--inner))
		outerCall := &css.VarCall{
			TokenName: "--outer",
			Range: css.Range{
				Start: css.Position{Line: 0, Character: 5},
				End:   css.Position{Line: 0, Character: 50},
			},
		}
		innerCall := &css.VarCall{
			TokenName: "--inner",
			Range: css.Range{
				Start: css.Position{Line: 0, Character: 20},
				End:   css.Position{Line: 0, Character: 40},
			},
		}

		// position inside both ranges, should return inner (smaller)
		result := findInnermostVarCall(protocol.Position{Line: 0, Character: 25}, []*css.VarCall{outerCall, innerCall})
		require.NotNil(t, result)
		assert.Equal(t, "--inner", result.TokenName)
	})
}

// TestFindInnermostVariable tests finding the innermost variable declaration at a position
func TestFindInnermostVariable(t *testing.T) {
	t.Run("returns nil for empty list", func(t *testing.T) {
		result := findInnermostVariable(protocol.Position{Line: 0, Character: 5}, nil)
		assert.Nil(t, result)
	})

	t.Run("returns variable when position is inside", func(t *testing.T) {
		variables := []*css.Variable{
			{
				Name: "--color-primary",
				Range: css.Range{
					Start: css.Position{Line: 1, Character: 2},
					End:   css.Position{Line: 1, Character: 17},
				},
			},
		}
		result := findInnermostVariable(protocol.Position{Line: 1, Character: 5}, variables)
		require.NotNil(t, result)
		assert.Equal(t, "--color-primary", result.Name)
	})
}

// TestHover_TokenFileHover_PlaintextFormat tests plaintext hover for token references
func TestHover_TokenFileHover_PlaintextFormat(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.SetPreferredHoverFormat(protocol.MarkupKindPlainText)
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:        "color-primary",
		Value:       "#ff0000",
		Type:        "color",
		Description: "Primary brand color",
	}))

	uri := "file:///tokens.json"
	jsonContent := `{
  "color": {
    "secondary": {
      "$value": "{color.primary}"
    }
  }
}`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "json", 1, jsonContent))

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 3, Character: 20},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)

	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Equal(t, protocol.MarkupKindPlainText, content.Kind)
	assert.NotContains(t, content.Value, "**") // no markdown
	assert.Contains(t, content.Value, "Value (CSS):")
}

// TestHover_PlaintextUnknownTokenRef tests the plaintext unknown token message for token references
func TestHover_PlaintextUnknownTokenRef(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.SetPreferredHoverFormat(protocol.MarkupKindPlainText)
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///tokens.json"
	jsonContent := `{
  "color": {
    "alias": {
      "$value": "{unknown.token}"
    }
  }
}`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "json", 1, jsonContent))

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 3, Character: 20},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)

	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Equal(t, protocol.MarkupKindPlainText, content.Kind)
	assert.Contains(t, content.Value, "Unknown token")
	assert.NotContains(t, content.Value, "**")
}

// TestHover_UnsupportedLanguage tests that hover returns nil for unsupported languages
func TestHover_UnsupportedLanguage(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///test.py"
	content := `print("hello")`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "python", 1, content))

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 5},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, hover)
}

// TestIsPositionInRange_MultiLine tests isPositionInRange with multi-line ranges
func TestIsPositionInRange_MultiLine(t *testing.T) {
	// Multi-line range: from line 1 char 5 to line 3 char 10
	r := css.Range{
		Start: css.Position{Line: 1, Character: 5},
		End:   css.Position{Line: 3, Character: 10},
	}

	tests := []struct {
		name     string
		pos      protocol.Position
		expected bool
	}{
		{"before start line", protocol.Position{Line: 0, Character: 5}, false},
		{"on start line, before start char", protocol.Position{Line: 1, Character: 4}, false},
		{"on start line, at start char", protocol.Position{Line: 1, Character: 5}, true},
		{"on middle line", protocol.Position{Line: 2, Character: 0}, true},
		{"on end line, before end char", protocol.Position{Line: 3, Character: 5}, true},
		{"on end line, at end char (excluded)", protocol.Position{Line: 3, Character: 10}, false},
		{"after end line", protocol.Position{Line: 4, Character: 0}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isPositionInRange(tt.pos, r))
		})
	}
}

func TestRenderTokenHover_StructuredColor(t *testing.T) {
	tokens2025 := parseTokensFile(t, "testdata/tokens-2025.json")
	tokensDraft := parseTokensFile(t, "testdata/tokens-draft.json")

	tests := []struct {
		name      string
		tokenName string
		tokens    map[string]*tokens.Token
		golden    string
		format    protocol.MarkupKind
	}{
		{"srgb color", "color-primary", tokens2025, "testdata/golden/color-primary.md", protocol.MarkupKindMarkdown},
		{"display-p3 color", "color-accent", tokens2025, "testdata/golden/color-accent.md", protocol.MarkupKindMarkdown},
		{"color with hex field", "color-brand", tokens2025, "testdata/golden/color-brand.md", protocol.MarkupKindMarkdown},
		{"color with none component", "color-achromatic", tokens2025, "testdata/golden/color-achromatic.md", protocol.MarkupKindMarkdown},
		{"color without alpha", "color-no-alpha", tokens2025, "testdata/golden/color-no-alpha.md", protocol.MarkupKindMarkdown},
		{"string color (draft schema)", "color-simple", tokensDraft, "testdata/golden/color-simple.md", protocol.MarkupKindMarkdown},
		{"non-color token", "spacing-large", tokens2025, "testdata/golden/spacing-large.md", protocol.MarkupKindMarkdown},
		{"srgb color plaintext", "color-primary", tokens2025, "testdata/golden/color-primary.txt", protocol.MarkupKindPlainText},
		{"resolved alias color", "color-alias", tokens2025, "testdata/golden/color-alias.md", protocol.MarkupKindMarkdown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, ok := tt.tokens[tt.tokenName]
			require.True(t, ok, "token %q not found in fixture", tt.tokenName)

			content, err := renderTokenHover(token, tt.format)
			require.NoError(t, err)

			if *update {
				err := os.WriteFile(tt.golden, []byte(content), 0o644)
				require.NoError(t, err)
				return
			}

			expected, err := os.ReadFile(tt.golden)
			require.NoError(t, err, "golden file %s not found; run with --update to create", tt.golden)
			assert.Equal(t, string(expected), content)
		})
	}
}
