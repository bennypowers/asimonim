package textDocument

import (
	"bennypowers.dev/asimonim/lsp/internal/documents"
	"bennypowers.dev/asimonim/lsp/internal/log"

	"bennypowers.dev/asimonim/lsp/types"
	"go.lsp.dev/protocol"
)

// DidOpen handles the textDocument/didOpen notification
func DidOpen(req *types.RequestContext, params *protocol.DidOpenTextDocumentParams) error {
	log.Info("Document opened: %s (language: %s, version: %d)",
		params.TextDocument.URI, params.TextDocument.LanguageID, int(params.TextDocument.Version))

	uri := string(params.TextDocument.URI)
	languageID := string(params.TextDocument.LanguageID)

	err := req.Server.DocumentManager().DidOpen(uri, languageID,
		int(params.TextDocument.Version), params.TextDocument.Text)
	if err != nil {
		return err
	}

	// Auto-load tokens from files that look like DTCG token files
	// This enables semantic tokens and other features for token files not in config
	content := params.TextDocument.Text
	if (languageID == "json" || languageID == "yaml") &&
		(documents.IsDesignTokensSchema(content) || documents.LooksLikeDTCGContent(content)) {
		if err := req.Server.LoadTokensFromDocumentContent(
			uri,
			languageID,
			content,
		); err != nil {
			log.Warn("Failed to auto-load tokens from %s: %v", uri, err)
		}
	}

	// Publish diagnostics for the opened document (only if using push model)
	// If client supports pull diagnostics (LSP 3.17), it will request them via textDocument/diagnostic
	if !req.Server.UsePullDiagnostics() {
		if ctx := req.Server.GLSPContext(); ctx != nil {
			if err := req.Server.PublishDiagnostics(ctx, uri); err != nil {
				log.Warn("Failed to publish diagnostics for %s: %v", uri, err)
			}
		}
	}

	return nil
}

// DidChange handles the textDocument/didChange notification
func DidChange(req *types.RequestContext, params *protocol.DidChangeTextDocumentParams) error {
	uri := string(params.TextDocument.URI)
	version := int(params.TextDocument.Version)

	log.Info("Document changed: %s (version: %d, changes: %d)", uri, version, len(params.ContentChanges))

	err := req.Server.DocumentManager().DidChange(uri, version, params.ContentChanges)
	if err != nil {
		return err
	}

	if !req.Server.UsePullDiagnostics() {
		if ctx := req.Server.GLSPContext(); ctx != nil {
			if err := req.Server.PublishDiagnostics(ctx, uri); err != nil {
				log.Warn("Failed to publish diagnostics for %s: %v", uri, err)
			}
		}
	}

	return nil
}

// DidClose handles the textDocument/didClose notification
func DidClose(req *types.RequestContext, params *protocol.DidCloseTextDocumentParams) error {
	uri := string(params.TextDocument.URI)

	log.Info("Document closed: %s", uri)

	// Invalidate semantic token cache for this document
	req.Server.SemanticTokenCache().Invalidate(uri)

	return req.Server.DocumentManager().DidClose(uri)
}
