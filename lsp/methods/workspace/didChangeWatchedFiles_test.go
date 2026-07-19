package workspace

import (
	"context"
	"testing"

	"bennypowers.dev/asimonim/lsp/testutil"
	"bennypowers.dev/asimonim/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestHandleDidChangeWatchedFiles(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	req := types.NewRequestContext(ctx, nil)
	ctx.SetRootPath("/workspace")

	config := ctx.GetConfig()
	config.TokensFiles = []any{"tokens.json"}
	ctx.SetConfig(config)

	// Create a change event
	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  uri.URI("file:///workspace/tokens.json"),
				Type: protocol.FileChangeTypeChanged,
			},
		},
	}

	// Handle the change
	err := DidChangeWatchedFiles(req, params)
	if err != nil {
		t.Errorf("DidChangeWatchedFiles failed: %v", err)
	}
}

func TestHandleDidChangeWatchedFiles_MultipleChanges(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	req := types.NewRequestContext(ctx, nil)
	ctx.SetRootPath("/workspace")

	config := ctx.GetConfig()
	config.TokensFiles = []any{"tokens.json", "design-tokens.json"}
	ctx.SetConfig(config)

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  uri.URI("file:///workspace/tokens.json"),
				Type: protocol.FileChangeTypeChanged,
			},
			{
				URI:  uri.URI("file:///workspace/design-tokens.json"),
				Type: protocol.FileChangeTypeChanged,
			},
			{
				URI:  uri.URI("file:///workspace/package.json"), // Not a token file
				Type: protocol.FileChangeTypeChanged,
			},
		},
	}

	err := DidChangeWatchedFiles(req, params)
	if err != nil {
		t.Errorf("DidChangeWatchedFiles failed: %v", err)
	}
}

func TestHandleDidChangeWatchedFiles_DeletedFile(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	req := types.NewRequestContext(ctx, nil)
	ctx.SetRootPath("/workspace")

	config := ctx.GetConfig()
	config.TokensFiles = []any{"tokens.json"}
	ctx.SetConfig(config)

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  uri.URI("file:///workspace/tokens.json"),
				Type: protocol.FileChangeTypeDeleted,
			},
		},
	}

	err := DidChangeWatchedFiles(req, params)
	if err != nil {
		t.Errorf("DidChangeWatchedFiles failed: %v", err)
	}

	// The handler should have called RemoveLoadedFile to clean up the deleted file
	// This prevents stale entries that would cause reload errors
}

func TestHandleDidChangeWatchedFiles_DeletedAndModified(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	req := types.NewRequestContext(ctx, nil)
	ctx.SetRootPath("/workspace")

	config := ctx.GetConfig()
	config.TokensFiles = []any{"tokens.json", "design-tokens.json"}
	ctx.SetConfig(config)

	// Test that when one file is deleted and another is modified,
	// we only reload the remaining files
	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  uri.URI("file:///workspace/tokens.json"),
				Type: protocol.FileChangeTypeDeleted,
			},
			{
				URI:  uri.URI("file:///workspace/design-tokens.json"),
				Type: protocol.FileChangeTypeChanged,
			},
		},
	}

	err := DidChangeWatchedFiles(req, params)
	if err != nil {
		t.Errorf("DidChangeWatchedFiles failed: %v", err)
	}

	// Should have triggered a reload for the modified file
	// Should NOT have tried to reload the deleted file
}

func TestHandleDidChangeWatchedFiles_NonTokenFile(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	req := types.NewRequestContext(ctx, nil)
	ctx.SetRootPath("/workspace")

	config := ctx.GetConfig()
	config.TokensFiles = []any{"tokens.json"}
	ctx.SetConfig(config)

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  uri.URI("file:///workspace/package.json"), // Not a token file
				Type: protocol.FileChangeTypeChanged,
			},
		},
	}

	// Should not trigger a reload since it's not a token file
	err := DidChangeWatchedFiles(req, params)
	if err != nil {
		t.Errorf("DidChangeWatchedFiles failed: %v", err)
	}
}

func TestHandleDidChangeWatchedFiles_NewlyCreatedFile(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	req := types.NewRequestContext(ctx, nil)
	ctx.SetRootPath("/workspace")

	// Empty TokensFiles means no tokens are loaded initially
	config := ctx.GetConfig()
	config.TokensFiles = []any{}
	ctx.SetConfig(config)

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  uri.URI("file:///workspace/new-tokens.json"),
				Type: protocol.FileChangeTypeCreated,
			},
		},
	}

	// When a new token file is created and explicitly configured,
	// the reload should load it
	err := DidChangeWatchedFiles(req, params)
	if err != nil {
		t.Errorf("DidChangeWatchedFiles failed: %v", err)
	}

	// The handler should have triggered a reload which would
	// call LoadTokensFromConfig, which would discover the new file
}

func TestHandleDidChangeWatchedFiles_SkipsDiagnosticsWithPullModel(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	ctx.SetRootPath("/workspace")
	ctx.SetUsePullDiagnostics(true)

	req := types.NewRequestContext(ctx, context.Background())

	publishCalled := false
	ctx.PublishDiagnosticsFunc = func(_ context.Context, uri string) error {
		publishCalled = true
		return nil
	}

	ctx.IsTokenFileFunc = func(path string) bool {
		return path == "/workspace/tokens.json"
	}

	config := ctx.GetConfig()
	config.TokensFiles = []any{"/workspace/tokens.json"}
	ctx.SetConfig(config)

	_ = ctx.DocumentManager().DidOpen("file:///workspace/test.css", "css", 1, ".test { color: red; }")

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  uri.URI("file:///workspace/tokens.json"),
				Type: protocol.FileChangeTypeChanged,
			},
		},
	}

	err := DidChangeWatchedFiles(req, params)
	require.NoError(t, err)

	assert.False(t, publishCalled, "Should not publish diagnostics with pull model")
	assert.True(t, ctx.NotifyDiagnosticRefreshCalled, "Should have sent diagnostic refresh")
}

func TestHandleDidChangeWatchedFiles_EmptyChanges(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	req := types.NewRequestContext(ctx, nil)

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{},
	}

	err := DidChangeWatchedFiles(req, params)
	if err != nil {
		t.Errorf("DidChangeWatchedFiles failed: %v", err)
	}

	// Should not have triggered a reload
	if ctx.LoadTokensCalled {
		t.Error("Expected LoadTokensFromConfig NOT to be called with empty changes")
	}
}

func TestHandleDidChangeWatchedFiles_DeleteClearsTokens(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	req := types.NewRequestContext(ctx, nil)
	ctx.SetRootPath("/workspace")

	ctx.IsTokenFileFunc = func(path string) bool {
		return path == "/workspace/tokens.json"
	}

	config := ctx.GetConfig()
	config.TokensFiles = []any{"/workspace/tokens.json"}
	ctx.SetConfig(config)

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  uri.URI("file:///workspace/tokens.json"),
				Type: protocol.FileChangeTypeDeleted,
			},
		},
	}

	err := DidChangeWatchedFiles(req, params)
	if err != nil {
		t.Errorf("DidChangeWatchedFiles failed: %v", err)
	}

	// Should have called LoadTokensFromConfig after deletion
	if !ctx.LoadTokensCalled {
		t.Error("Expected LoadTokensFromConfig to be called after token file deletion")
	}
}

func TestHandleDidChangeWatchedFiles_PublishesDiagnostics(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	req := types.NewRequestContext(ctx, nil)
	ctx.SetRootPath("/workspace")

	// Set up GLSP context
	ctx.SetServerCtx(context.Background())

	// Track PublishDiagnostics calls
	publishedURIs := []string{}
	ctx.PublishDiagnosticsFunc = func(_ context.Context, uri string) error {
		publishedURIs = append(publishedURIs, uri)
		return nil
	}

	// Set up IsTokenFile to recognize the path
	ctx.IsTokenFileFunc = func(path string) bool {
		return path == "/workspace/tokens.json"
	}

	config := ctx.GetConfig()
	config.TokensFiles = []any{"/workspace/tokens.json"}
	ctx.SetConfig(config)

	// Open a document so we have something to publish diagnostics for
	_ = ctx.DocumentManager().DidOpen("file:///workspace/test.css", "css", 1, ".test { color: red; }")

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  uri.URI("file:///workspace/tokens.json"),
				Type: protocol.FileChangeTypeChanged,
			},
		},
	}

	// Handle the change
	err := DidChangeWatchedFiles(req, params)
	if err != nil {
		t.Errorf("DidChangeWatchedFiles failed: %v", err)
	}

	// Should have published diagnostics for the open document
	if len(publishedURIs) != 1 {
		t.Errorf("Expected 1 diagnostics publish, got %d", len(publishedURIs))
	}
	if len(publishedURIs) > 0 && publishedURIs[0] != "file:///workspace/test.css" {
		t.Errorf("Expected diagnostics for test.css, got %s", publishedURIs[0])
	}
}
