package definition_test

import (
	"testing"

	"bennypowers.dev/asimonim/lsp/internal/documents"
	"bennypowers.dev/asimonim/lsp/internal/tokens"
	"bennypowers.dev/asimonim/lsp/methods/textDocument/definition"
	"bennypowers.dev/asimonim/lsp/testutil"
	"bennypowers.dev/asimonim/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefinition_Draft_CurlyBraceReference(t *testing.T) {
	// Test go-to-definition for curly brace references in draft schema
	content := `{
  "$schema": "https://www.designtokens.org/schemas/draft.json",
  "color": {
    "primary": {
      "$type": "color",
      "$value": "#FF0000"
    },
    "secondary": {
      "$type": "color",
      "$value": "{color.primary}"
    }
  }
}`

	mockServer := testutil.NewMockServer()
	doc := documents.NewDocument("file:///test.json", "json", 1, content)
	mockServer.AddDocument(doc)

	// Add the token with definition location
	mockServer.TokenManager().Add(&tokens.Token{
		Name:          "color-primary",
		Value:         "#FF0000",
		DefinitionURI: "file:///test.json",
		Line:          3,
		Character:     4,
		Path:          []string{"color", "primary"},
	})

	req := &types.RequestContext{
		Server: mockServer,
	}

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.json",
			},
			// Position in the middle of "{color.primary}" on line 9
			Position: protocol.Position{Line: 9, Character: 20},
		},
	}

	result, err := definition.Definition(req, params)
	require.NoError(t, err)

	// Should return location pointing to the definition of color.primary (line 3)
	locations, ok := result.([]protocol.Location)
	require.True(t, ok, "Result should be []protocol.Location")
	assert.NotEmpty(t, locations, "Should find definition location")

	if len(locations) > 0 {
		assert.Equal(t, "file:///test.json", string(locations[0].URI))
		assert.Equal(t, uint32(3), locations[0].Range.Start.Line, "Should point to line where 'primary' is defined")
	}
}

func TestDefinition_2025_JSONPointerReference(t *testing.T) {
	// Test go-to-definition for JSON Pointer references in 2025.10 schema
	content := `{
  "$schema": "https://www.designtokens.org/schemas/2025.10.json",
  "color": {
    "primary": {
      "$type": "color",
      "$value": {
        "colorSpace": "srgb",
        "components": [1.0, 0, 0]
      }
    },
    "secondary": {
      "$type": "color",
      "$ref": "#/color/primary"
    }
  }
}`

	mockServer := testutil.NewMockServer()
	doc := documents.NewDocument("file:///test.json", "json", 1, content)
	mockServer.AddDocument(doc)

	// Add the token with definition location
	mockServer.TokenManager().Add(&tokens.Token{
		Name:          "color-primary",
		Value:         "srgb color",
		DefinitionURI: "file:///test.json",
		Line:          3,
		Character:     4,
		Path:          []string{"color", "primary"},
	})

	req := &types.RequestContext{
		Server: mockServer,
	}

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.json",
			},
			// Position in the JSON Pointer path on line 12
			Position: protocol.Position{Line: 12, Character: 20},
		},
	}

	result, err := definition.Definition(req, params)
	require.NoError(t, err)

	// Should return location pointing to the definition of color/primary (line 3)
	locations, ok := result.([]protocol.Location)
	require.True(t, ok, "Result should be []protocol.Location")
	assert.NotEmpty(t, locations, "Should find definition location for JSON Pointer")

	if len(locations) > 0 {
		assert.Equal(t, "file:///test.json", string(locations[0].URI))
		assert.Equal(t, uint32(3), locations[0].Range.Start.Line, "Should point to line where 'primary' is defined")
	}
}

func TestDefinition_TokenFile_UnknownToken(t *testing.T) {
	// Test that definition returns nil when the referenced token doesn't exist
	content := `{
  "color": {
    "secondary": {
      "$value": "{color.nonexistent}"
    }
  }
}`

	mockServer := testutil.NewMockServer()
	doc := documents.NewDocument("file:///test.json", "json", 1, content)
	mockServer.AddDocument(doc)
	// Don't add any tokens -- the reference should be unresolvable

	req := &types.RequestContext{
		Server: mockServer,
	}

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.json",
			},
			Position: protocol.Position{Line: 3, Character: 20},
		},
	}

	result, err := definition.Definition(req, params)
	require.NoError(t, err)
	assert.Nil(t, result, "Unknown token reference should return nil")
}

func TestDefinition_TokenFile_TokenWithoutDefinitionURI(t *testing.T) {
	// Test that definition returns nil when the token has no DefinitionURI
	content := `{
  "color": {
    "secondary": {
      "$value": "{color.primary}"
    }
  }
}`

	mockServer := testutil.NewMockServer()
	doc := documents.NewDocument("file:///test.json", "json", 1, content)
	mockServer.AddDocument(doc)

	mockServer.TokenManager().Add(&tokens.Token{
		Name:  "color-primary",
		Value: "#FF0000",
		// No DefinitionURI or Path
	})

	req := &types.RequestContext{
		Server: mockServer,
	}

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.json",
			},
			Position: protocol.Position{Line: 3, Character: 20},
		},
	}

	result, err := definition.Definition(req, params)
	require.NoError(t, err)
	assert.Nil(t, result, "Token without DefinitionURI should return nil")
}

func TestDefinition_TokenFile_CursorBeyondLastLine(t *testing.T) {
	// Test position beyond the last line of the document
	content := `{"$value": "{color.primary}"}`

	mockServer := testutil.NewMockServer()
	doc := documents.NewDocument("file:///test.json", "json", 1, content)
	mockServer.AddDocument(doc)

	req := &types.RequestContext{
		Server: mockServer,
	}

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.json",
			},
			// Line 99 is way beyond the document
			Position: protocol.Position{Line: 99, Character: 0},
		},
	}

	result, err := definition.Definition(req, params)
	require.NoError(t, err)
	assert.Nil(t, result, "Position beyond last line should return nil")
}

func TestDefinition_TokenFile_CRLFLineEndings(t *testing.T) {
	// Test that CRLF line endings are handled correctly
	content := "{\r\n  \"color\": {\r\n    \"secondary\": {\r\n      \"$value\": \"{color.primary}\"\r\n    }\r\n  }\r\n}"

	mockServer := testutil.NewMockServer()
	doc := documents.NewDocument("file:///test.json", "json", 1, content)
	mockServer.AddDocument(doc)

	mockServer.TokenManager().Add(&tokens.Token{
		Name:          "color-primary",
		Value:         "#FF0000",
		DefinitionURI: "file:///test.json",
		Line:          1,
		Character:     0,
		Path:          []string{"color", "primary"},
	})

	req := &types.RequestContext{
		Server: mockServer,
	}

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.json",
			},
			// Position in the middle of "{color.primary}" on line 3 (after CRLF normalization)
			Position: protocol.Position{Line: 3, Character: 20},
		},
	}

	result, err := definition.Definition(req, params)
	require.NoError(t, err)

	// Should find the definition despite CRLF line endings
	locations, ok := result.([]protocol.Location)
	require.True(t, ok, "Result should be []protocol.Location")
	assert.NotEmpty(t, locations, "Should find definition with CRLF line endings")
}

func TestDefinition_TokenFile_CursorOnNonReferenceLine(t *testing.T) {
	// Test cursor on a line that has no token references
	content := `{
  "color": {
    "primary": {
      "$type": "color",
      "$value": "#FF0000"
    },
    "secondary": {
      "$value": "{color.primary}"
    }
  }
}`

	mockServer := testutil.NewMockServer()
	doc := documents.NewDocument("file:///test.json", "json", 1, content)
	mockServer.AddDocument(doc)

	mockServer.TokenManager().Add(&tokens.Token{
		Name:          "color-primary",
		Value:         "#FF0000",
		DefinitionURI: "file:///test.json",
		Line:          2,
		Character:     0,
		Path:          []string{"color", "primary"},
	})

	req := &types.RequestContext{
		Server: mockServer,
	}

	// Cursor on "$value": "#FF0000" line - a literal value, not a reference
	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.json",
			},
			Position: protocol.Position{Line: 4, Character: 20},
		},
	}

	result, err := definition.Definition(req, params)
	require.NoError(t, err)
	assert.Nil(t, result, "Cursor on literal value should not trigger definition")
}

func TestDefinition_TokenFile_GetLineTextFromDocument(t *testing.T) {
	// Test the getLineText path via DefinitionForTokenFile when the token
	// definition is in an open document (document manager path)
	content := `{
  "color": {
    "primary": {
      "$type": "color",
      "$value": "#FF0000"
    },
    "secondary": {
      "$value": "{color.primary}"
    }
  }
}`

	mockServer := testutil.NewMockServer()
	doc := documents.NewDocument("file:///tokens.json", "json", 1, content)
	mockServer.AddDocument(doc)

	// Token definition is also in this open document
	mockServer.TokenManager().Add(&tokens.Token{
		Name:          "color-primary",
		Value:         "#FF0000",
		DefinitionURI: "file:///tokens.json",
		Line:          2,
		Character:     4,
		Path:          []string{"color", "primary"},
	})

	req := &types.RequestContext{
		Server: mockServer,
	}

	// Hover over {color.primary} on line 7
	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///tokens.json",
			},
			Position: protocol.Position{Line: 7, Character: 20},
		},
	}

	result, err := definition.Definition(req, params)
	require.NoError(t, err)

	locations, ok := result.([]protocol.Location)
	require.True(t, ok, "Result should be []protocol.Location")
	require.Len(t, locations, 1)

	// getLineText should have retrieved the line from the document manager
	// and converted byte offset to UTF-16
	assert.Equal(t, "file:///tokens.json", locations[0].URI)
	assert.Equal(t, uint32(2), locations[0].Range.Start.Line)
}

func TestDefinition_TokenFile_GetLineTextOutOfBounds(t *testing.T) {
	// Test when token.Line points beyond the document length
	content := `{
  "color": {
    "secondary": {
      "$value": "{color.primary}"
    }
  }
}`

	mockServer := testutil.NewMockServer()
	doc := documents.NewDocument("file:///tokens.json", "json", 1, content)
	mockServer.AddDocument(doc)

	// Token definition URI points to an open document but line is beyond bounds
	mockServer.TokenManager().Add(&tokens.Token{
		Name:          "color-primary",
		Value:         "#FF0000",
		DefinitionURI: "file:///tokens.json",
		Line:          999, // beyond end of document
		Character:     0,
		Path:          []string{"color", "primary"},
	})

	req := &types.RequestContext{
		Server: mockServer,
	}

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///tokens.json",
			},
			Position: protocol.Position{Line: 3, Character: 20},
		},
	}

	result, err := definition.Definition(req, params)
	require.NoError(t, err)

	// Should fall back to zero-width range when line text is empty
	locations, ok := result.([]protocol.Location)
	require.True(t, ok)
	require.Len(t, locations, 1)
	assert.Equal(t, uint32(999), locations[0].Range.Start.Line)
	assert.Equal(t, uint32(0), locations[0].Range.Start.Character, "Should fallback to 0 when line is out of bounds")
}

func TestDefinition_TokenFile_GetLineTextFromDisk(t *testing.T) {
	// Test the getLineText disk fallback path by having the token definition
	// in a URI that's NOT in the document manager (not opened)
	content := `{
  "color": {
    "secondary": {
      "$value": "{color.primary}"
    }
  }
}`

	mockServer := testutil.NewMockServer()
	doc := documents.NewDocument("file:///source.json", "json", 1, content)
	mockServer.AddDocument(doc)

	// Token definition is in a different file that's NOT opened in the editor
	// The file doesn't exist on disk, so getLineText will error and fall back
	mockServer.TokenManager().Add(&tokens.Token{
		Name:          "color-primary",
		Value:         "#FF0000",
		DefinitionURI: "file:///nonexistent-definition-file.json",
		Line:          5,
		Character:     4,
		Path:          []string{"color", "primary"},
	})

	req := &types.RequestContext{
		Server: mockServer,
	}

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///source.json",
			},
			Position: protocol.Position{Line: 3, Character: 20},
		},
	}

	result, err := definition.Definition(req, params)
	require.NoError(t, err)

	// Should fall back to zero-width range when file can't be read from disk
	locations, ok := result.([]protocol.Location)
	require.True(t, ok)
	require.Len(t, locations, 1)
	assert.Equal(t, "file:///nonexistent-definition-file.json", locations[0].URI)
	assert.Equal(t, uint32(5), locations[0].Range.Start.Line)
	assert.Equal(t, uint32(0), locations[0].Range.Start.Character, "Should fallback to 0 when file is unreadable")
}

func TestDefinition_NoReferenceCursor(t *testing.T) {
	// Test that definition returns nil when cursor is not on a reference
	content := `{
  "$schema": "https://www.designtokens.org/schemas/2025.10.json",
  "color": {
    "primary": {
      "$type": "color",
      "$value": {
        "colorSpace": "srgb",
        "components": [1.0, 0, 0]
      }
    }
  }
}`

	mockServer := testutil.NewMockServer()
	doc := documents.NewDocument("file:///test.json", "json", 1, content)
	mockServer.AddDocument(doc)

	req := &types.RequestContext{
		Server: mockServer,
	}

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.json",
			},
			// Position on "$type" keyword (not a reference)
			Position: protocol.Position{Line: 4, Character: 10},
		},
	}

	result, err := definition.Definition(req, params)
	require.NoError(t, err)

	// Should return nil when not on a reference
	assert.Nil(t, result)
}
