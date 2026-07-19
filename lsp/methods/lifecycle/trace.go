package lifecycle

import (
	"bennypowers.dev/asimonim/lsp/internal/log"

	"bennypowers.dev/asimonim/lsp/types"
	"go.lsp.dev/protocol"
)

// SetTrace handles the $/setTrace notification
func SetTrace(req *types.RequestContext, params *protocol.SetTraceParams) error {
	log.Info("Trace level set to: %s", params.Value)
	return nil
}
