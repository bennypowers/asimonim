package inlayhint

import (
	"fmt"

	"bennypowers.dev/asimonim/lsp/internal/log"
	"bennypowers.dev/asimonim/lsp/internal/parser"
	"bennypowers.dev/asimonim/lsp/internal/parser/css"
	"bennypowers.dev/asimonim/lsp/types"
	"go.lsp.dev/protocol"
)

// InlayHint handles the textDocument/inlayHint request.
// Returns resolved token values as inlay hints next to var() calls.
func InlayHint(req *types.RequestContext, params *protocol.InlayHintParams) ([]protocol.InlayHint, error) {
	if !req.Server.InlayHintsEnabled() {
		return nil, nil
	}

	uri := string(params.TextDocument.URI)
	log.Info("InlayHint requested: %s", uri)

	doc := req.Server.Document(uri)
	if doc == nil {
		return nil, nil
	}

	if !parser.IsCSSSupportedLanguage(doc.LanguageID()) {
		return nil, nil
	}

	result, err := parser.ParseCSSFromDocument(doc.Content(), doc.LanguageID())
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSS: %w", err)
	}
	if result == nil {
		return nil, nil
	}

	var hints []protocol.InlayHint
	for _, varCall := range result.VarCalls {
		if hint, ok := hintForVarCall(req, params.Range, varCall); ok {
			hints = append(hints, hint)
		}
	}

	return hints, nil
}

func hintForVarCall(req *types.RequestContext, requestRange protocol.Range, varCall *css.VarCall) (protocol.InlayHint, bool) {
	var zero protocol.InlayHint

	if varCall.Fallback != nil {
		return zero, false
	}

	if !hintPositionInRange(varCall, requestRange) {
		return zero, false
	}

	token := req.Server.Token(varCall.TokenName)
	if token == nil {
		return zero, false
	}

	displayValue := token.DisplayValue()
	if displayValue == "" {
		return zero, false
	}

	return protocol.InlayHint{
		Position: protocol.Position{
			Line:      varCall.Range.End.Line,
			Character: varCall.Range.End.Character - 1,
		},
		Label: protocol.String(", " + displayValue),
	}, true
}

func hintPositionInRange(varCall *css.VarCall, r protocol.Range) bool {
	line := varCall.Range.End.Line
	char := varCall.Range.End.Character - 1

	if line < r.Start.Line || line > r.End.Line {
		return false
	}
	if line == r.Start.Line && char < r.Start.Character {
		return false
	}
	if line == r.End.Line && char >= r.End.Character {
		return false
	}
	return true
}
