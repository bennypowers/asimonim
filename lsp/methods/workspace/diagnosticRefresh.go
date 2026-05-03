package workspace

import (
	"bennypowers.dev/asimonim/lsp/internal/log"
	"github.com/bennypowers/glsp"
	protocol "github.com/bennypowers/glsp/protocol_3_17"
)

// NotifyDiagnosticRefresh sends workspace/diagnostic/refresh to the client,
// asking it to re-request diagnostics for all documents.
func NotifyDiagnosticRefresh(ctx *glsp.Context) {
	if ctx == nil || ctx.Notify == nil {
		return
	}
	log.Info("Sending workspace/diagnostic/refresh")
	ctx.Notify(protocol.MethodWorkspaceDiagnosticRefresh, nil)
}
