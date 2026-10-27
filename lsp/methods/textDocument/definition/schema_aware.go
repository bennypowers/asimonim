package definition

import (
	"strings"

	"bennypowers.dev/asimonim/lsp/internal/documents"
	"bennypowers.dev/asimonim/lsp/internal/parser/common"
	posutil "bennypowers.dev/asimonim/lsp/internal/position"
	"bennypowers.dev/asimonim/lsp/types"
	"bennypowers.dev/asimonim/schema"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// normalizeLineEndings normalizes line endings to LF for consistent processing
func normalizeLineEndings(content string) string {
	// Replace CRLF with LF, then replace any remaining CR with LF
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return content
}

// findReferenceAtPosition finds a token reference at the given position in a JSON/YAML file
// Returns the token name if found, empty string otherwise
func findReferenceAtPosition(content string, pos protocol.Position) string {
	// Normalize line endings (CRLF -> LF) to handle Windows files correctly
	content = normalizeLineEndings(content)

	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return ""
	}

	line := lines[pos.Line]

	// Check for curly brace references
	if tokenName := findCurlyBraceReferenceAtPosition(line, pos); tokenName != "" {
		return tokenName
	}

	// Check for JSON Pointer references
	if tokenName := findJSONPointerReferenceAtPosition(line, pos); tokenName != "" {
		return tokenName
	}

	return ""
}

// findCurlyBraceReferenceAtPosition finds a curly brace reference at the position
func findCurlyBraceReferenceAtPosition(line string, pos protocol.Position) string {
	matches := common.CurlyBraceReferenceRegexp.FindAllStringSubmatchIndex(line, -1)
	if matches == nil {
		return ""
	}

	for _, match := range matches {
		// match[0], match[1] - full match including braces
		// match[2], match[3] - captured reference without braces

		// Convert byte offsets to UTF-16 positions
		matchStartUTF16 := posutil.ByteOffsetToUTF16Uint32(line, match[0])
		matchEndUTF16 := posutil.ByteOffsetToUTF16Uint32(line, match[1])

		// Check if cursor is within this match
		if pos.Character >= matchStartUTF16 && pos.Character <= matchEndUTF16 {
			// Extract the reference (e.g., "color.primary")
			reference := line[match[2]:match[3]]
			// Convert to token name (e.g., "color-primary")
			return strings.ReplaceAll(reference, ".", "-")
		}
	}

	return ""
}

// findJSONPointerReferenceAtPosition finds a JSON Pointer reference at the position
func findJSONPointerReferenceAtPosition(line string, pos protocol.Position) string {
	matches := common.JSONPointerReferenceRegexp.FindAllStringSubmatchIndex(line, -1)
	if matches == nil {
		return ""
	}

	for _, match := range matches {
		// match[0], match[1] - full match
		// match[2], match[3] - JSON Pointer path (e.g., "#/color/primary")

		// Convert byte offsets to UTF-16 positions
		matchStartUTF16 := posutil.ByteOffsetToUTF16Uint32(line, match[0])
		matchEndUTF16 := posutil.ByteOffsetToUTF16Uint32(line, match[1])

		// Check if cursor is within this match
		if pos.Character >= matchStartUTF16 && pos.Character <= matchEndUTF16 {
			// Extract the JSON Pointer path (e.g., "#/color/primary")
			pointerPath := line[match[2]:match[3]]
			// Convert to token name: remove "#/" prefix and replace "/" with "-"
			// e.g., "#/color/primary" -> "color-primary"
			tokenName := strings.TrimPrefix(pointerPath, "#/")
			return strings.ReplaceAll(tokenName, "/", "-")
		}
	}

	return ""
}

// DefinitionForTokenFile handles go-to-definition for references within token files
func DefinitionForTokenFile(req *types.RequestContext, doc *documents.Document, position protocol.Position) (protocol.DefinitionResult, error) {
	// Find reference at the cursor position. For JSONC, replace comment bytes
	// with spaces (preserving offsets) so references inside comments do not
	// resolve.
	content := doc.Content()
	if doc.LanguageID() == "jsonc" {
		// Mask comment text (preserving line breaks and UTF-16 columns) so
		// references inside comments do not resolve.
		content = string(schema.MaskJSONCComments([]byte(content)))
	}
	tokenName := findReferenceAtPosition(content, position)
	if tokenName == "" {
		return nil, nil
	}

	// Look up the token
	token := req.Server.Token(tokenName)
	if token == nil {
		return nil, nil
	}

	// Return the definition location
	if token.DefinitionURI != "" && len(token.Path) > 0 {
		// token.Character is a 0-based UTF-16 offset, the same unit LSP uses.
		startCharUTF16 := token.Character
		tokenNameLenUTF16 := posutil.StringLengthUTF16Uint32(token.Name)
		endCharUTF16 := startCharUTF16 + tokenNameLenUTF16

		location := protocol.Location{
			URI: uri.URI(token.DefinitionURI),
			Range: protocol.Range{
				Start: protocol.Position{Line: token.Line, Character: startCharUTF16},
				End:   protocol.Position{Line: token.Line, Character: endCharUTF16},
			},
		}

		return protocol.LocationSlice([]protocol.Location{location}), nil
	}

	return nil, nil
}
