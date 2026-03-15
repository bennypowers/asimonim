package lifecycle

import (
	"bennypowers.dev/asimonim/lsp/internal/log"

	"bennypowers.dev/asimonim/lsp/internal/parser/css"
	"bennypowers.dev/asimonim/lsp/types"
)

// Shutdown handles the LSP shutdown request
func Shutdown(req *types.RequestContext) error {
	log.Info("Server shutting down")

	// Clean up the CSS parser pool
	// Note: This is currently handled by server.Close() but we put it here
	// for completeness in case we need other cleanup logic
	css.ClosePool()

	return nil
}
