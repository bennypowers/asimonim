package lsp

import (
	"context"
	"fmt"
	"runtime/debug"

	"bennypowers.dev/asimonim/lsp/internal/log"
	"bennypowers.dev/asimonim/lsp/methods/workspace"
	"bennypowers.dev/asimonim/lsp/types"
	"github.com/bennypowers/glsp"
)

// glspToCtx extracts a context.Context from a glsp.Context.
// Migration shim: will be removed when glsp is fully replaced.
func glspToCtx(g *glsp.Context) context.Context {
	if g != nil && g.Context != nil {
		return g.Context
	}
	return context.Background()
}

// method wraps an LSP handler that returns (result, error) with middleware
// Returns the underlying function type so it's compatible with protocol.Handler field types
func method[P, R any](
	s types.ServerContext,
	methodName string,
	handler func(*types.RequestContext, P) (R, error),
) func(*glsp.Context, P) (R, error) {
	return func(glspCtx *glsp.Context, params P) (result R, err error) {
		// Panic recovery - prevents LSP server crashes
		defer func() {
			if r := recover(); r != nil {
				stackTrace := string(debug.Stack())
				log.Error("PANIC in %s: %v\nStack trace:\n%s",
					methodName, r, stackTrace)
				// Log panic to LSP client
				workspace.LogError(glspCtx, "Internal error in %s: %v", methodName, r)
				err = fmt.Errorf("internal error in %s", methodName)
				var zero R
				result = zero
			}
		}()

		// Request logging
		log.Debug("%s started", methodName)

		req := types.NewRequestContext(s, glspToCtx(glspCtx))

		result, err = handler(req, params)

		if err == nil && req.HasWarnings() {
			for _, w := range req.Warnings() {
				workspace.LogWarning(glspCtx, "%s warning: %v", methodName, w)
			}
		}

		if err != nil {
			log.Error("%s error: %v", methodName, err)
			workspace.LogError(glspCtx, "%s: %v", methodName, err)
			return result, fmt.Errorf("%s: %w", methodName, err)
		}

		log.Debug("%s completed successfully", methodName)
		return result, nil
	}
}

// notify wraps an LSP notification handler that returns only error
func notify[P any](
	s types.ServerContext,
	methodName string,
	handler func(*types.RequestContext, P) error,
) func(*glsp.Context, P) error {
	return func(glspCtx *glsp.Context, params P) (err error) {
		defer func() {
			if r := recover(); r != nil {
				stackTrace := string(debug.Stack())
				log.Error("PANIC in %s: %v\nStack trace:\n%s",
					methodName, r, stackTrace)
				workspace.LogError(glspCtx, "Internal error in %s: %v", methodName, r)
				err = fmt.Errorf("internal error in %s", methodName)
			}
		}()

		log.Debug("%s started", methodName)

		req := types.NewRequestContext(s, glspToCtx(glspCtx))

		err = handler(req, params)

		if err == nil && req.HasWarnings() {
			for _, w := range req.Warnings() {
				workspace.LogWarning(glspCtx, "%s warning: %v", methodName, w)
			}
		}

		if err != nil {
			log.Error("%s error: %v", methodName, err)
			workspace.LogError(glspCtx, "%s: %v", methodName, err)
			return fmt.Errorf("%s: %w", methodName, err)
		}

		log.Debug("%s completed successfully", methodName)
		return nil
	}
}

// noParam wraps an LSP handler that takes no params (like Shutdown)
func noParam(
	s types.ServerContext,
	methodName string,
	handler func(*types.RequestContext) error,
) func(*glsp.Context) error {
	return func(glspCtx *glsp.Context) (err error) {
		defer func() {
			if r := recover(); r != nil {
				stackTrace := string(debug.Stack())
				log.Error("PANIC in %s: %v\nStack trace:\n%s",
					methodName, r, stackTrace)
				workspace.LogError(glspCtx, "Internal error in %s: %v", methodName, r)
				err = fmt.Errorf("internal error in %s", methodName)
			}
		}()

		log.Debug("%s started", methodName)

		req := types.NewRequestContext(s, glspToCtx(glspCtx))

		err = handler(req)

		if err == nil && req.HasWarnings() {
			for _, w := range req.Warnings() {
				workspace.LogWarning(glspCtx, "%s warning: %v", methodName, w)
			}
		}

		if err != nil {
			log.Error("%s error: %v", methodName, err)
			workspace.LogError(glspCtx, "%s: %v", methodName, err)
			return fmt.Errorf("%s: %w", methodName, err)
		}

		log.Debug("%s completed successfully", methodName)
		return nil
	}
}
