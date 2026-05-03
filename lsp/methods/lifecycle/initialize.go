package lifecycle

import (
	"bennypowers.dev/asimonim/lsp/internal/log"

	"bennypowers.dev/asimonim/lsp/internal/uriutil"
	"bennypowers.dev/asimonim/lsp/types"
	protocol "github.com/bennypowers/glsp/protocol_3_17"
)

// Initialize handles the LSP initialize request
func Initialize(req *types.RequestContext, params *protocol.InitializeParams) (any, error) {
	clientName := "unknown"
	if params.ClientInfo != nil {
		clientName = params.ClientInfo.Name
	}

	log.Info("Initializing for client: %s", clientName)

	// Store client capabilities for later use by handlers
	req.Server.SetClientCapabilities(params.Capabilities)

	// CustomHandler detects pull diagnostics support from raw JSON before this handler runs,
	// because capability detection must happen before the protocol handler sets initialized state.
	supportsPullDiagnostics := false
	if detectedCapability := req.Server.ClientDiagnosticCapability(); detectedCapability != nil {
		// Use the detected capability from raw JSON parsing
		supportsPullDiagnostics = *detectedCapability
	}
	// else: capability not detected (nil) - default to push diagnostics (false)

	req.Server.SetUsePullDiagnostics(supportsPullDiagnostics)

	if supportsPullDiagnostics {
		log.Info("Using pull diagnostics model (LSP 3.17) - client will request diagnostics")
	} else {
		log.Info("Using push diagnostics model (LSP 3.0) - server will push diagnostics")
	}

	// Store the workspace root
	if params.RootURI != nil {
		req.Server.SetRootURI(*params.RootURI)
		// Convert URI to file path
		req.Server.SetRootPath(uriutil.URIToPath(*params.RootURI))
		log.Info("Workspace root: %s", req.Server.RootPath())
	} else if params.RootPath != nil {
		req.Server.SetRootPath(*params.RootPath)
		req.Server.SetRootURI(uriutil.PathToURI(*params.RootPath))
		log.Info("Workspace root (from rootPath): %s", req.Server.RootPath())
	}

	syncKind := protocol.TextDocumentSyncKindIncremental
	capabilities := protocol.ServerCapabilities{}
	capabilities.TextDocumentSync = protocol.TextDocumentSyncOptions{
		OpenClose: boolPtr(true),
		Change:    &syncKind,
	}
	capabilities.HoverProvider = true
	capabilities.CompletionProvider = &protocol.CompletionOptions{
		TriggerCharacters: []string{"-"},
		ResolveProvider:   boolPtr(true),
	}
	capabilities.DefinitionProvider = true
	capabilities.ReferencesProvider = true
	capabilities.CodeActionProvider = protocol.CodeActionOptions{
		ResolveProvider: boolPtr(true),
	}
	capabilities.ColorProvider = true
	capabilities.InlayHintProvider = true
	capabilities.SemanticTokensProvider = protocol.SemanticTokensOptions{
		Legend: protocol.SemanticTokensLegend{
			TokenTypes:     []string{"class", "property"},
			TokenModifiers: []string{},
		},
		Full: protocol.SemanticDelta{
			Delta: boolPtr(true),
		},
	}

	if supportsPullDiagnostics {
		capabilities.DiagnosticProvider = protocol.DiagnosticOptions{
			InterFileDependencies: false,
			WorkspaceDiagnostics:  true,
		}
	}

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    "design-tokens-language-server",
			Version: strPtr(req.Server.Version()),
		},
	}, nil
}

func boolPtr(b bool) *bool {
	return &b
}

func strPtr(s string) *string {
	return &s
}
