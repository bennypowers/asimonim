package lsp

import (
	"context"
	"fmt"
	"runtime/debug"

	"bennypowers.dev/asimonim/lsp/internal/log"
	"bennypowers.dev/asimonim/lsp/methods/lifecycle"
	"bennypowers.dev/asimonim/lsp/methods/textDocument"
	codeaction "bennypowers.dev/asimonim/lsp/methods/textDocument/codeAction"
	"bennypowers.dev/asimonim/lsp/methods/textDocument/completion"
	"bennypowers.dev/asimonim/lsp/methods/textDocument/definition"
	"bennypowers.dev/asimonim/lsp/methods/textDocument/diagnostic"
	documentcolor "bennypowers.dev/asimonim/lsp/methods/textDocument/documentColor"
	"bennypowers.dev/asimonim/lsp/methods/textDocument/hover"
	inlayhint "bennypowers.dev/asimonim/lsp/methods/textDocument/inlayHint"
	"bennypowers.dev/asimonim/lsp/methods/textDocument/references"
	semantictokens "bennypowers.dev/asimonim/lsp/methods/textDocument/semanticTokens"
	"bennypowers.dev/asimonim/lsp/methods/workspace"
	"bennypowers.dev/asimonim/lsp/types"
	"go.lsp.dev/protocol"
)

// handler implements protocol.Server by delegating to existing handler functions.
type handler struct {
	protocol.UnimplementedServer
	s *Server
}

// wrap provides panic recovery and logging for notification handler methods.
func (h *handler) wrap(ctx context.Context, methodName string, fn func(req *types.RequestContext) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("PANIC in %s: %v\nStack trace:\n%s", methodName, r, string(debug.Stack()))
			err = fmt.Errorf("internal error in %s", methodName)
		}
	}()
	log.Debug("%s started", methodName)
	req := types.NewRequestContext(h.s, ctx)
	err = fn(req)
	if err == nil && req.HasWarnings() {
		for _, w := range req.Warnings() {
			log.Warn("%s warning: %v", methodName, w)
			h.logToClient(ctx, protocol.MessageTypeWarning, fmt.Sprintf("%s warning: %v", methodName, w))
		}
	}
	if err != nil {
		log.Error("%s error: %v", methodName, err)
		h.logToClient(ctx, protocol.MessageTypeError, fmt.Sprintf("%s: %v", methodName, err))
		return fmt.Errorf("%s: %w", methodName, err)
	}
	log.Debug("%s completed successfully", methodName)
	return nil
}

// wrapResult provides panic recovery and logging for request handler methods that return a value.
func wrapResult[R any](h *handler, ctx context.Context, methodName string, fn func(req *types.RequestContext) (R, error)) (result R, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("PANIC in %s: %v\nStack trace:\n%s", methodName, r, string(debug.Stack()))
			err = fmt.Errorf("internal error in %s", methodName)
			var zero R
			result = zero
		}
	}()
	log.Debug("%s started", methodName)
	req := types.NewRequestContext(h.s, ctx)
	result, err = fn(req)
	if err == nil && req.HasWarnings() {
		for _, w := range req.Warnings() {
			log.Warn("%s warning: %v", methodName, w)
			h.logToClient(ctx, protocol.MessageTypeWarning, fmt.Sprintf("%s warning: %v", methodName, w))
		}
	}
	if err != nil {
		log.Error("%s error: %v", methodName, err)
		h.logToClient(ctx, protocol.MessageTypeError, fmt.Sprintf("%s: %v", methodName, err))
		return result, fmt.Errorf("%s: %w", methodName, err)
	}
	log.Debug("%s completed successfully", methodName)
	return result, nil
}

// logToClient sends a window/logMessage notification to the LSP client.
func (h *handler) logToClient(ctx context.Context, msgType protocol.MessageType, message string) {
	if h.s.client == nil {
		return
	}
	go func() {
		_ = h.s.client.LogMessage(ctx, &protocol.LogMessageParams{
			Type:    msgType,
			Message: message,
		})
	}()
}

func (h *handler) Initialize(ctx context.Context, params *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	// Detect pull diagnostics from typed params
	if params.Capabilities.TextDocument != nil && params.Capabilities.TextDocument.Diagnostic != nil {
		h.s.SetClientDiagnosticCapability(true)
	} else {
		h.s.SetClientDiagnosticCapability(false)
	}
	return wrapResult(h, ctx, "initialize", func(req *types.RequestContext) (*protocol.InitializeResult, error) {
		return lifecycle.Initialize(req, params)
	})
}

func (h *handler) Initialized(ctx context.Context, params *protocol.InitializedParams) error {
	return h.wrap(ctx, "initialized", func(req *types.RequestContext) error {
		return lifecycle.Initialized(req, params)
	})
}

func (h *handler) Shutdown(ctx context.Context) error {
	return h.wrap(ctx, "shutdown", func(req *types.RequestContext) error {
		return lifecycle.Shutdown(req)
	})
}

func (h *handler) SetTrace(ctx context.Context, params *protocol.SetTraceParams) error {
	return h.wrap(ctx, "$/setTrace", func(req *types.RequestContext) error {
		return lifecycle.SetTrace(req, params)
	})
}

func (h *handler) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	return h.wrap(ctx, "textDocument/didOpen", func(req *types.RequestContext) error {
		return textDocument.DidOpen(req, params)
	})
}

func (h *handler) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	return h.wrap(ctx, "textDocument/didChange", func(req *types.RequestContext) error {
		return textDocument.DidChange(req, params)
	})
}

func (h *handler) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) error {
	return h.wrap(ctx, "textDocument/didClose", func(req *types.RequestContext) error {
		return textDocument.DidClose(req, params)
	})
}

func (h *handler) Hover(ctx context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	return wrapResult(h, ctx, "textDocument/hover", func(req *types.RequestContext) (*protocol.Hover, error) {
		return hover.Hover(req, params)
	})
}

func (h *handler) Completion(ctx context.Context, params *protocol.CompletionParams) (protocol.CompletionResult, error) {
	return wrapResult(h, ctx, "textDocument/completion", func(req *types.RequestContext) (protocol.CompletionResult, error) {
		return completion.Completion(req, params)
	})
}

func (h *handler) CompletionResolve(ctx context.Context, params *protocol.CompletionItem) (*protocol.CompletionItem, error) {
	return wrapResult(h, ctx, "completionItem/resolve", func(req *types.RequestContext) (*protocol.CompletionItem, error) {
		return completion.CompletionResolve(req, params)
	})
}

func (h *handler) Definition(ctx context.Context, params *protocol.DefinitionParams) (protocol.DefinitionResult, error) {
	return wrapResult(h, ctx, "textDocument/definition", func(req *types.RequestContext) (protocol.DefinitionResult, error) {
		return definition.Definition(req, params)
	})
}

func (h *handler) References(ctx context.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	return wrapResult(h, ctx, "textDocument/references", func(req *types.RequestContext) ([]protocol.Location, error) {
		return references.References(req, params)
	})
}

func (h *handler) DocumentColor(ctx context.Context, params *protocol.DocumentColorParams) ([]protocol.ColorInformation, error) {
	return wrapResult(h, ctx, "textDocument/documentColor", func(req *types.RequestContext) ([]protocol.ColorInformation, error) {
		return documentcolor.DocumentColor(req, params)
	})
}

func (h *handler) ColorPresentation(ctx context.Context, params *protocol.ColorPresentationParams) ([]protocol.ColorPresentation, error) {
	return wrapResult(h, ctx, "textDocument/colorPresentation", func(req *types.RequestContext) ([]protocol.ColorPresentation, error) {
		return documentcolor.ColorPresentation(req, params)
	})
}

func (h *handler) CodeAction(ctx context.Context, params *protocol.CodeActionParams) ([]protocol.CommandOrCodeAction, error) {
	return wrapResult(h, ctx, "textDocument/codeAction", func(req *types.RequestContext) ([]protocol.CommandOrCodeAction, error) {
		return codeaction.CodeAction(req, params)
	})
}

func (h *handler) CodeActionResolve(ctx context.Context, params *protocol.CodeAction) (*protocol.CodeAction, error) {
	return wrapResult(h, ctx, "codeAction/resolve", func(req *types.RequestContext) (*protocol.CodeAction, error) {
		return codeaction.CodeActionResolve(req, params)
	})
}

func (h *handler) SemanticTokensFull(ctx context.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	return wrapResult(h, ctx, "textDocument/semanticTokens/full", func(req *types.RequestContext) (*protocol.SemanticTokens, error) {
		return semantictokens.SemanticTokensFull(req, params)
	})
}

func (h *handler) SemanticTokensFullDelta(ctx context.Context, params *protocol.SemanticTokensDeltaParams) (protocol.SemanticTokensDeltaResult, error) {
	return wrapResult(h, ctx, "textDocument/semanticTokens/full/delta", func(req *types.RequestContext) (protocol.SemanticTokensDeltaResult, error) {
		return semantictokens.SemanticTokensFullDelta(req, params)
	})
}

func (h *handler) Diagnostic(ctx context.Context, params *protocol.DocumentDiagnosticParams) (protocol.DocumentDiagnosticReport, error) {
	return wrapResult(h, ctx, "textDocument/diagnostic", func(req *types.RequestContext) (protocol.DocumentDiagnosticReport, error) {
		return diagnostic.DocumentDiagnostic(req, params)
	})
}

func (h *handler) DiagnosticWorkspace(ctx context.Context, params *protocol.WorkspaceDiagnosticParams) (*protocol.WorkspaceDiagnosticReport, error) {
	return wrapResult(h, ctx, "workspace/diagnostic", func(req *types.RequestContext) (*protocol.WorkspaceDiagnosticReport, error) {
		return workspace.WorkspaceDiagnostic(req, params)
	})
}

func (h *handler) InlayHint(ctx context.Context, params *protocol.InlayHintParams) ([]protocol.InlayHint, error) {
	return wrapResult(h, ctx, "textDocument/inlayHint", func(req *types.RequestContext) ([]protocol.InlayHint, error) {
		return inlayhint.InlayHint(req, params)
	})
}

func (h *handler) DidChangeConfiguration(ctx context.Context, params *protocol.DidChangeConfigurationParams) error {
	return h.wrap(ctx, "workspace/didChangeConfiguration", func(req *types.RequestContext) error {
		return workspace.DidChangeConfiguration(req, params)
	})
}

func (h *handler) DidChangeWatchedFiles(ctx context.Context, params *protocol.DidChangeWatchedFilesParams) error {
	return h.wrap(ctx, "workspace/didChangeWatchedFiles", func(req *types.RequestContext) error {
		return workspace.DidChangeWatchedFiles(req, params)
	})
}
