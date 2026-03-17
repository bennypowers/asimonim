package references

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

// TestReferences_CSSFile_ReturnsTokenDefinition tests that references from CSS returns the token definition
func TestReferences_CSSFile_ReturnsTokenDefinition(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add token with definition location
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:          "color-primary",
		Path:          []string{"color", "primary"},
		DefinitionURI: "file:///tokens.json",
		Line:          2,
		Character:     4,
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 24},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: false},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result, 1)
	assert.Equal(t, "file:///tokens.json", string(result[0].URI))
	assert.Equal(t, uint32(2), result[0].Range.Start.Line)
	assert.Equal(t, uint32(4), result[0].Range.Start.Character)
}

// TestReferences_CSSFile_UnknownToken tests that references returns nil when token is not found
func TestReferences_CSSFile_UnknownToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///test.css"
	cssContent := `.button { color: var(--unknown-token); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 24},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: false},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestReferences_CSSFile_OutsideVarCall tests that references returns nil when cursor is not on var()
func TestReferences_CSSFile_OutsideVarCall(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a token
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:          "color-primary",
		Path:          []string{"color", "primary"},
		DefinitionURI: "file:///tokens.json",
		Line:          2,
		Character:     4,
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	// Cursor at position 0 (on the dot of .button)
	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 0},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: false},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestReferences_CSSFile_TokenWithoutDefinitionURI tests that references returns nil when token has no DefinitionURI
func TestReferences_CSSFile_TokenWithoutDefinitionURI(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add token without definition location
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name: "color-primary",
		Path: []string{"color", "primary"},
		// No DefinitionURI set
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 24},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: false},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestReferences_CSSFile_PositionOnDifferentLine tests cursor on a different line than the var() call
func TestReferences_CSSFile_PositionOnDifferentLine(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a token
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:          "color-primary",
		Path:          []string{"color", "primary"},
		DefinitionURI: "file:///tokens.json",
		Line:          2,
		Character:     4,
	})

	uri := "file:///test.css"
	// Multi-line CSS - var() is on line 1
	cssContent := `.button {
  color: var(--color-primary);
}`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	// Cursor on line 0, which is before the var() call on line 1
	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 5},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: false},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestReferences_CSSFile_PositionPastVarCall tests cursor position after the var() call ends
func TestReferences_CSSFile_PositionPastVarCall(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a token
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:          "color-primary",
		Path:          []string{"color", "primary"},
		DefinitionURI: "file:///tokens.json",
		Line:          2,
		Character:     4,
	})

	uri := "file:///test.css"
	// .button { color: var(--color-primary); }
	//                                      ^^ position 36 is ')' closing paren, position 37 is ';'
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	// Cursor at position 37 (on the semicolon, after the var() call ends)
	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 37},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: false},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestReferences_JSONFile_FindsReferencesInCSS tests finding CSS var() references from JSON token file
func TestReferences_JSONFile_FindsReferencesInCSS(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a token with extension data
	token := &tokens.Token{
		Name:          "color-primary",
		Value:         "#ff0000",
		Type:          "color",
		Path:          []string{"color", "primary"},
		Reference:     "{color.primary}",
		DefinitionURI: "file:///tokens.json",
	}
	_ = ctx.TokenManager().Add(token)

	// Open JSON token file
	jsonURI := "file:///tokens.json"
	jsonContent := `{
  "color": {
    "primary": {
      "$type": "color",
      "$value": "#ff0000"
    }
  }
}`
	_ = ctx.DocumentManager().DidOpen(jsonURI, "json", 1, jsonContent)

	// Open CSS files with var() calls
	cssURI1 := "file:///styles1.css"
	cssContent1 := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(cssURI1, "css", 1, cssContent1)

	cssURI2 := "file:///styles2.css"
	cssContent2 := `.link { background: var(--color-primary, red); }`
	_ = ctx.DocumentManager().DidOpen(cssURI2, "css", 1, cssContent2)

	// Request references from the JSON token file (cursor on token)
	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: jsonURI},
			Position:     protocol.Position{Line: 2, Character: 6}, // On "primary" key
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should find references in both CSS files
	assert.GreaterOrEqual(t, len(result), 2)

	foundInCSS1 := false
	foundInCSS2 := false
	for _, loc := range result {
		if loc.URI == cssURI1 {
			foundInCSS1 = true
		}
		if loc.URI == cssURI2 {
			foundInCSS2 = true
		}
	}
	assert.True(t, foundInCSS1, "Should find var() reference in styles1.css")
	assert.True(t, foundInCSS2, "Should find var() reference in styles2.css")
}

// TestReferences_JSONFile_FindsReferencesInJSON tests finding token references in other JSON files
func TestReferences_JSONFile_FindsReferencesInJSON(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add tokens
	primaryToken := &tokens.Token{
		Name:          "color-primary",
		Value:         "#ff0000",
		Type:          "color",
		Path:          []string{"color", "primary"},
		Reference:     "{color.primary}",
		DefinitionURI: "file:///tokens.json",
	}
	_ = ctx.TokenManager().Add(primaryToken)

	// Open JSON token file with token definition
	jsonURI := "file:///tokens.json"
	jsonContent := `{
  "color": {
    "primary": {
      "$type": "color",
      "$value": "#ff0000"
    },
    "brand": {
      "$type": "color",
      "$value": "{color.primary}"
    }
  }
}`
	_ = ctx.DocumentManager().DidOpen(jsonURI, "json", 1, jsonContent)

	// Request references from the JSON file
	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: jsonURI},
			Position:     protocol.Position{Line: 2, Character: 6}, // On "primary"
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should find reference in the same JSON file where brand references primary
	foundReference := false
	for _, loc := range result {
		if loc.URI == jsonURI && loc.Range.Start.Line == 8 {
			foundReference = true
		}
	}
	assert.True(t, foundReference, "Should find {color.primary} reference in brand token")
}

// TestReferences_WithIncludeDeclaration tests including the token definition
func TestReferences_WithIncludeDeclaration(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	token := &tokens.Token{
		Name:          "color-primary",
		Value:         "#ff0000",
		Type:          "color",
		Path:          []string{"color", "primary"},
		Reference:     "{color.primary}",
		DefinitionURI: "file:///tokens.json",
	}
	_ = ctx.TokenManager().Add(token)

	jsonURI := "file:///tokens.json"
	jsonContent := `{
  "color": {
    "primary": {
      "$type": "color",
      "$value": "#ff0000"
    }
  }
}`
	_ = ctx.DocumentManager().DidOpen(jsonURI, "json", 1, jsonContent)

	cssURI := "file:///styles.css"
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(cssURI, "css", 1, cssContent)

	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: jsonURI},
			Position:     protocol.Position{Line: 2, Character: 6},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: true,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should include declaration
	foundDeclaration := false
	for _, loc := range result {
		if loc.URI == jsonURI && loc.Range.Start.Line == 2 {
			foundDeclaration = true
		}
	}
	assert.True(t, foundDeclaration, "Should include declaration when IncludeDeclaration is true")
}

// TestReferences_UnknownToken tests when cursor is not on a token
func TestReferences_UnknownToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	jsonURI := "file:///tokens.json"
	jsonContent := `{
  "color": {
    "primary": {
      "$type": "color",
      "$value": "#ff0000"
    }
  }
}`
	_ = ctx.DocumentManager().DidOpen(jsonURI, "json", 1, jsonContent)

	// Position not on a token
	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: jsonURI},
			Position:     protocol.Position{Line: 0, Character: 0}, // On opening brace
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestReferences_DocumentNotFound tests when document doesn't exist
func TestReferences_DocumentNotFound(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.json"},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestReferences_YAMLFile tests references from YAML token files
func TestReferences_YAMLFile(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	token := &tokens.Token{
		Name:          "color-primary",
		Value:         "#ff0000",
		Type:          "color",
		Path:          []string{"color", "primary"},
		Reference:     "{color.primary}",
		DefinitionURI: "file:///tokens.yaml",
	}
	_ = ctx.TokenManager().Add(token)

	yamlURI := "file:///tokens.yaml"
	yamlContent := `color:
  primary:
    $type: color
    $value: "#ff0000"
`
	_ = ctx.DocumentManager().DidOpen(yamlURI, "yaml", 1, yamlContent)

	cssURI := "file:///styles.css"
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(cssURI, "css", 1, cssContent)

	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: yamlURI},
			Position:     protocol.Position{Line: 1, Character: 3}, // On "primary"
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should find var() reference in CSS
	foundInCSS := false
	for _, loc := range result {
		if loc.URI == cssURI {
			foundInCSS = true
		}
	}
	assert.True(t, foundInCSS, "Should find var() reference in CSS from YAML token file")
}

// TestGetLine tests the getLine helper function
func TestGetLine(t *testing.T) {
	content := "line0\nline1\nline2"

	t.Run("valid line", func(t *testing.T) {
		assert.Equal(t, "line0", getLine(content, 0))
		assert.Equal(t, "line1", getLine(content, 1))
		assert.Equal(t, "line2", getLine(content, 2))
	})

	t.Run("negative line number returns empty string", func(t *testing.T) {
		assert.Equal(t, "", getLine(content, -1))
	})

	t.Run("line number out of bounds returns empty string", func(t *testing.T) {
		assert.Equal(t, "", getLine(content, 3))
		assert.Equal(t, "", getLine(content, 100))
	})

	t.Run("empty content", func(t *testing.T) {
		// empty string splits into one empty element
		assert.Equal(t, "", getLine("", 0))
		assert.Equal(t, "", getLine("", 1))
	})
}

// TestGetCharAt tests the getCharAt helper function
func TestGetCharAt(t *testing.T) {
	content := "abc\ndef"

	t.Run("valid position", func(t *testing.T) {
		// 'a' at line 0, char 0
		assert.Equal(t, 'a', getCharAt(content, protocol.Position{Line: 0, Character: 0}))
		// 'd' at line 1, char 0
		assert.Equal(t, 'd', getCharAt(content, protocol.Position{Line: 1, Character: 0}))
	})

	t.Run("line out of bounds returns null rune", func(t *testing.T) {
		assert.Equal(t, rune(0), getCharAt(content, protocol.Position{Line: 5, Character: 0}))
	})

	t.Run("character out of bounds returns null rune", func(t *testing.T) {
		assert.Equal(t, rune(0), getCharAt(content, protocol.Position{Line: 0, Character: 100}))
	})
}

// TestIsValidCSSReference tests the isValidCSSReference helper
func TestIsValidCSSReference(t *testing.T) {
	t.Run("valid when followed by closing paren", func(t *testing.T) {
		// var(--color-primary)
		content := "var(--color-primary)"
		// endPos points to the character after "--color-primary" which is ')'
		valid := isValidCSSReference(content, protocol.Position{Line: 0, Character: 19})
		assert.True(t, valid)
	})

	t.Run("valid when followed by comma", func(t *testing.T) {
		// var(--color-primary, red)
		content := "var(--color-primary, red)"
		// endPos at the comma after "--color-primary"
		valid := isValidCSSReference(content, protocol.Position{Line: 0, Character: 19})
		assert.True(t, valid)
	})

	t.Run("invalid when followed by other character", func(t *testing.T) {
		// --color-primary-dark (suffix extends the name)
		content := "--color-primary-dark: red;"
		valid := isValidCSSReference(content, protocol.Position{Line: 0, Character: 15})
		assert.False(t, valid)
	})

	t.Run("invalid when character at end of line", func(t *testing.T) {
		content := "--color-primary"
		// endPos.Character == len(line), so no char after
		valid := isValidCSSReference(content, protocol.Position{Line: 0, Character: 15})
		assert.False(t, valid)
	})

	t.Run("invalid on non-existent line", func(t *testing.T) {
		content := "line0"
		valid := isValidCSSReference(content, protocol.Position{Line: 5, Character: 0})
		assert.False(t, valid)
	})
}

// TestFindSubstringRanges tests the findSubstringRanges helper
func TestFindSubstringRanges(t *testing.T) {
	t.Run("single occurrence", func(t *testing.T) {
		content := "hello world"
		ranges := findSubstringRanges(content, "world")
		require.Len(t, ranges, 1)
		// "world" starts at char 6, ends at char 11
		assert.Equal(t, uint32(0), ranges[0].Start.Line)
		assert.Equal(t, uint32(6), ranges[0].Start.Character)
		assert.Equal(t, uint32(0), ranges[0].End.Line)
		assert.Equal(t, uint32(11), ranges[0].End.Character)
	})

	t.Run("multiple occurrences on same line", func(t *testing.T) {
		content := "foo bar foo baz foo"
		ranges := findSubstringRanges(content, "foo")
		require.Len(t, ranges, 3)
		assert.Equal(t, uint32(0), ranges[0].Start.Character)
		assert.Equal(t, uint32(8), ranges[1].Start.Character)
		assert.Equal(t, uint32(16), ranges[2].Start.Character)
	})

	t.Run("occurrences on multiple lines", func(t *testing.T) {
		content := "foo\nbar\nfoo"
		ranges := findSubstringRanges(content, "foo")
		require.Len(t, ranges, 2)
		assert.Equal(t, uint32(0), ranges[0].Start.Line)
		assert.Equal(t, uint32(2), ranges[1].Start.Line)
	})

	t.Run("no occurrences", func(t *testing.T) {
		content := "hello world"
		ranges := findSubstringRanges(content, "xyz")
		assert.Empty(t, ranges)
	})

	t.Run("empty content", func(t *testing.T) {
		ranges := findSubstringRanges("", "foo")
		assert.Empty(t, ranges)
	})
}

// TestFindCSSReferences_SkipsNonCSSDocuments tests that findCSSReferences skips non-CSS documents
func TestFindCSSReferences_SkipsNonCSSDocuments(t *testing.T) {
	ctx := testutil.NewMockServerContext()

	// Open a JSON document (not CSS)
	jsonURI := "file:///tokens.json"
	_ = ctx.DocumentManager().DidOpen(jsonURI, "json", 1, `{"color": {"$value": "--color-primary"}}`)

	// Open a CSS document with a valid reference
	cssURI := "file:///styles.css"
	_ = ctx.DocumentManager().DidOpen(cssURI, "css", 1, `.btn { color: var(--color-primary); }`)

	locationMap := make(map[string]protocol.Location)
	findCSSReferences(ctx.AllDocuments(), "--color-primary", locationMap)

	// Should only find the CSS reference, not the JSON one
	for _, loc := range locationMap {
		assert.Equal(t, protocol.DocumentUri(cssURI), loc.URI, "Should only find references in CSS documents")
	}
	assert.Len(t, locationMap, 1)
}

// TestFindCSSReferences_ExcludesInvalidReferences tests that partial name matches are excluded
func TestFindCSSReferences_ExcludesInvalidReferences(t *testing.T) {
	ctx := testutil.NewMockServerContext()

	// CSS with both valid and invalid references:
	// --color-primary) is valid (followed by ')')
	// --color-primary-dark is invalid (followed by '-')
	cssURI := "file:///styles.css"
	cssContent := `.a { color: var(--color-primary); }
.b { --color-primary-dark: blue; }`
	_ = ctx.DocumentManager().DidOpen(cssURI, "css", 1, cssContent)

	locationMap := make(map[string]protocol.Location)
	findCSSReferences(ctx.AllDocuments(), "--color-primary", locationMap)

	// Only the var() call should be a valid reference (followed by ')')
	assert.Len(t, locationMap, 1)
}

// TestFindJSONReferences_SkipsCSSDocuments tests that findJSONReferences skips CSS documents
func TestFindJSONReferences_SkipsCSSDocuments(t *testing.T) {
	ctx := testutil.NewMockServerContext()

	// Open a CSS document containing the reference text
	cssURI := "file:///styles.css"
	_ = ctx.DocumentManager().DidOpen(cssURI, "css", 1, `/* {color.primary} */`)

	// Open a JSON document containing the reference
	jsonURI := "file:///tokens.json"
	_ = ctx.DocumentManager().DidOpen(jsonURI, "json", 1, `{"brand": {"$value": "{color.primary}"}}`)

	locationMap := make(map[string]protocol.Location)
	findJSONReferences(ctx.AllDocuments(), "{color.primary}", locationMap)

	// Should only find the JSON reference, not the CSS one
	for _, loc := range locationMap {
		assert.Equal(t, protocol.DocumentUri(jsonURI), loc.URI, "Should only find references in JSON documents")
	}
	assert.Len(t, locationMap, 1)
}

// TestFindJSONReferences_EmptyReference tests that empty reference returns early
func TestFindJSONReferences_EmptyReference(t *testing.T) {
	ctx := testutil.NewMockServerContext()

	_ = ctx.DocumentManager().DidOpen("file:///tokens.json", "json", 1, `{"foo": "bar"}`)

	locationMap := make(map[string]protocol.Location)
	findJSONReferences(ctx.AllDocuments(), "", locationMap)

	assert.Empty(t, locationMap)
}

// TestFindTokenDefinitionRange_EmptyPath tests that empty path returns zero range
func TestFindTokenDefinitionRange_EmptyPath(t *testing.T) {
	content := `{"color": {"primary": {"$value": "#ff0000"}}}`
	result := findTokenDefinitionRange(content, []string{}, "json")
	assert.Equal(t, uint32(0), result.Start.Line)
	assert.Equal(t, uint32(0), result.Start.Character)
	assert.Equal(t, uint32(0), result.End.Line)
	assert.Equal(t, uint32(0), result.End.Character)
}

// TestFindTokenDefinitionRange_KeyNotFound tests when key is not in content
func TestFindTokenDefinitionRange_KeyNotFound(t *testing.T) {
	content := `{"color": {"primary": {"$value": "#ff0000"}}}`
	result := findTokenDefinitionRange(content, []string{"nonexistent"}, "json")
	// Should return zero range when key is not found
	assert.Equal(t, uint32(0), result.Start.Line)
	assert.Equal(t, uint32(0), result.Start.Character)
}

// TestReferences_NonTokenFile tests that references returns nil for non-token files
func TestReferences_NonTokenFile(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.ShouldProcessAsTokenFileFunc = func(uri string) bool { return false }
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	jsonURI := "file:///package.json"
	_ = ctx.DocumentManager().DidOpen(jsonURI, "json", 1, `{"name": "my-package"}`)

	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: jsonURI},
			Position:     protocol.Position{Line: 0, Character: 3},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: false},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestReferences_WithIncludeDeclaration_NoDefinitionURI tests that declaration is not added when token has no DefinitionURI
func TestReferences_WithIncludeDeclaration_NoDefinitionURI(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	token := &tokens.Token{
		Name:      "color-primary",
		Value:     "#ff0000",
		Type:      "color",
		Path:      []string{"color", "primary"},
		Reference: "{color.primary}",
		// No DefinitionURI
	}
	_ = ctx.TokenManager().Add(token)

	jsonURI := "file:///tokens.json"
	jsonContent := `{
  "color": {
    "primary": {
      "$type": "color",
      "$value": "#ff0000"
    }
  }
}`
	_ = ctx.DocumentManager().DidOpen(jsonURI, "json", 1, jsonContent)

	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: jsonURI},
			Position:     protocol.Position{Line: 2, Character: 6},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: true,
		},
	})

	require.NoError(t, err)
	// Should not include declaration since there's no DefinitionURI
	for _, loc := range result {
		// None of the locations should be from the token definition
		if loc.URI == jsonURI && loc.Range.Start.Line == 2 {
			t.Error("Should not include declaration when DefinitionURI is empty")
		}
	}
}

// TestReferences_WithIncludeDeclaration_DefinitionDocNotFound tests when definition document is not open
func TestReferences_WithIncludeDeclaration_DefinitionDocNotFound(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	token := &tokens.Token{
		Name:          "color-primary",
		Value:         "#ff0000",
		Type:          "color",
		Path:          []string{"color", "primary"},
		Reference:     "{color.primary}",
		DefinitionURI: "file:///other-tokens.json", // this doc is not open
	}
	_ = ctx.TokenManager().Add(token)

	jsonURI := "file:///tokens.json"
	jsonContent := `{
  "color": {
    "primary": {
      "$type": "color",
      "$value": "#ff0000"
    }
  }
}`
	_ = ctx.DocumentManager().DidOpen(jsonURI, "json", 1, jsonContent)

	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: jsonURI},
			Position:     protocol.Position{Line: 2, Character: 6},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: true,
		},
	})

	require.NoError(t, err)
	// Should not crash, and should not include declaration from non-open doc
	for _, loc := range result {
		assert.NotEqual(t, protocol.DocumentUri("file:///other-tokens.json"), loc.URI,
			"Should not include declaration when definition document is not open")
	}
}

// TestFindTokenAtPosition_InvalidJSON tests findTokenAtPosition with invalid JSON
func TestFindTokenAtPosition_InvalidJSON(t *testing.T) {
	// Invalid JSON should return empty string
	result := findTokenAtPosition("not valid json {{{", protocol.Position{Line: 0, Character: 5}, "json")
	assert.Equal(t, "", result)
}

// TestFindTokenAtPosition_EmptyDocument tests findTokenAtPosition with empty document
func TestFindTokenAtPosition_EmptyDocument(t *testing.T) {
	result := findTokenAtPosition("{}", protocol.Position{Line: 0, Character: 0}, "json")
	assert.Equal(t, "", result)
}

// TestFindTokenAtPosition_JSONC tests findTokenAtPosition with JSONC (comments)
func TestFindTokenAtPosition_JSONC(t *testing.T) {
	content := `{
  // comment
  "color": {
    "$value": "#ff0000"
  }
}`
	// cursor on "color" key at line 2, char 3
	result := findTokenAtPosition(content, protocol.Position{Line: 2, Character: 3}, "jsonc")
	assert.Equal(t, "color", result)
}
