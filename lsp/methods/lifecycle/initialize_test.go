package lifecycle

import (
	"testing"

	"bennypowers.dev/asimonim/lsp/internal/uriutil"
	"bennypowers.dev/asimonim/lsp/testutil"
	"bennypowers.dev/asimonim/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/bennypowers/glsp"
	protocol "github.com/bennypowers/glsp/protocol_3_17"
)

func TestInitialize(t *testing.T) {
	t.Run("sets root URI from params.RootURI", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)
		rootURI := "file:///workspace"

		params := &protocol.InitializeParams{}
		params.RootURI = &rootURI

		result, err := Initialize(req, params)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify root was set
		assert.Equal(t, "file:///workspace", ctx.RootURI())
		assert.Equal(t, "/workspace", ctx.RootPath())
	})

	t.Run("sets root path from params.RootPath", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)
		rootPath := "/workspace"

		params := &protocol.InitializeParams{}
		params.RootPath = &rootPath

		result, err := Initialize(req, params)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify root was set
		assert.Equal(t, "/workspace", ctx.RootPath())
		assert.Equal(t, "file:///workspace", ctx.RootURI())
	})

	t.Run("returns server capabilities", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		params := &protocol.InitializeParams{}

		result, err := Initialize(req, params)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Result should have Capabilities and ServerInfo fields
		initResult := result.(protocol.InitializeResult)

		assert.NotNil(t, initResult.Capabilities)
		assert.NotNil(t, initResult.ServerInfo)
		assert.Equal(t, "design-tokens-language-server", initResult.ServerInfo.Name)
		assert.NotNil(t, initResult.ServerInfo.Version)
		assert.NotEqual(t, "", *initResult.ServerInfo.Version, "Version should not be empty")

		// Verify version matches the server context
		assert.Equal(t, ctx.Version(), *initResult.ServerInfo.Version,
			"ServerInfo version should match server Version()")
	})

	t.Run("capabilities include all LSP features", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		// Simulate LSP 3.17 client with diagnostic capability
		// (In practice, CustomHandler detects this from raw JSON during initialize)
		ctx.SetClientDiagnosticCapability(true)

		// Provide client capabilities to enable pull diagnostics (LSP 3.17)
		params := &protocol.InitializeParams{
			Capabilities: protocol.ClientCapabilities{
				TextDocument: &protocol.TextDocumentClientCapabilities{},
			},
		}

		result, err := Initialize(req, params)
		require.NoError(t, err)

		initResult := result.(protocol.InitializeResult)

		caps := initResult.Capabilities

		// Verify all expected capabilities are present
		assert.NotNil(t, caps.TextDocumentSync)
		assert.NotNil(t, caps.HoverProvider)
		assert.NotNil(t, caps.CompletionProvider)
		assert.NotNil(t, caps.DefinitionProvider)
		assert.NotNil(t, caps.ReferencesProvider)
		assert.NotNil(t, caps.CodeActionProvider)
		assert.NotNil(t, caps.ColorProvider)
		assert.NotNil(t, caps.SemanticTokensProvider)
		assert.NotNil(t, caps.DiagnosticProvider)

		// Verify completion provider options
		assert.NotNil(t, caps.CompletionProvider.ResolveProvider)
		assert.True(t, *caps.CompletionProvider.ResolveProvider)
		assert.Equal(t, []string{"-"}, caps.CompletionProvider.TriggerCharacters)

		codeActionProvider, ok := caps.CodeActionProvider.(protocol.CodeActionOptions)
		assert.True(t, ok)
		assert.NotNil(t, codeActionProvider.ResolveProvider)
		assert.True(t, *codeActionProvider.ResolveProvider)
	})

	t.Run("handles client info", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		clientVersion := "1.85.0"
		params := &protocol.InitializeParams{}
		params.ClientInfo = &struct {
			Name    string  `json:"name"`
			Version *string `json:"version,omitempty"`
		}{
			Name:    "vscode",
			Version: &clientVersion,
		}

		result, err := Initialize(req, params)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("handles nil params gracefully", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		params := &protocol.InitializeParams{}

		result, err := Initialize(req, params)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should still return valid capabilities
		assert.Empty(t, ctx.RootURI())
		assert.Empty(t, ctx.RootPath())
	})
}

func TestInitialize_StoresClientCapabilities(t *testing.T) {
	t.Run("stores client capabilities during initialization", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		textDoc := &protocol.TextDocumentClientCapabilities{}
		textDoc.Completion = &protocol.CompletionClientCapabilities{
			CompletionItem: &struct {
				SnippetSupport          *bool                   `json:"snippetSupport,omitempty"`
				CommitCharactersSupport *bool                   `json:"commitCharactersSupport,omitempty"`
				DocumentationFormat     []protocol.MarkupKind   `json:"documentationFormat,omitempty"`
				DeprecatedSupport       *bool                   `json:"deprecatedSupport,omitempty"`
				PreselectSupport        *bool                   `json:"preselectSupport,omitempty"`
				TagSupport              *struct {
					ValueSet []protocol.CompletionItemTag `json:"valueSet"`
				} `json:"tagSupport,omitempty"`
				InsertReplaceSupport *bool `json:"insertReplaceSupport,omitempty"`
				ResolveSupport       *struct {
					Properties []string `json:"properties"`
				} `json:"resolveSupport,omitempty"`
				InsertTextModeSupport *struct {
					ValueSet []protocol.InsertTextMode `json:"valueSet"`
				} `json:"insertTextModeSupport,omitempty"`
			}{
				SnippetSupport: boolPtr(true),
			},
		}
		params := &protocol.InitializeParams{
			Capabilities: protocol.ClientCapabilities{
				TextDocument: textDoc,
			},
		}

		result, err := Initialize(req, params)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify client capabilities were stored
		storedCaps := ctx.ClientCapabilities()
		require.NotNil(t, storedCaps, "ClientCapabilities should be stored")
		require.NotNil(t, storedCaps.TextDocument)
		require.NotNil(t, storedCaps.TextDocument.Completion)
		require.NotNil(t, storedCaps.TextDocument.Completion.CompletionItem)
		assert.True(t, *storedCaps.TextDocument.Completion.CompletionItem.SnippetSupport)
	})

	t.Run("stores empty capabilities when none provided", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		params := &protocol.InitializeParams{}

		result, err := Initialize(req, params)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Capabilities should still be stored (even if empty)
		storedCaps := ctx.ClientCapabilities()
		require.NotNil(t, storedCaps, "ClientCapabilities should be stored even when empty")
	})
}

func TestPathConversion(t *testing.T) {
	t.Run("uriToPath strips file:// prefix", func(t *testing.T) {
		tests := []struct {
			name string
			uri  string
			want string
		}{
			{
				name: "simple path",
				uri:  "file:///workspace",
				want: "/workspace",
			},
			{
				name: "nested path",
				uri:  "file:///home/user/project",
				want: "/home/user/project",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := uriutil.URIToPath(tt.uri)
				assert.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("pathToURI adds file:// prefix", func(t *testing.T) {
		tests := []struct {
			name string
			path string
			want string
		}{
			{
				name: "simple path",
				path: "/workspace",
				want: "file:///workspace",
			},
			{
				name: "nested path",
				path: "/home/user/project",
				want: "file:///home/user/project",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := uriutil.PathToURI(tt.path)
				assert.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("round trip conversion", func(t *testing.T) {
		paths := []string{
			"/workspace",
			"/home/user/project",
		}

		for _, path := range paths {
			uri := uriutil.PathToURI(path)
			got := uriutil.URIToPath(uri)
			assert.Equal(t, path, got, "round trip should preserve path")
		}
	})
}
