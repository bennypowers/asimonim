package documentcolor

import (
	"testing"

	"bennypowers.dev/asimonim/lsp/internal/tokens"
	"bennypowers.dev/asimonim/lsp/testutil"
	"bennypowers.dev/asimonim/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestDocumentColor_ColorTokenInVar(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a color token
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
		Type:  "color",
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := DocumentColor(req, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result, 1)

	// Check color value
	assert.Equal(t, protocol.Decimal(1.0), result[0].Color.Red)
	assert.Equal(t, protocol.Decimal(0.0), result[0].Color.Green)
	assert.Equal(t, protocol.Decimal(0.0), result[0].Color.Blue)
	assert.Equal(t, protocol.Decimal(1.0), result[0].Color.Alpha)
}

func TestDocumentColor_ColorTokenInDeclaration(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a color token
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#00ff00",
		Type:  "color",
	})

	uri := "file:///test.css"
	cssContent := `:root { --color-primary: #00ff00; }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := DocumentColor(req, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.GreaterOrEqual(t, len(result), 1)

	// Check that we found a green color
	foundGreen := false
	for _, colorInfo := range result {
		if colorInfo.Color.Green == 1.0 && colorInfo.Color.Red == 0.0 && colorInfo.Color.Blue == 0.0 {
			foundGreen = true
			break
		}
	}
	assert.True(t, foundGreen, "Should find green color in declarations")
}

func TestDocumentColor_NonColorToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a non-color token
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "spacing.small",
		Value: "8px",
		Type:  "dimension",
	})

	uri := "file:///test.css"
	cssContent := `.button { padding: var(--spacing-small); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := DocumentColor(req, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})

	require.NoError(t, err)
	assert.Empty(t, result) // Should not include non-color tokens
}

func TestDocumentColor_NonCSSDocument(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///test.json"
	jsonContent := `{"color": {"$value": "#ff0000"}}`
	_ = ctx.DocumentManager().DidOpen(uri, "json", 1, jsonContent)

	result, err := DocumentColor(req, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDocumentColor_DocumentNotFound(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	result, err := DocumentColor(req, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.css"},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestColorPresentation_MatchingTokens(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add multiple tokens with red color
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
		Type:  "color",
	})
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.danger",
		Value: "rgb(255, 0, 0)", // Same color, different format
		Type:  "color",
	})
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.safe",
		Value: "#00ff00", // Different color
		Type:  "color",
	})

	// Request presentations for red color
	result, err := ColorPresentation(req, &protocol.ColorPresentationParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
		Color: protocol.Color{
			Red:   1.0,
			Green: 0.0,
			Blue:  0.0,
			Alpha: 1.0,
		},
	})

	require.NoError(t, err)
	require.Len(t, result, 2) // Should match color.primary and color.danger

	// Check that we got token names, not format strings
	labels := make([]string, len(result))
	for i, p := range result {
		labels[i] = p.Label
	}

	assert.Contains(t, labels, "color.primary")
	assert.Contains(t, labels, "color.danger")
	assert.NotContains(t, labels, "color.safe") // Green should not match
}

func TestColorPresentation_WithAlpha(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add tokens with alpha channel (using same hex value to ensure exact match)
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.overlay",
		Value: "#ff000080", // Red with alpha=0x80/255≈0.502
		Type:  "color",
	})
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.transparent",
		Value: "rgba(255, 0, 0, 0.5)", // csscolorparser converts to #ff000080
		Type:  "color",
	})
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.opaque",
		Value: "#ff0000", // Same color but fully opaque - should NOT match
		Type:  "color",
	})

	// Request presentations for semi-transparent red
	// Alpha 0.5 will be converted to #ff000080 by csscolorparser
	result, err := ColorPresentation(req, &protocol.ColorPresentationParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
		Color: protocol.Color{
			Red:   1.0,
			Green: 0.0,
			Blue:  0.0,
			Alpha: 0.5,
		},
	})

	require.NoError(t, err)
	require.Len(t, result, 2) // Should match overlay and transparent

	labels := make([]string, len(result))
	for i, p := range result {
		labels[i] = p.Label
	}

	// Should match tokens with alpha, not opaque ones
	assert.Contains(t, labels, "color.overlay")
	assert.Contains(t, labels, "color.transparent")
	assert.NotContains(t, labels, "color.opaque") // Different alpha should not match
}

// TestParseColor tests the parseColor helper function
func TestParseColor(t *testing.T) {
	tests := []struct {
		expected    *protocol.Color
		name        string
		input       string
		expectError bool
	}{
		{
			name:  "6-digit hex color",
			input: "#ff0000",
			expected: &protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "6-digit hex color uppercase",
			input: "#FF0000",
			expected: &protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "6-digit hex color with whitespace",
			input: "  #00ff00  ",
			expected: &protocol.Color{
				Red:   0.0,
				Green: 1.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "3-digit hex color",
			input: "#f00",
			expected: &protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "3-digit hex color - blue",
			input: "#00f",
			expected: &protocol.Color{
				Red:   0.0,
				Green: 0.0,
				Blue:  1.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "8-digit hex color with alpha",
			input: "#ff000080",
			expected: &protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: protocol.Decimal(128.0 / 255.0), // ~0.502
			},
			expectError: false,
		},
		{
			name:  "8-digit hex color with full alpha",
			input: "#0000ffff",
			expected: &protocol.Color{
				Red:   0.0,
				Green: 0.0,
				Blue:  1.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "8-digit hex color with zero alpha",
			input: "#ff000000",
			expected: &protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 0.0,
			},
			expectError: false,
		},
		{
			name:  "4-digit hex color (#RGBA) - red with full alpha",
			input: "#f00f",
			expected: &protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "4-digit hex color (#RGBA) - blue with half alpha",
			input: "#00f8",
			expected: &protocol.Color{
				Red:   0.0,
				Green: 0.0,
				Blue:  1.0,
				Alpha: protocol.Decimal(136.0 / 255.0), // 0x88 = 136
			},
			expectError: false,
		},
		{
			name:  "4-digit hex color (#RGBA) - green with zero alpha",
			input: "#0f00",
			expected: &protocol.Color{
				Red:   0.0,
				Green: 1.0,
				Blue:  0.0,
				Alpha: 0.0,
			},
			expectError: false,
		},
		{
			name:  "4-digit hex color (#RGBA) - gray with half alpha",
			input: "#8888",
			expected: &protocol.Color{
				Red:   protocol.Decimal(136.0 / 255.0),
				Green: protocol.Decimal(136.0 / 255.0),
				Blue:  protocol.Decimal(136.0 / 255.0),
				Alpha: protocol.Decimal(136.0 / 255.0),
			},
			expectError: false,
		},
		{
			name:        "invalid hex - 5 digits (unsupported length)",
			input:       "#ff000",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "invalid hex - non-hex characters",
			input:       "#gggggg",
			expected:    nil,
			expectError: true,
		},
		{
			name:  "rgb() format - red",
			input: "rgb(255, 0, 0)",
			expected: &protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "rgb() format - green with spaces",
			input: "rgb(0, 255, 0)",
			expected: &protocol.Color{
				Red:   0.0,
				Green: 1.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "rgb() format - blue",
			input: "rgb(0,0,255)",
			expected: &protocol.Color{
				Red:   0.0,
				Green: 0.0,
				Blue:  1.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "rgba() format - red with half alpha",
			input: "rgba(255, 0, 0, 0.5)",
			expected: &protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 0.5,
			},
			expectError: false,
		},
		{
			name:  "rgba() format - blue with full alpha",
			input: "rgba(0, 0, 255, 1.0)",
			expected: &protocol.Color{
				Red:   0.0,
				Green: 0.0,
				Blue:  1.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "rgba() format - gray with zero alpha",
			input: "rgba(128, 128, 128, 0)",
			expected: &protocol.Color{
				Red:   protocol.Decimal(128.0 / 255.0),
				Green: protocol.Decimal(128.0 / 255.0),
				Blue:  protocol.Decimal(128.0 / 255.0),
				Alpha: 0.0,
			},
			expectError: false,
		},
		{
			name:  "hsl() format - red (0°, 100%, 50%)",
			input: "hsl(0, 100%, 50%)",
			expected: &protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "hsl() format - green (120°, 100%, 50%)",
			input: "hsl(120, 100%, 50%)",
			expected: &protocol.Color{
				Red:   0.0,
				Green: 1.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "hsl() format - blue (240°, 100%, 50%)",
			input: "hsl(240, 100%, 50%)",
			expected: &protocol.Color{
				Red:   0.0,
				Green: 0.0,
				Blue:  1.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "hsla() format - red with half alpha",
			input: "hsla(0, 100%, 50%, 0.5)",
			expected: &protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 0.5,
			},
			expectError: false,
		},
		{
			name:  "hsla() format - cyan with alpha",
			input: "hsla(180, 100%, 50%, 0.8)",
			expected: &protocol.Color{
				Red:   0.0,
				Green: 1.0,
				Blue:  1.0,
				Alpha: 0.8,
			},
			expectError: false,
		},
		{
			name:  "named color - red",
			input: "red",
			expected: &protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:        "empty string",
			input:       "",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "just hash",
			input:       "#",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseColor(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)

				// Compare with small tolerance for floating point
				const tolerance = 0.001
				assert.InDelta(t, tt.expected.Red, result.Red, tolerance, "Red channel mismatch")
				assert.InDelta(t, tt.expected.Green, result.Green, tolerance, "Green channel mismatch")
				assert.InDelta(t, tt.expected.Blue, result.Blue, tolerance, "Blue channel mismatch")
				assert.InDelta(t, tt.expected.Alpha, result.Alpha, tolerance, "Alpha channel mismatch")
			}
		})
	}
}

func TestDocumentColor_HTMLDocument(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
		Type:  "color",
	})

	uri := "file:///test.html"
	content := `<style>.btn { color: var(--color-primary); }</style>`
	_ = ctx.DocumentManager().DidOpen(uri, "html", 1, content)

	colors, err := DocumentColor(req, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	require.NoError(t, err)
	require.Len(t, colors, 1)
	assert.InDelta(t, 1.0, colors[0].Color.Red, 0.01)
}

func TestDocumentColor_UnparseableColorToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a color token with an unparseable value
	require.NoError(t, ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.weird",
		Value: "not-a-valid-color-value-xyz",
		Type:  "color",
	}))

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-weird); }`
	require.NoError(t, ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent))

	result, err := DocumentColor(req, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})

	// Should not error, just skip the unparseable color
	require.NoError(t, err)
	assert.Empty(t, result, "Unparseable color should be skipped")
	// Should have a warning
	assert.NotEmpty(t, req.Warnings(), "Should have a warning for unparseable color")
}

func TestDocumentColor_UnparseableColorInDeclaration(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a color token with an unparseable value
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.bad",
		Value: "not-a-color!!",
		Type:  "color",
	})

	uri := "file:///test.css"
	cssContent := `:root { --color-bad: something; }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := DocumentColor(req, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})

	require.NoError(t, err)
	assert.Empty(t, result, "Unparseable color in declaration should be skipped")
	assert.NotEmpty(t, req.Warnings(), "Should have a warning for unparseable declaration color")
}

func TestDocumentColor_NonColorTokenInDeclaration(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a non-color token
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "spacing.small",
		Value: "8px",
		Type:  "dimension",
	})

	uri := "file:///test.css"
	cssContent := `:root { --spacing-small: 8px; }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := DocumentColor(req, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})

	require.NoError(t, err)
	assert.Empty(t, result, "Non-color token in declaration should not produce color info")
}

func TestDocumentColor_UnknownTokenInDeclaration(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Don't add any tokens - --local-var is unknown
	uri := "file:///test.css"
	cssContent := `:root { --local-var: blue; }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := DocumentColor(req, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})

	require.NoError(t, err)
	assert.Empty(t, result, "Unknown token in declaration should not produce color info")
}

func TestDocumentColor_UnknownTokenInVarCall(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// No tokens loaded
	uri := "file:///test.css"
	cssContent := `.button { color: var(--unknown-thing); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := DocumentColor(req, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})

	require.NoError(t, err)
	assert.Empty(t, result, "Unknown token var() call should not produce color info")
}

func TestDocumentColor_MultipleColorsAndNonColors(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
		Type:  "color",
	})
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "spacing.large",
		Value: "2rem",
		Type:  "dimension",
	})
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.secondary",
		Value: "#0000ff",
		Type:  "color",
	})

	uri := "file:///test.css"
	cssContent := `.card {
  color: var(--color-primary);
  padding: var(--spacing-large);
  background: var(--color-secondary);
}`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := DocumentColor(req, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})

	require.NoError(t, err)
	// Should find 2 colors (not the spacing token)
	require.Len(t, result, 2)
	// Verify exactly one red and one blue (no duplicates)
	seenRed, seenBlue := false, false
	for _, ci := range result {
		r, g, b := ci.Color.Red, ci.Color.Green, ci.Color.Blue
		if r > 0.9 && g < 0.1 && b < 0.1 {
			seenRed = true
		} else if r < 0.1 && g < 0.1 && b > 0.9 {
			seenBlue = true
		} else {
			t.Errorf("unexpected color: R=%.2f G=%.2f B=%.2f", r, g, b)
		}
	}
	assert.True(t, seenRed, "expected red color")
	assert.True(t, seenBlue, "expected blue color")
}

func TestColorPresentation_NoMatchingTokens(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
		Type:  "color",
	})

	// Request presentations for green - no tokens should match
	result, err := ColorPresentation(req, &protocol.ColorPresentationParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
		Color: protocol.Color{
			Red:   0.0,
			Green: 1.0,
			Blue:  0.0,
			Alpha: 1.0,
		},
	})

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestColorPresentation_NonColorTokensIgnored(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "spacing.small",
		Value: "8px",
		Type:  "dimension",
	})

	result, err := ColorPresentation(req, &protocol.ColorPresentationParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
		Color: protocol.Color{
			Red:   1.0,
			Green: 0.0,
			Blue:  0.0,
			Alpha: 1.0,
		},
	})

	require.NoError(t, err)
	assert.Empty(t, result, "Non-color tokens should not appear in color presentations")
}

func TestColorPresentation_UnparseableTokenColor(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.weird",
		Value: "not-a-parseable-color",
		Type:  "color",
	})

	result, err := ColorPresentation(req, &protocol.ColorPresentationParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
		Color: protocol.Color{
			Red:   1.0,
			Green: 0.0,
			Blue:  0.0,
			Alpha: 1.0,
		},
	})

	require.NoError(t, err)
	assert.Empty(t, result, "Unparseable color tokens should be skipped")
	assert.NotEmpty(t, req.Warnings(), "Should have a warning for unparseable color")
}

func TestDocumentColor_JSDocument(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
		Type:  "color",
	})

	uri := "file:///test.js"
	content := "const s = css`\n  .card { color: var(--color-primary); }\n`;"
	_ = ctx.DocumentManager().DidOpen(uri, "javascript", 1, content)

	colors, err := DocumentColor(req, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	require.NoError(t, err)
	require.Len(t, colors, 1)
	assert.InDelta(t, 1.0, colors[0].Color.Red, 0.01)
}

func TestDocumentColor_HTMLNoCSS(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///test.html"
	content := `<p>Hello</p>`
	_ = ctx.DocumentManager().DidOpen(uri, "html", 1, content)

	colors, err := DocumentColor(req, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	require.NoError(t, err)
	assert.Empty(t, colors)
}
