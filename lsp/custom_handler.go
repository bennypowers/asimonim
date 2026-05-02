package lsp

import (
	"encoding/json"

	"github.com/bennypowers/glsp"
	protocol "github.com/bennypowers/glsp/protocol_3_17"
)

// CustomHandler wraps protocol.Handler to intercept initialize for capability detection.
type CustomHandler struct {
	*protocol.Handler
	server *Server
}

// Handle implements glsp.Handler interface
func (h *CustomHandler) Handle(context *glsp.Context) (r any, validMethod, validParams bool, err error) {
	if context.Method == "initialize" {
		supportsPullDiagnostics := DetectPullDiagnosticsSupport(context.Params)
		h.server.SetClientDiagnosticCapability(supportsPullDiagnostics)
	}

	return h.Handler.Handle(context)
}

// DetectPullDiagnosticsSupport parses raw initialize params to detect LSP 3.17 diagnostic capability.
func DetectPullDiagnosticsSupport(params json.RawMessage) bool {
	var raw struct {
		Capabilities struct {
			TextDocument *struct {
				Diagnostic *json.RawMessage `json:"diagnostic"`
			} `json:"textDocument"`
		} `json:"capabilities"`
	}

	if err := json.Unmarshal(params, &raw); err != nil {
		return false
	}

	return raw.Capabilities.TextDocument != nil && raw.Capabilities.TextDocument.Diagnostic != nil
}
