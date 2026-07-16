package workspace

import (
	"bennypowers.dev/asimonim/lsp/internal/log"
	"bennypowers.dev/asimonim/lsp/methods/textDocument/diagnostic"
	"bennypowers.dev/asimonim/lsp/types"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// WorkspaceDiagnostic handles the workspace/diagnostic request (LSP 3.17).
// Returns diagnostics for all tracked documents in a single batch.
func WorkspaceDiagnostic(req *types.RequestContext, params *protocol.WorkspaceDiagnosticParams) (*protocol.WorkspaceDiagnosticReport, error) {
	log.Info("Workspace diagnostics requested")
	return collectWorkspaceDiagnostics(req.Server, diagnostic.GetDiagnostics)
}

func collectWorkspaceDiagnostics(ctx types.ServerContext, getDiagnostics func(types.ServerContext, string) ([]protocol.Diagnostic, error)) (*protocol.WorkspaceDiagnosticReport, error) {
	docs := ctx.AllDocuments()
	items := make([]protocol.WorkspaceDocumentDiagnosticReport, 0, len(docs))

	for _, doc := range docs {
		diagnostics, err := getDiagnostics(ctx, doc.URI())
		if err != nil {
			log.Info("Warning: failed to get diagnostics for %s: %v", doc.URI(), err)
			continue
		}

		version := int32(doc.Version())
		items = append(items, &protocol.WorkspaceFullDocumentDiagnosticReport{
			FullDocumentDiagnosticReport: protocol.FullDocumentDiagnosticReport{
				Kind:  string(protocol.DocumentDiagnosticReportKindFull),
				Items: diagnostics,
			},
			URI:     uri.URI(doc.URI()),
			Version: &version,
		})
	}

	return &protocol.WorkspaceDiagnosticReport{
		Items: items,
	}, nil
}
