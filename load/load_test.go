/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package load_test

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"bennypowers.dev/asimonim/load"
	"bennypowers.dev/asimonim/schema"
)

//go:embed testdata/cdn-fallback.json
var cdnFallbackFixture []byte

func testdataDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata")
}

func TestLoad_SimpleFile(t *testing.T) {
	root := testdataDir()
	tokenMap, err := load.Load(t.Context(), "simple.json", load.Options{
		Root: root,
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if tokenMap.Len() != 2 {
		t.Errorf("expected 2 tokens, got %d", tokenMap.Len())
	}

	// Check primary token
	primary, ok := tokenMap.Get("color-primary")
	if !ok {
		t.Fatal("expected to find color-primary")
	}
	if primary.Value != "#FF6B35" {
		t.Errorf("primary.Value = %q, want %q", primary.Value, "#FF6B35")
	}

	// Check secondary token (alias resolution)
	secondary, ok := tokenMap.Get("color-secondary")
	if !ok {
		t.Fatal("expected to find color-secondary")
	}
	if !secondary.IsResolved {
		t.Error("expected secondary to be resolved")
	}
}

func TestLoad_WithPrefix(t *testing.T) {
	root := testdataDir()
	tokenMap, err := load.Load(t.Context(), "simple.json", load.Options{
		Root:   root,
		Prefix: "rh",
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should find by short name
	tok, ok := tokenMap.Get("color-primary")
	if !ok {
		t.Fatal("expected to find token by short name")
	}
	if tok.Prefix != "rh" {
		t.Errorf("tok.Prefix = %q, want %q", tok.Prefix, "rh")
	}

	// Should also find by full CSS name
	tok2, ok := tokenMap.Get("--rh-color-primary")
	if !ok {
		t.Fatal("expected to find token by full CSS name")
	}
	if tok2.Value != "#FF6B35" {
		t.Errorf("tok2.Value = %q, want %q", tok2.Value, "#FF6B35")
	}
}

func TestLoad_WithSchemaVersion(t *testing.T) {
	root := testdataDir()
	tokenMap, err := load.Load(t.Context(), "simple.json", load.Options{
		Root:          root,
		SchemaVersion: schema.Draft,
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	tok, ok := tokenMap.Get("color-primary")
	if !ok {
		t.Fatal("expected to find token")
	}
	if tok.SchemaVersion != schema.Draft {
		t.Errorf("tok.SchemaVersion = %v, want %v", tok.SchemaVersion, schema.Draft)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	root := testdataDir()
	_, err := load.Load(t.Context(), "nonexistent.json", load.Options{
		Root: root,
	})
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	root := testdataDir()

	// Create an invalid JSON file for this test
	_, err := load.Load(t.Context(), "../load_test.go", load.Options{
		Root: root,
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// mockFetcher implements load.Fetcher for testing.
type mockFetcher struct {
	content []byte
	err     error
	called  bool
	url     string
}

func (m *mockFetcher) Fetch(ctx context.Context, url string) ([]byte, error) {
	m.called = true
	m.url = url
	if m.err != nil {
		return nil, m.err
	}
	return m.content, nil
}

func TestLoad_NetworkFallback(t *testing.T) {
	fetcher := &mockFetcher{content: cdnFallbackFixture}
	tokenMap, err := load.Load(t.Context(), "npm:@rhds/tokens/json/rhds.tokens.json", load.Options{
		Root:    testdataDir(),
		Fetcher: fetcher,
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !fetcher.called {
		t.Fatal("expected fetcher to be called")
	}
	if fetcher.url != "https://unpkg.com/@rhds/tokens/json/rhds.tokens.json" {
		t.Errorf("fetcher.url = %q, want unpkg URL", fetcher.url)
	}
	if tokenMap.Len() != 1 {
		t.Errorf("expected 1 token, got %d", tokenMap.Len())
	}
}

func TestLoad_LocalSuccessSkipsNetwork(t *testing.T) {
	fetcher := &mockFetcher{}
	_, err := load.Load(t.Context(), "simple.json", load.Options{
		Root:    testdataDir(),
		Fetcher: fetcher,
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if fetcher.called {
		t.Error("expected fetcher not to be called when local resolution succeeds")
	}
}

func TestLoad_NoFetcherPreservesError(t *testing.T) {
	_, err := load.Load(t.Context(), "npm:@nonexistent/pkg/tokens.json", load.Options{
		Root: testdataDir(),
	})
	if err == nil {
		t.Fatal("expected error when no fetcher and local resolution fails")
	}
}

func TestLoad_LocalSpecifierNeverTriggersNetwork(t *testing.T) {
	fetcher := &mockFetcher{}
	_, err := load.Load(t.Context(), "nonexistent.json", load.Options{
		Root:    testdataDir(),
		Fetcher: fetcher,
	})
	if err == nil {
		t.Fatal("expected error for nonexistent local file")
	}
	if fetcher.called {
		t.Error("expected fetcher not to be called for local specifier")
	}
}

func TestLoad_NetworkFallback_JSR(t *testing.T) {
	fetcher := &mockFetcher{content: cdnFallbackFixture}
	tokenMap, err := load.Load(t.Context(), "jsr:@scope/tokens/tokens.json", load.Options{
		Root:    testdataDir(),
		Fetcher: fetcher,
		CDN:     "esm.sh",
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !fetcher.called {
		t.Fatal("expected fetcher to be called")
	}
	if fetcher.url != "https://esm.sh/jsr/@scope/tokens/tokens.json" {
		t.Errorf("fetcher.url = %q, want esm.sh jsr URL", fetcher.url)
	}
	if tokenMap.Len() != 1 {
		t.Errorf("expected 1 token, got %d", tokenMap.Len())
	}
}

func TestLoad_NetworkFallback_JSR_UnsupportedCDN(t *testing.T) {
	fetcher := &mockFetcher{content: cdnFallbackFixture}
	_, err := load.Load(t.Context(), "jsr:@scope/tokens/tokens.json", load.Options{
		Root:    testdataDir(),
		Fetcher: fetcher,
		CDN:     "unpkg",
	})
	if err == nil {
		t.Fatal("expected error when jsr specifier uses CDN that doesn't support it")
	}
	// unpkg doesn't support jsr, so fetcher should not be called with a CDN URL.
	// The CDNURL returns false, so the original local error is returned.
}

func TestLoad_NetworkFallbackError(t *testing.T) {
	fetcher := &mockFetcher{err: fmt.Errorf("CDN unavailable")}
	_, err := load.Load(t.Context(), "npm:@rhds/tokens/json/rhds.tokens.json", load.Options{
		Root:    testdataDir(),
		Fetcher: fetcher,
	})
	if err == nil {
		t.Fatal("expected error when both local and network fail")
	}
	if !errors.Is(err, load.ErrLocalResolution) {
		t.Errorf("expected ErrLocalResolution in error chain, got: %v", err)
	}
	if !errors.Is(err, load.ErrNetworkFallback) {
		t.Errorf("expected ErrNetworkFallback in error chain, got: %v", err)
	}
}
