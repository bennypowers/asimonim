package codeaction

import (
	"testing"

	cssparser "bennypowers.dev/asimonim/lsp/internal/parser/css"
	"bennypowers.dev/asimonim/lsp/internal/tokens"
	"bennypowers.dev/asimonim/lsp/testutil"
	"bennypowers.dev/asimonim/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestCodeAction_IncorrectFallback(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.SetSupportsCodeActionLiterals(true)
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a color token
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#0000ff",
		Type:  "color",
	})

	uri := "file:///test.css"
	// Incorrect fallback: token is #0000ff but fallback is #ff0000
	cssContent := `.button { color: var(--color-primary, #ff0000); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 17},
			End:   protocol.Position{Line: 0, Character: 45},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{},
		},
	})

	require.NoError(t, err)
	actions, ok := result.([]protocol.CodeAction)
	require.True(t, ok)
	require.NotEmpty(t, actions)

	// Should have a fix fallback action with the correct value
	var fixAction *protocol.CodeAction
	for i := range actions {
		if actions[i].Title == "Fix fallback value to '#0000ff'" {
			fixAction = &actions[i]
			break
		}
	}
	require.NotNil(t, fixAction, "Should have 'Fix fallback value to '#0000ff'' action")

	// Verify it's a quick fix
	assert.NotNil(t, fixAction.Kind)
	assert.Equal(t, protocol.CodeActionKindQuickFix, *fixAction.Kind)

	// Verify the edit contains the correct replacement
	require.NotNil(t, fixAction.Edit)
	require.NotNil(t, fixAction.Edit.Changes)
	edits := fixAction.Edit.Changes[uri]
	require.Len(t, edits, 1)
	assert.Contains(t, edits[0].NewText, "var(--color-primary, #0000ff)")
}

func TestCodeAction_AddFallback(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.SetSupportsCodeActionLiterals(true)
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a color token
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#0000ff",
		Type:  "color",
	})

	uri := "file:///test.css"
	// No fallback provided
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 17},
			End:   protocol.Position{Line: 0, Character: 36},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{},
		},
	})

	require.NoError(t, err)
	actions, ok := result.([]protocol.CodeAction)
	require.True(t, ok)
	require.NotEmpty(t, actions)

	// Should suggest adding a fallback with the correct value
	var addAction *protocol.CodeAction
	for i := range actions {
		if actions[i].Title == "Add fallback value '#0000ff'" {
			addAction = &actions[i]
			break
		}
	}
	require.NotNil(t, addAction, "Should have 'Add fallback value '#0000ff'' action")

	// Verify the edit contains the correct value
	require.NotNil(t, addAction.Edit)
	require.NotNil(t, addAction.Edit.Changes)
	edits := addAction.Edit.Changes[uri]
	require.Len(t, edits, 1)
	assert.Contains(t, edits[0].NewText, "var(--color-primary, #0000ff)")
}

func TestCodeAction_NonCSSDocument(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///test.json"
	jsonContent := `{"color": {"$value": "#ff0000"}}`
	_ = ctx.DocumentManager().DidOpen(uri, "json", 1, jsonContent)

	result, err := CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 10},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestCodeAction_DocumentNotFound(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	result, err := CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.css"},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 10},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestCodeAction_OutsideRange(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.SetSupportsCodeActionLiterals(true)
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#0000ff",
		Type:  "color",
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary, #ff0000); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	// Request range that doesn't intersect with var()
	result, err := CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 7}, // Before var()
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{},
		},
	})

	require.NoError(t, err)
	actions, ok := result.([]protocol.CodeAction)
	require.True(t, ok)
	assert.Empty(t, actions) // No actions for range outside var()
}

func TestCodeActionResolve_ReturnsActionUnchanged(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	action := &protocol.CodeAction{
		Title: "Test action",
		Kind:  ptrCodeActionKind(protocol.CodeActionKindQuickFix),
	}

	resolved, err := CodeActionResolve(req, action)

	require.NoError(t, err)
	assert.Equal(t, action, resolved) // Should return same action
}

// ptrCodeActionKind returns a pointer to the given CodeActionKind
func ptrCodeActionKind(kind protocol.CodeActionKind) *protocol.CodeActionKind {
	return &kind
}

func TestExtractRecommendedToken(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "empty message",
			message:  "",
			expected: "",
		},
		{
			name:     "Use X instead pattern",
			message:  "Use color.secondary instead",
			expected: "color.secondary",
		},
		{
			name:     "Use X.Y.Z instead pattern with nested path",
			message:  "Use brand.color.primary instead",
			expected: "brand.color.primary",
		},
		{
			name:     "Replaced by X pattern",
			message:  "Replaced by color.accent",
			expected: "color.accent",
		},
		{
			name:     "Replaced by X with trailing text",
			message:  "Replaced by color.accent in version 2",
			expected: "color.accent",
		},
		{
			name:     "no recognized pattern",
			message:  "This token is deprecated",
			expected: "",
		},
		{
			name:     "Use pattern without instead suffix",
			message:  "Use caution when deploying",
			expected: "",
		},
		{
			name:     "Use X instead with leading text",
			message:  "Deprecated. Use spacing.lg instead",
			expected: "spacing.lg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRecommendedToken(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateDeprecatedTokenActions(t *testing.T) {
	t.Run("deprecated token with recommended replacement", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		ctx.SetSupportsCodeActionLiterals(true)
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		// Add deprecated token and its replacement
		_ = ctx.TokenManager().Add(&tokens.Token{
			Name:               "color.old",
			Value:              "#ff0000",
			Type:               "color",
			Deprecated:         true,
			DeprecationMessage: "Use color.new instead",
		})
		_ = ctx.TokenManager().Add(&tokens.Token{
			Name:  "color.new",
			Value: "#00ff00",
			Type:  "color",
		})

		uri := "file:///test.css"
		// var() referencing the deprecated token
		cssContent := `.button { color: var(--color-old, #ff0000); }`
		_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

		result, err := CodeAction(req, &protocol.CodeActionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 17},
				End:   protocol.Position{Line: 0, Character: 44},
			},
			Context: protocol.CodeActionContext{
				Diagnostics: []protocol.Diagnostic{
					{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 17},
							End:   protocol.Position{Line: 0, Character: 44},
						},
						Message: "Token is deprecated",
						Code:    &protocol.IntegerOrString{Value: "deprecated-token"},
					},
				},
			},
		})

		require.NoError(t, err)
		actions, ok := result.([]protocol.CodeAction)
		require.True(t, ok)
		require.NotEmpty(t, actions)

		// Should have a replacement action for the recommended token
		var replaceAction *protocol.CodeAction
		for i := range actions {
			if actions[i].Title == "Replace with '--color-new'" {
				replaceAction = &actions[i]
				break
			}
		}
		require.NotNil(t, replaceAction, "Should have replacement action for --color-new")
		assert.NotNil(t, replaceAction.Kind)
		assert.Equal(t, protocol.CodeActionKindQuickFix, *replaceAction.Kind)
		require.NotNil(t, replaceAction.Edit)
		edits := replaceAction.Edit.Changes[uri]
		require.Len(t, edits, 1)
		// Has fallback, so replacement should include formatted fallback from replacement token
		assert.Equal(t, "var(--color-new, #00ff00)", edits[0].NewText)

		// Should also have matching diagnostic and isPreferred
		require.NotEmpty(t, replaceAction.Diagnostics)
		require.NotNil(t, replaceAction.IsPreferred)
		assert.True(t, *replaceAction.IsPreferred)

		// Should also have a literal value action
		var literalAction *protocol.CodeAction
		for i := range actions {
			if actions[i].Title == "Replace with literal value '#ff0000'" {
				literalAction = &actions[i]
				break
			}
		}
		require.NotNil(t, literalAction, "Should have literal value action")
		require.NotNil(t, literalAction.Edit)
		literalEdits := literalAction.Edit.Changes[uri]
		require.Len(t, literalEdits, 1)
		assert.Equal(t, "#ff0000", literalEdits[0].NewText)
	})

	t.Run("deprecated token without recommended replacement", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		ctx.SetSupportsCodeActionLiterals(true)
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		// Deprecated token with no recommendation
		_ = ctx.TokenManager().Add(&tokens.Token{
			Name:               "color.legacy",
			Value:              "#aabbcc",
			Type:               "color",
			Deprecated:         true,
			DeprecationMessage: "This token is deprecated",
		})

		uri := "file:///test.css"
		cssContent := `.card { color: var(--color-legacy); }`
		_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

		result, err := CodeAction(req, &protocol.CodeActionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 15},
				End:   protocol.Position{Line: 0, Character: 34},
			},
			Context: protocol.CodeActionContext{
				Diagnostics: []protocol.Diagnostic{},
			},
		})

		require.NoError(t, err)
		actions, ok := result.([]protocol.CodeAction)
		require.True(t, ok)

		// Should have literal value action but no replacement action
		var literalAction *protocol.CodeAction
		for i := range actions {
			if actions[i].Title == "Replace with literal value '#aabbcc'" {
				literalAction = &actions[i]
				break
			}
		}
		require.NotNil(t, literalAction, "Should have literal value action")

		// Should NOT have a replacement action (no recommendation found)
		for _, action := range actions {
			assert.NotContains(t, action.Title, "Replace with '--")
		}
	})

	t.Run("deprecated token with replacement but no fallback in var()", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		ctx.SetSupportsCodeActionLiterals(true)
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		_ = ctx.TokenManager().Add(&tokens.Token{
			Name:               "spacing.old",
			Value:              "8px",
			Type:               "dimension",
			Deprecated:         true,
			DeprecationMessage: "Replaced by spacing.medium",
		})
		_ = ctx.TokenManager().Add(&tokens.Token{
			Name:  "spacing.medium",
			Value: "12px",
			Type:  "dimension",
		})

		uri := "file:///test.css"
		// No fallback in var() call
		cssContent := `.box { padding: var(--spacing-old); }`
		_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

		result, err := CodeAction(req, &protocol.CodeActionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 16},
				End:   protocol.Position{Line: 0, Character: 34},
			},
			Context: protocol.CodeActionContext{
				Diagnostics: []protocol.Diagnostic{},
			},
		})

		require.NoError(t, err)
		actions, ok := result.([]protocol.CodeAction)
		require.True(t, ok)

		// Replacement action should not include fallback (original had none)
		var replaceAction *protocol.CodeAction
		for i := range actions {
			if actions[i].Title == "Replace with '--spacing-medium'" {
				replaceAction = &actions[i]
				break
			}
		}
		require.NotNil(t, replaceAction, "Should have replacement action")
		edits := replaceAction.Edit.Changes[uri]
		require.Len(t, edits, 1)
		// No fallback in original, so replacement should not have fallback
		assert.Equal(t, "var(--spacing-medium)", edits[0].NewText)
	})
}

func TestCreateReplacementAction_UnformattableToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.SetSupportsCodeActionLiterals(true)
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Deprecated token pointing to a replacement with unsupported type
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:               "border.old",
		Value:              "1px solid #000",
		Type:               "border",
		Deprecated:         true,
		DeprecationMessage: "Use border.new instead",
	})
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "border.new",
		Value: "2px solid #fff",
		Type:  "border", // composite type, cannot be formatted for CSS
	})

	uri := "file:///test.css"
	// Has fallback, which triggers FormatTokenValueForCSS on the replacement token
	cssContent := `.box { border: var(--border-old, 1px solid #000); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 15},
			End:   protocol.Position{Line: 0, Character: 49},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{},
		},
	})

	require.NoError(t, err)
	actions, ok := result.([]protocol.CodeAction)
	require.True(t, ok)

	// Should NOT have replacement action (replacement token is composite type)
	for _, action := range actions {
		assert.NotContains(t, action.Title, "Replace with '--border-new'",
			"Should not offer replacement when token value can't be formatted")
	}
}

func TestCreateLiteralValueAction_UnformattableToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.SetSupportsCodeActionLiterals(true)
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Deprecated token with composite type that can't be formatted
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:               "shadow.old",
		Value:              "0 2px 4px rgba(0,0,0,0.2)",
		Type:               "shadow",
		Deprecated:         true,
		DeprecationMessage: "This token is deprecated",
	})

	uri := "file:///test.css"
	cssContent := `.card { box-shadow: var(--shadow-old); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 20},
			End:   protocol.Position{Line: 0, Character: 37},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{},
		},
	})

	require.NoError(t, err)
	actions, ok := result.([]protocol.CodeAction)
	require.True(t, ok)

	// Should NOT have literal value action (shadow is composite type)
	for _, action := range actions {
		assert.NotContains(t, action.Title, "Replace with literal value",
			"Should not offer literal value when token type is unsupported")
	}
}

func TestFixFallbackAction_WithMatchingDiagnostic(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.SetSupportsCodeActionLiterals(true)
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#0000ff",
		Type:  "color",
	})

	uri := "file:///test.css"
	// Incorrect fallback: token is #0000ff but fallback is red
	cssContent := `.button { color: var(--color-primary, red); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	// Include a matching diagnostic at the var() call position
	diag := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 17},
			End:   protocol.Position{Line: 0, Character: 43},
		},
		Message: "Incorrect fallback value",
		Code:    &protocol.IntegerOrString{Value: "incorrect-fallback"},
	}

	result, err := CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 17},
			End:   protocol.Position{Line: 0, Character: 43},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{diag},
		},
	})

	require.NoError(t, err)
	actions, ok := result.([]protocol.CodeAction)
	require.True(t, ok)

	var fixAction *protocol.CodeAction
	for i := range actions {
		if actions[i].Title == "Fix fallback value to '#0000ff'" {
			fixAction = &actions[i]
			break
		}
	}
	require.NotNil(t, fixAction, "Should have fix fallback action")

	// When diagnostic matches, action should include it and be preferred
	require.NotEmpty(t, fixAction.Diagnostics)
	assert.Equal(t, diag.Message, fixAction.Diagnostics[0].Message)
	require.NotNil(t, fixAction.IsPreferred)
	assert.True(t, *fixAction.IsPreferred)
}

func TestFixFallbackAction_UnformattableToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.SetSupportsCodeActionLiterals(true)
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Token with composite type that can't be formatted for CSS fallback
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "border.primary",
		Value: "1px solid #000",
		Type:  "border",
	})

	uri := "file:///test.css"
	cssContent := `.box { border: var(--border-primary, 2px dashed red); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 15},
			End:   protocol.Position{Line: 0, Character: 53},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{},
		},
	})

	require.NoError(t, err)
	actions, ok := result.([]protocol.CodeAction)
	require.True(t, ok)

	// Should NOT have fix fallback action (border can't be formatted)
	for _, action := range actions {
		assert.NotContains(t, action.Title, "Fix fallback value",
			"Should not offer fix fallback when token type is unsupported")
	}
}

func TestCreateAddFallbackAction_UnformattableToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Token with composite type that FormatTokenValueForCSS rejects
	token := &tokens.Token{
		Name:  "border.main",
		Value: "1px solid #000",
		Type:  "border",
	}

	varCall := cssparser.VarCall{
		TokenName: "--border-main",
		Range: cssparser.Range{
			Start: cssparser.Position{Line: 0, Character: 15},
			End:   cssparser.Position{Line: 0, Character: 35},
		},
	}

	// Directly call createAddFallbackAction with a token type that can't be formatted
	result := createAddFallbackAction(req, "file:///test.css", varCall, token)
	assert.Nil(t, result, "Should return nil when token value can't be formatted for CSS")
}

func TestCreateFixAllActionIfNeeded(t *testing.T) {
	t.Run("returns nil with fewer than 2 diagnostics", func(t *testing.T) {
		diags := []protocol.Diagnostic{
			{Code: &protocol.IntegerOrString{Value: "incorrect-fallback"}},
		}
		result := createFixAllActionIfNeeded("file:///test.css", nil, diags)
		assert.Nil(t, result)
	})

	t.Run("returns nil when fewer than 2 incorrect-fallback diagnostics", func(t *testing.T) {
		diags := []protocol.Diagnostic{
			{Code: &protocol.IntegerOrString{Value: "incorrect-fallback"}},
			{Code: &protocol.IntegerOrString{Value: "deprecated-token"}},
		}
		result := createFixAllActionIfNeeded("file:///test.css", nil, diags)
		assert.Nil(t, result)
	})

	t.Run("returns action when 2+ incorrect-fallback diagnostics", func(t *testing.T) {
		diags := []protocol.Diagnostic{
			{Code: &protocol.IntegerOrString{Value: "incorrect-fallback"}},
			{Code: &protocol.IntegerOrString{Value: "incorrect-fallback"}},
		}
		result := createFixAllActionIfNeeded("file:///test.css", nil, diags)
		require.NotNil(t, result)
		assert.Equal(t, "Fix all token fallback values", result.Title)
	})

	t.Run("returns nil with nil diagnostic code", func(t *testing.T) {
		diags := []protocol.Diagnostic{
			{Code: nil},
			{Code: nil},
		}
		result := createFixAllActionIfNeeded("file:///test.css", nil, diags)
		assert.Nil(t, result)
	})
}

func TestResolveFixAllFallbacks(t *testing.T) {
	t.Run("returns action unchanged when data is not map", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		action := &protocol.CodeAction{
			Title: "Fix all token fallback values",
			Data:  "not-a-map",
		}

		resolved, err := resolveFixAllFallbacks(req, action)
		require.NoError(t, err)
		assert.Equal(t, action, resolved)
	})

	t.Run("returns action unchanged when uri key missing", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		action := &protocol.CodeAction{
			Title: "Fix all token fallback values",
			Data:  map[string]any{"other": "data"},
		}

		resolved, err := resolveFixAllFallbacks(req, action)
		require.NoError(t, err)
		assert.Equal(t, action, resolved)
	})

	t.Run("returns action unchanged when document not found", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		action := &protocol.CodeAction{
			Title: "Fix all token fallback values",
			Data:  map[string]any{"uri": "file:///nonexistent.css"},
		}

		resolved, err := resolveFixAllFallbacks(req, action)
		require.NoError(t, err)
		assert.Equal(t, action, resolved)
	})

	t.Run("resolves edits for all incorrect fallbacks", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		ctx.SetSupportsCodeActionLiterals(true)
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		_ = ctx.TokenManager().Add(&tokens.Token{
			Name:  "color.primary",
			Value: "#ff0000",
			Type:  "color",
		})
		_ = ctx.TokenManager().Add(&tokens.Token{
			Name:  "color.secondary",
			Value: "#00ff00",
			Type:  "color",
		})

		uri := "file:///fixall.css"
		cssContent := `.a { color: var(--color-primary, blue); }
.b { color: var(--color-secondary, red); }`
		_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

		action := &protocol.CodeAction{
			Title: "Fix all token fallback values",
			Data:  map[string]any{"uri": uri},
		}

		resolved, err := resolveFixAllFallbacks(req, action)
		require.NoError(t, err)
		require.NotNil(t, resolved.Edit)
		require.NotNil(t, resolved.Edit.Changes)

		edits := resolved.Edit.Changes[uri]
		// Both var() calls have incorrect fallbacks
		assert.Len(t, edits, 2)

		// Verify the edits contain the correct token values
		editTexts := make([]string, len(edits))
		for i, edit := range edits {
			editTexts[i] = edit.NewText
		}
		assert.Contains(t, editTexts, "var(--color-primary, #ff0000)")
		assert.Contains(t, editTexts, "var(--color-secondary, #00ff00)")
	})

	t.Run("skips var calls with correct fallbacks", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		ctx.SetSupportsCodeActionLiterals(true)
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		_ = ctx.TokenManager().Add(&tokens.Token{
			Name:  "color.correct",
			Value: "#ff0000",
			Type:  "color",
		})

		uri := "file:///correct.css"
		// Fallback matches token value
		cssContent := `.a { color: var(--color-correct, #ff0000); }`
		_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

		action := &protocol.CodeAction{
			Title: "Fix all token fallback values",
			Data:  map[string]any{"uri": uri},
		}

		resolved, err := resolveFixAllFallbacks(req, action)
		require.NoError(t, err)
		require.NotNil(t, resolved.Edit)

		edits := resolved.Edit.Changes[uri]
		// No edits needed since fallback is correct
		assert.Empty(t, edits)
	})

	t.Run("skips unformattable tokens", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		ctx.SetSupportsCodeActionLiterals(true)
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		_ = ctx.TokenManager().Add(&tokens.Token{
			Name:  "border.main",
			Value: "1px solid #000",
			Type:  "border", // composite type, can't be formatted
		})

		uri := "file:///border.css"
		cssContent := `.a { border: var(--border-main, 2px dashed red); }`
		_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

		action := &protocol.CodeAction{
			Title: "Fix all token fallback values",
			Data:  map[string]any{"uri": uri},
		}

		resolved, err := resolveFixAllFallbacks(req, action)
		require.NoError(t, err)
		require.NotNil(t, resolved.Edit)

		edits := resolved.Edit.Changes[uri]
		// No edits for unformattable token types
		assert.Empty(t, edits)
	})
}

func TestCreateToggleFallbackAction_UnformattableToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Token with composite type
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "border.main",
		Value: "1px solid #000",
		Type:  "border",
	})

	varCall := cssparser.VarCall{
		TokenName: "--border-main",
		Fallback:  nil, // no fallback, so toggle tries to add one and formatting fails
		Range: cssparser.Range{
			Start: cssparser.Position{Line: 0, Character: 15},
			End:   cssparser.Position{Line: 0, Character: 35},
		},
	}

	result := createToggleFallbackAction(req, "file:///test.css", varCall)
	assert.Nil(t, result, "Should return nil when token value can't be formatted for toggle")
}

func TestCreateToggleFallbackAction_UnknownToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Don't add any tokens

	varCall := cssparser.VarCall{
		TokenName: "--nonexistent",
		Range: cssparser.Range{
			Start: cssparser.Position{Line: 0, Character: 10},
			End:   cssparser.Position{Line: 0, Character: 30},
		},
	}

	result := createToggleFallbackAction(req, "file:///test.css", varCall)
	assert.Nil(t, result, "Should return nil when token is not found")
}

func TestToggleFallbackAction_UnformattableToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.SetSupportsCodeActionLiterals(true)
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Token with composite type that can't be formatted
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "shadow.main",
		Value: "0 2px 4px rgba(0,0,0,0.2)",
		Type:  "shadow",
	})

	uri := "file:///test.css"
	// No fallback -- toggle would try to add one, but formatting will fail
	cssContent := `.card { box-shadow: var(--shadow-main); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			// Collapsed cursor on the var() call
			Start: protocol.Position{Line: 0, Character: 24},
			End:   protocol.Position{Line: 0, Character: 24},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{},
		},
	})

	require.NoError(t, err)
	// Toggle action should not be present for unformattable token types
	if result != nil {
		actions, ok := result.([]protocol.CodeAction)
		if ok {
			for _, action := range actions {
				assert.NotEqual(t, "Toggle design token fallback value", action.Title,
					"Should not offer toggle when token value can't be formatted")
			}
		}
	}
}

func TestToggleRangeFallbacksAction_UnformattableToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.SetSupportsCodeActionLiterals(true)
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Only add an unformattable token
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "shadow.main",
		Value: "0 2px 4px rgba(0,0,0,0.2)",
		Type:  "shadow",
	})

	uri := "file:///test.css"
	cssContent := `.card { box-shadow: var(--shadow-main); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			// Expanded selection covering the var() call
			Start: protocol.Position{Line: 0, Character: 20},
			End:   protocol.Position{Line: 0, Character: 38},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{},
		},
	})

	require.NoError(t, err)
	// Range toggle should not be present for unformattable tokens
	if result != nil {
		actions, ok := result.([]protocol.CodeAction)
		if ok {
			for _, action := range actions {
				assert.NotEqual(t, "Toggle design token fallback values (in range)", action.Title,
					"Should not offer range toggle when token value can't be formatted")
			}
		}
	}
}

func TestProcessVarCalls_UnknownToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.SetSupportsCodeActionLiterals(true)
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Don't add any tokens -- var() references an unknown token

	uri := "file:///test.css"
	cssContent := `.button { color: var(--unknown-token); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 17},
			End:   protocol.Position{Line: 0, Character: 37},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{},
		},
	})

	require.NoError(t, err)
	// Should return empty actions list (token not found, no actions to offer)
	actions, ok := result.([]protocol.CodeAction)
	require.True(t, ok)
	assert.Empty(t, actions)
}

func TestCodeAction_NoAddFallbackForNonColorDimension(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.SetSupportsCodeActionLiterals(true)
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Token with type "number" -- add-fallback only for color/dimension
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "opacity.default",
		Value: "0.5",
		Type:  "number",
	})

	uri := "file:///test.css"
	cssContent := `.card { opacity: var(--opacity-default); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 17},
			End:   protocol.Position{Line: 0, Character: 39},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{},
		},
	})

	require.NoError(t, err)
	if result != nil {
		actions, ok := result.([]protocol.CodeAction)
		if ok {
			for _, action := range actions {
				assert.NotContains(t, action.Title, "Add fallback value",
					"Should not suggest add-fallback for number type tokens (only color/dimension)")
			}
		}
	}
}

