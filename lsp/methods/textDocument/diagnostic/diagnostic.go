package diagnostic

import (
	"bennypowers.dev/asimonim/lsp/internal/log"
	"fmt"
	"strings"

	"bennypowers.dev/asimonim/lsp/internal/parser"
	"bennypowers.dev/asimonim/lsp/types"
	"go.lsp.dev/protocol"
	lspuri "go.lsp.dev/uri"
)

// DocumentDiagnostic handles the textDocument/diagnostic request (pull diagnostics, LSP 3.17)
func DocumentDiagnostic(req *types.RequestContext, params *protocol.DocumentDiagnosticParams) (protocol.DocumentDiagnosticReport, error) {
	docURI := string(params.TextDocument.URI)
	log.Info("Pull diagnostics requested for: %s", docURI)

	diagnostics, err := GetDiagnostics(req.Server, docURI)
	if err != nil {
		log.Info("Error getting diagnostics: %v", err)
		return nil, err
	}

	return &protocol.RelatedFullDocumentDiagnosticReport{
		FullDocumentDiagnosticReport: protocol.FullDocumentDiagnosticReport{
			Kind:  string(protocol.DocumentDiagnosticReportKindFull),
			Items: diagnostics,
		},
	}, nil
}

// GetDiagnostics returns diagnostics for a document
// Always returns a non-nil array (empty if no diagnostics) to conform to LSP protocol.
// Returning nil would serialize to JSON null which crashes some LSP clients like Neovim.
func GetDiagnostics(ctx types.ServerContext, uri string) ([]protocol.Diagnostic, error) {
	// Get document
	doc := ctx.Document(uri)
	if doc == nil {
		return []protocol.Diagnostic{}, nil
	}

	// Only process CSS-supported files
	if !parser.IsCSSSupportedLanguage(doc.LanguageID()) {
		return []protocol.Diagnostic{}, nil
	}

	// Parse CSS to find var() calls
	result, err := parser.ParseCSSFromDocument(doc.Content(), doc.LanguageID())
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSS: %w", err)
	}
	if result == nil {
		return []protocol.Diagnostic{}, nil
	}

	// Initialize as empty slice, not nil, to ensure proper JSON serialization
	diagnostics := []protocol.Diagnostic{}

	// Check each var() call
	for _, varCall := range result.VarCalls {
		// Look up the token
		token := ctx.Token(varCall.TokenName)
		if token == nil {
			// Unknown tokens are not errors - they're handled by hover
			continue
		}

		// Check for deprecated token
		if token.Deprecated {
			message := fmt.Sprintf("%s is deprecated", varCall.TokenName)
			if token.DeprecationMessage != "" {
				message += ": " + token.DeprecationMessage
			}

			diag := protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      varCall.Range.Start.Line,
						Character: varCall.Range.Start.Character,
					},
					End: protocol.Position{
						Line:      varCall.Range.End.Line,
						Character: varCall.Range.End.Character,
					},
				},
				Severity: protocol.DiagnosticSeverityInformation,
				Message:  protocol.String(message),
				Tags:     protocol.NewDiagnosticTags(protocol.DiagnosticTagDeprecated),
			}

			// Add related information pointing to token definition when supported
			if ctx.SupportsDiagnosticRelatedInfo() && token.DefinitionURI != "" {
				diag.RelatedInformation = []protocol.DiagnosticRelatedInformation{{
					Location: protocol.Location{
						URI: lspuri.URI(token.DefinitionURI),
						Range: protocol.Range{
							Start: protocol.Position{Line: token.Line, Character: token.Character},
							End:   protocol.Position{Line: token.Line, Character: token.Character},
						},
					},
					Message: fmt.Sprintf("Token %s defined here", token.CSSVariableName()),
				}}
			}

			diagnostics = append(diagnostics, diag)
		}

		// Check for incorrect fallback
		if varCall.Fallback != nil {
			fallbackValue := *varCall.Fallback
			tokenValue := token.Value

			// Check semantic equivalence (case-insensitive, whitespace-normalized)
			if !isCSSValueSemanticallyEquivalent(fallbackValue, tokenValue) {
				diagnostics = append(diagnostics, protocol.Diagnostic{
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      varCall.Range.Start.Line,
							Character: varCall.Range.Start.Character,
						},
						End: protocol.Position{
							Line:      varCall.Range.End.Line,
							Character: varCall.Range.End.Character,
						},
					},
					Severity: protocol.DiagnosticSeverityError,
					Message:  protocol.String(fmt.Sprintf("Token fallback does not match expected value: %s", tokenValue)),
				})
			}
		}
	}

	return diagnostics, nil
}

// PublishDiagnostics publishes diagnostics for a document

// isCSSValueSemanticallyEquivalent checks if two CSS values are semantically equivalent
// Ignores whitespace and case differences
func isCSSValueSemanticallyEquivalent(a, b string) bool {
	// Normalize: remove all whitespace and convert to lowercase
	normalize := func(s string) string {
		s = strings.ReplaceAll(s, " ", "")
		s = strings.ReplaceAll(s, "\t", "")
		s = strings.ReplaceAll(s, "\n", "")
		s = strings.ToLower(s)
		return s
	}

	return normalize(a) == normalize(b)
}
