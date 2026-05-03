package workspace

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"bennypowers.dev/asimonim/lsp/internal/tokens"
	"bennypowers.dev/asimonim/lsp/testutil"
	"bennypowers.dev/asimonim/lsp/types"
	"bennypowers.dev/asimonim/schema"
	rootutil "bennypowers.dev/asimonim/testutil"
	"github.com/bennypowers/glsp"
	protocol "github.com/bennypowers/glsp/protocol_3_17"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func shouldUpdate() bool {
	f := flag.Lookup("update")
	if f == nil {
		return false
	}
	return f.Value.String() == "true"
}

// loadFixtureCSS reads a CSS fixture file from testdata/workspace-diagnostic/.
func loadFixtureCSS(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "workspace-diagnostic", name))
	require.NoError(t, err)
	return string(data)
}

// goldenReport is a serializable subset of WorkspaceFullDocumentDiagnosticReport
// for golden file comparison. Omits pointer fields that don't round-trip cleanly.
type goldenReport struct {
	URI         string                `json:"uri"`
	Kind        string                `json:"kind"`
	Version     int32                 `json:"version"`
	Diagnostics []protocol.Diagnostic `json:"diagnostics"`
}

func TestWorkspaceDiagnostic_Golden(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Load tokens from fixture (color.old: deprecated, color.primary: #0000ff)
	fixtureTokens := rootutil.ParseFixtureTokens(t, "workspace-diagnostic", schema.Draft)
	for _, tok := range fixtureTokens {
		_ = ctx.TokenManager().Add(tok)
	}

	// Open documents from fixtures
	_ = ctx.DocumentManager().DidOpen(
		"file:///workspace/deprecated.css", "css", 1,
		loadFixtureCSS(t, "deprecated.css"),
	)
	_ = ctx.DocumentManager().DidOpen(
		"file:///workspace/wrong-fallback.css", "css", 2,
		loadFixtureCSS(t, "wrong-fallback.css"),
	)
	_ = ctx.DocumentManager().DidOpen(
		"file:///workspace/clean.css", "css", 3,
		loadFixtureCSS(t, "clean.css"),
	)

	result, err := WorkspaceDiagnostic(req, &protocol.WorkspaceDiagnosticParams{})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Items, 3)

	// Build sorted golden-comparable output
	reports := make([]goldenReport, 0, len(result.Items))
	for _, item := range result.Items {
		report := item.(protocol.WorkspaceFullDocumentDiagnosticReport)
		var ver int32
		if report.Version != nil {
			ver = *report.Version
		}
		reports = append(reports, goldenReport{
			URI:         report.URI,
			Kind:        report.Kind,
			Version:     ver,
			Diagnostics: report.Items,
		})
	}
	sort.Slice(reports, func(i, j int) bool { return reports[i].URI < reports[j].URI })

	actual, err := json.MarshalIndent(reports, "", "  ")
	require.NoError(t, err)
	actual = append(actual, '\n')

	goldenPath := filepath.Join("testdata", "workspace-diagnostic", "golden.json")

	if shouldUpdate() {
		err := os.WriteFile(goldenPath, actual, 0644)
		require.NoError(t, err)
		t.Log("Updated golden file")
		return
	}

	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "Golden file missing; run with -update to create")

	assert.JSONEq(t, string(expected), string(actual))
}

func TestWorkspaceDiagnostic_NoDocuments(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	result, err := WorkspaceDiagnostic(req, &protocol.WorkspaceDiagnosticParams{})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Items)
}

func TestWorkspaceDiagnostic_NonCSSDocumentsIncludedEmpty(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:       "color.old",
		Value:      "#ff0000",
		Type:       "color",
		Deprecated: true,
	})

	_ = ctx.DocumentManager().DidOpen(
		"file:///test.css", "css", 1,
		loadFixtureCSS(t, "deprecated.css"),
	)
	_ = ctx.DocumentManager().DidOpen("file:///data.json", "json", 1, `{"test": true}`)

	result, err := WorkspaceDiagnostic(req, &protocol.WorkspaceDiagnosticParams{})
	require.NoError(t, err)
	require.Len(t, result.Items, 2)

	reportsByURI := make(map[string]protocol.WorkspaceFullDocumentDiagnosticReport)
	for _, item := range result.Items {
		report := item.(protocol.WorkspaceFullDocumentDiagnosticReport)
		reportsByURI[report.URI] = report
	}

	// CSS document: 1 diagnostic (deprecated)
	assert.Len(t, reportsByURI["file:///test.css"].Items, 1)
	// JSON document: 0 diagnostics (not CSS)
	assert.Empty(t, reportsByURI["file:///data.json"].Items)
}

func TestCollectWorkspaceDiagnostics_SkipsOnError(t *testing.T) {
	ctx := testutil.NewMockServerContext()

	_ = ctx.DocumentManager().DidOpen("file:///ok.css", "css", 1, `.a { color: red; }`)
	_ = ctx.DocumentManager().DidOpen("file:///bad.css", "css", 2, `.b { color: blue; }`)

	// Injected diagnostics provider that errors for bad.css
	failingFn := func(_ types.ServerContext, uri string) ([]protocol.Diagnostic, error) {
		if uri == "file:///bad.css" {
			return nil, fmt.Errorf("parse failure")
		}
		return []protocol.Diagnostic{}, nil
	}

	result, err := collectWorkspaceDiagnostics(ctx, failingFn)
	require.NoError(t, err)
	require.NotNil(t, result)

	// bad.css skipped, only ok.css in results
	require.Len(t, result.Items, 1)
	report := result.Items[0].(protocol.WorkspaceFullDocumentDiagnosticReport)
	assert.Equal(t, "file:///ok.css", report.URI)
}
