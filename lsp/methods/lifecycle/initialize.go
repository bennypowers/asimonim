package lifecycle

import (
	"bennypowers.dev/asimonim/lsp/internal/log"

	"bennypowers.dev/asimonim/lsp/internal/uriutil"
	"bennypowers.dev/asimonim/lsp/types"
	"go.lsp.dev/protocol"
)

// Initialize handles the LSP initialize request
func Initialize(req *types.RequestContext, params *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	clientName := "unknown"
	if params.ClientInfo.Name != "" {
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
		rootURIStr := string(*params.RootURI)
		req.Server.SetRootURI(rootURIStr)
		// Convert URI to file path
		req.Server.SetRootPath(uriutil.URIToPath(rootURIStr))
		log.Info("Workspace root: %s", req.Server.RootPath())
	} else if rootPath, ok := params.RootPath.Get(); ok {
		req.Server.SetRootPath(rootPath)
		req.Server.SetRootURI(uriutil.PathToURI(rootPath))
		log.Info("Workspace root (from rootPath): %s", req.Server.RootPath())
	}

	syncKind := protocol.TextDocumentSyncKindIncremental
	capabilities := protocol.ServerCapabilities{}
	capabilities.TextDocumentSync = &protocol.TextDocumentSyncOptions{
		OpenClose: boolPtr(true),
		Change:    &syncKind,
	}
	capabilities.HoverProvider = protocol.Boolean(true)
	capabilities.CompletionProvider = &protocol.CompletionOptions{
		TriggerCharacters: []string{"-"},
		ResolveProvider:   boolPtr(true),
	}
	capabilities.DefinitionProvider = protocol.Boolean(true)
	capabilities.ReferencesProvider = protocol.Boolean(true)
	capabilities.CodeActionProvider = &protocol.CodeActionOptions{
		ResolveProvider: boolPtr(true),
	}
	capabilities.ColorProvider = protocol.Boolean(true)
	capabilities.InlayHintProvider = protocol.Boolean(true)
	capabilities.SemanticTokensProvider = &protocol.SemanticTokensOptions{
		Legend: protocol.SemanticTokensLegend{
			TokenTypes:     []string{"class", "property"},
			TokenModifiers: []string{},
		},
		Full: &protocol.SemanticTokensFullDelta{
			Delta: boolPtr(true),
		},
	}

	if supportsPullDiagnostics {
		capabilities.DiagnosticProvider = &protocol.DiagnosticOptions{
			Identifier:            strPtr("asimonim"),
			InterFileDependencies: false,
			WorkspaceDiagnostics:  true,
		}
	}

	return &protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: protocol.ServerInfo{
			Name:    "design-tokens-language-server",
			Version: protocol.NewOptional(req.Server.Version()),
		},
	}, nil
}

func boolPtr(b bool) *bool {
	return &b
}

func strPtr(s string) *string {
	return &s
}
