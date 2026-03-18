package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"bennypowers.dev/asimonim/lsp/testutil"
	"bennypowers.dev/asimonim/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// loadConfigFixture loads a JSON fixture file as a settings map for parseConfiguration.
func loadConfigFixture(t *testing.T, name string) map[string]any {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(thisFile), "testdata", "config", name)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read fixture %s", name)
	var settings map[string]any
	require.NoError(t, json.Unmarshal(data, &settings), "failed to parse fixture %s", name)
	return settings
}

func TestDidChangeConfiguration_WithValidConfig(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)
	ctx.SetGLSPContext(glspCtx)

	// Prepare configuration with tokens files
	settings := map[string]any{
		"designTokensLanguageServer": map[string]any{
			"prefix": "--custom",
			"tokensFiles": []any{
				"tokens.json",
			},
		},
	}

	params := &protocol.DidChangeConfigurationParams{
		Settings: settings,
	}

	err := DidChangeConfiguration(req, params)
	require.NoError(t, err)

	// Verify config was updated
	config := ctx.GetConfig()
	assert.Equal(t, "--custom", config.Prefix)
}

func TestDidChangeConfiguration_WithNilSettings(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	params := &protocol.DidChangeConfigurationParams{
		Settings: nil,
	}

	err := DidChangeConfiguration(req, params)
	require.NoError(t, err)

	// Should use default config
	config := ctx.GetConfig()
	assert.Equal(t, types.DefaultConfig().Prefix, config.Prefix)
}

func TestDidChangeConfiguration_WithInvalidSettings(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Settings that's not a map
	params := &protocol.DidChangeConfigurationParams{
		Settings: "invalid",
	}

	err := DidChangeConfiguration(req, params)
	// Should not error (warns and uses defaults)
	require.NoError(t, err)
}

func TestDidChangeConfiguration_WithAlternateKey(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Using hyphenated key instead of camelCase
	settings := map[string]any{
		"design-tokens-language-server": map[string]any{
			"prefix": "--alt",
		},
	}

	params := &protocol.DidChangeConfigurationParams{
		Settings: settings,
	}

	err := DidChangeConfiguration(req, params)
	require.NoError(t, err)

	config := ctx.GetConfig()
	assert.Equal(t, "--alt", config.Prefix)
}

func TestDidChangeConfiguration_WithoutGLSPContext(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	// Don't set GLSP context
	req := types.NewRequestContext(ctx, nil)

	settings := map[string]any{
		"designTokensLanguageServer": map[string]any{
			"prefix": "--test",
		},
	}

	params := &protocol.DidChangeConfigurationParams{
		Settings: settings,
	}

	// Should not panic when glspCtx is nil
	err := DidChangeConfiguration(req, params)
	require.NoError(t, err)
}

func TestDidChangeConfiguration_PublishesDiagnosticsForOpenDocs(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)
	ctx.SetGLSPContext(glspCtx)

	// Open a document
	_ = ctx.DocumentManager().DidOpen("file:///workspace/test.css", "css", 1, ".test { color: red; }")

	// Track PublishDiagnostics calls
	publishedURIs := []string{}
	ctx.PublishDiagnosticsFunc = func(context *glsp.Context, uri string) error {
		publishedURIs = append(publishedURIs, uri)
		return nil
	}

	settings := map[string]any{
		"designTokensLanguageServer": map[string]any{
			"prefix": "--new",
		},
	}

	params := &protocol.DidChangeConfigurationParams{
		Settings: settings,
	}

	err := DidChangeConfiguration(req, params)
	require.NoError(t, err)

	// Should have published diagnostics for the open document
	assert.Len(t, publishedURIs, 1)
	assert.Equal(t, "file:///workspace/test.css", publishedURIs[0])
}

func TestDidChangeConfiguration_SkipsDiagnosticsWithPullModel(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)
	ctx.SetGLSPContext(glspCtx)

	// Enable pull diagnostics
	ctx.SetUsePullDiagnostics(true)

	// Open a document
	_ = ctx.DocumentManager().DidOpen("file:///workspace/test.css", "css", 1, ".test { color: red; }")

	// Track PublishDiagnostics calls
	publishCalled := false
	ctx.PublishDiagnosticsFunc = func(context *glsp.Context, uri string) error {
		publishCalled = true
		return nil
	}

	settings := map[string]any{
		"designTokensLanguageServer": map[string]any{
			"prefix": "--new",
		},
	}

	params := &protocol.DidChangeConfigurationParams{
		Settings: settings,
	}

	err := DidChangeConfiguration(req, params)
	require.NoError(t, err)

	// Should NOT have published diagnostics
	assert.False(t, publishCalled, "Should not publish diagnostics with pull model")
}

func TestDidChangeConfiguration_WithGroupMarkers(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	settings := map[string]any{
		"designTokensLanguageServer": map[string]any{
			"groupMarkers": []any{"value", "DEFAULT"},
		},
	}

	params := &protocol.DidChangeConfigurationParams{
		Settings: settings,
	}

	err := DidChangeConfiguration(req, params)
	require.NoError(t, err)

	config := ctx.GetConfig()
	assert.True(t, config.GroupMarkersSet)
	assert.Equal(t, []string{"value", "DEFAULT"}, config.GroupMarkers)
}

func TestParseConfiguration_DefaultConfig(t *testing.T) {
	config, err := parseConfiguration(nil)
	require.NoError(t, err)
	assert.Equal(t, types.DefaultConfig(), config)
}

func TestParseConfiguration_ValidSettings(t *testing.T) {
	settings := map[string]any{
		"designTokensLanguageServer": map[string]any{
			"prefix": "--my-prefix",
			"tokensFiles": []any{
				"tokens/colors.json",
				"tokens/spacing.json",
			},
			"groupMarkers": []any{"value"},
		},
	}

	config, err := parseConfiguration(settings)
	require.NoError(t, err)
	assert.Equal(t, "--my-prefix", config.Prefix)
	assert.Len(t, config.TokensFiles, 2)
	assert.Len(t, config.GroupMarkers, 1)
	assert.Equal(t, "value", config.GroupMarkers[0])
}

func TestParseConfiguration_WithComplexTokensFiles(t *testing.T) {
	settings := map[string]any{
		"designTokensLanguageServer": map[string]any{
			"prefix": "--",
			"tokensFiles": []any{
				"simple.json",
				map[string]any{
					"path":   "complex.json",
					"prefix": "--override",
				},
			},
		},
	}

	config, err := parseConfiguration(settings)
	require.NoError(t, err)
	assert.Len(t, config.TokensFiles, 2)
}

func TestParseConfiguration_InvalidMap(t *testing.T) {
	// Settings that's not a map
	settings := "not a map"

	_, err := parseConfiguration(settings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a map")
}

func TestParseConfiguration_MissingKey(t *testing.T) {
	// Map without our configuration key
	settings := map[string]any{
		"someOtherKey": map[string]any{
			"value": "test",
		},
	}

	config, err := parseConfiguration(settings)
	require.NoError(t, err)
	// Should return default config
	assert.Equal(t, types.DefaultConfig(), config)
}

func TestParseConfiguration_InvalidJSON(t *testing.T) {
	// Create a value that can't be marshaled to JSON
	// (functions can't be marshaled)
	settings := map[string]any{
		"designTokensLanguageServer": map[string]any{
			"invalidField": func() {}, // Functions can't be marshaled to JSON
		},
	}

	_, err := parseConfiguration(settings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "marshal")
}

func TestParseConfiguration_NetworkFallback(t *testing.T) {
	settings := map[string]any{
		"designTokensLanguageServer": map[string]any{
			"networkFallback": true,
			"networkTimeout":  float64(60),
		},
	}

	config, err := parseConfiguration(settings)
	require.NoError(t, err)
	assert.True(t, config.NetworkFallback)
	assert.Equal(t, 60, config.NetworkTimeout)
}

func TestParseConfiguration_CDNProvider(t *testing.T) {
	settings := map[string]any{
		"designTokensLanguageServer": map[string]any{
			"networkFallback": true,
			"cdn":             "jsdelivr",
		},
	}

	config, err := parseConfiguration(settings)
	require.NoError(t, err)
	assert.Equal(t, "jsdelivr", config.CDN)
}

func TestParseConfiguration_NetworkFallbackDefaults(t *testing.T) {
	settings := map[string]any{
		"designTokensLanguageServer": map[string]any{
			"prefix": "ds",
		},
	}

	config, err := parseConfiguration(settings)
	require.NoError(t, err)
	assert.False(t, config.NetworkFallback)
	assert.Equal(t, 0, config.NetworkTimeout)
	assert.Equal(t, "", config.CDN)
}

func TestParseConfiguration_AsimonimNamespace(t *testing.T) {
	// asimonim-namespace.json: {"asimonim": {"prefix": "--asimonim", "tokensFiles": ["tokens.json"]}}
	settings := loadConfigFixture(t, "asimonim-namespace.json")
	config, err := parseConfiguration(settings)
	require.NoError(t, err)
	assert.Equal(t, "--asimonim", config.Prefix)
	require.Len(t, config.TokensFiles, 1)
	assert.Equal(t, "tokens.json", config.TokensFiles[0])
}

func TestParseConfiguration_AsimonimTakesPrecedenceOverLegacy(t *testing.T) {
	// both-namespaces.json: {"asimonim": {"prefix": "--new"}, "designTokensLanguageServer": {"prefix": "--old"}}
	settings := loadConfigFixture(t, "both-namespaces.json")
	config, err := parseConfiguration(settings)
	require.NoError(t, err)
	assert.Equal(t, "--new", config.Prefix)
}

func TestDidChangeConfiguration_WithAsimonimNamespace(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)
	ctx.SetGLSPContext(glspCtx)

	settings := loadConfigFixture(t, "asimonim-namespace.json")

	params := &protocol.DidChangeConfigurationParams{
		Settings: settings,
	}

	err := DidChangeConfiguration(req, params)
	require.NoError(t, err)

	config := ctx.GetConfig()
	assert.Equal(t, "--asimonim", config.Prefix)
}

func TestParseConfiguration_AsimonimInvalidType(t *testing.T) {
	// asimonim-invalid-type.json: {"asimonim": "not an object"}
	settings := loadConfigFixture(t, "asimonim-invalid-type.json")
	_, err := parseConfiguration(settings)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be an object")
}
