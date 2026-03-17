/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package snippets_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/convert/formatter/snippets"
	"bennypowers.dev/asimonim/parser"
	"bennypowers.dev/asimonim/resolver"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/testutil"
)

func TestFormat_Basic(t *testing.T) {
	runFixtureTest(t, "basic", snippets.Options{})
}

func TestFormat_WithPrefix(t *testing.T) {
	runFixtureTest(t, "with-prefix", snippets.Options{})
}

func TestFormat_LightDark(t *testing.T) {
	runFixtureTest(t, "light-dark", snippets.Options{})
}

func TestFormat_TextMate(t *testing.T) {
	runFixtureTest(t, "textmate", snippets.Options{Type: snippets.TypeTextMate})
}

func TestFormat_Zed(t *testing.T) {
	runFixtureTest(t, "zed", snippets.Options{Type: snippets.TypeZed})
}

func TestFormat_TextMateLightDark(t *testing.T) {
	runFixtureTest(t, "textmate-light-dark", snippets.Options{Type: snippets.TypeTextMate})
}

func TestFormat_ZedLightDark(t *testing.T) {
	runFixtureTest(t, "zed-light-dark", snippets.Options{Type: snippets.TypeZed})
}

func TestFormat_NonHexColors(t *testing.T) {
	runFixtureTest(t, "non-hex-colors", snippets.Options{})
}

func TestFormat_LightDarkNoRoot(t *testing.T) {
	runFixtureTest(t, "light-dark-no-root", snippets.Options{})
}

func TestNew_DefaultsToVSCode(t *testing.T) {
	runFixtureTestWithNew(t, "default-vscode")
}

func TestFormat_LightDarkEmptyFallback(t *testing.T) {
	runFixtureTest(t, "light-dark-empty", snippets.Options{})
}

func TestFormat_ZedDescription(t *testing.T) {
	runFixtureTest(t, "zed-description", snippets.Options{Type: snippets.TypeZed})
}

func TestFormat_StructuredColorPrefix(t *testing.T) {
	runFixtureTestWithSchema(t, "structured-color-prefix", snippets.Options{}, schema.V2025_10)
}

func TestFormat_LightDarkStructured(t *testing.T) {
	runFixtureTestWithSchema(t, "light-dark-structured", snippets.Options{}, schema.V2025_10)
}

// runFixtureTest runs a fixture-based test for the snippets formatter.
func runFixtureTest(t *testing.T, fixtureName string, snippetOpts snippets.Options) {
	t.Helper()
	runFixtureTestWithSchema(t, fixtureName, snippetOpts, schema.Draft)
}

// runFixtureTestWithSchema runs a fixture-based test for the snippets formatter with a specific schema.
func runFixtureTestWithSchema(t *testing.T, fixtureName string, snippetOpts snippets.Options, schemaVersion schema.Version) {
	t.Helper()

	fixturePath := filepath.Join("fixtures", fixtureName)
	mfs := testutil.NewFixtureFS(t, fixturePath, "/test")

	p := parser.NewJSONParser()
	tokens, err := p.ParseFile(mfs, "/test/tokens.json", parser.Options{
		SchemaVersion: schemaVersion,
		SkipPositions: true,
	})
	if err != nil {
		t.Fatalf("failed to parse tokens.json: %v", err)
	}

	if err := resolver.ResolveAliases(tokens, schemaVersion); err != nil {
		t.Fatalf("failed to resolve aliases: %v", err)
	}

	// Check for options.json to load options
	fmtOpts := formatter.Options{}
	if optData, err := mfs.ReadFile("/test/options.json"); err == nil {
		var fileOpts struct {
			Prefix string `json:"prefix"`
			Type   string `json:"type"`
		}
		if err := json.Unmarshal(optData, &fileOpts); err != nil {
			t.Fatalf("invalid options.json: %v", err)
		}
		if fileOpts.Prefix != "" {
			fmtOpts.Prefix = fileOpts.Prefix
		}
		if fileOpts.Type != "" {
			snippetOpts.Type = snippets.Type(fileOpts.Type)
		}
	}

	f := snippets.NewWithOptions(snippetOpts)
	result, err := f.Format(tokens, fmtOpts)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Determine expected file extension
	expectedExt := ".json"
	if snippetOpts.Type == snippets.TypeTextMate {
		expectedExt = ".plist"
	}
	goldenRelPath := filepath.Join(fixturePath, "expected"+expectedExt)

	// Update golden file if -update flag is set
	testutil.UpdateGoldenFile(t, goldenRelPath, result)

	expected := testutil.LoadFixtureFile(t, goldenRelPath)

	// Normalize line endings for comparison
	gotStr := strings.ReplaceAll(string(result), "\r\n", "\n")
	expectedStr := strings.ReplaceAll(string(expected), "\r\n", "\n")

	if gotStr != expectedStr {
		t.Errorf("output mismatch for fixture %q.\n\nGot:\n%s\n\nExpected:\n%s", fixtureName, gotStr, expectedStr)
	}
}

// runFixtureTestWithNew tests the New() constructor (default VSCode format).
func runFixtureTestWithNew(t *testing.T, fixtureName string) {
	t.Helper()

	fixturePath := filepath.Join("fixtures", fixtureName)
	mfs := testutil.NewFixtureFS(t, fixturePath, "/test")

	p := parser.NewJSONParser()
	tokens, err := p.ParseFile(mfs, "/test/tokens.json", parser.Options{
		SchemaVersion: schema.Draft,
		SkipPositions: true,
	})
	if err != nil {
		t.Fatalf("failed to parse tokens.json: %v", err)
	}

	if err := resolver.ResolveAliases(tokens, schema.Draft); err != nil {
		t.Fatalf("failed to resolve aliases: %v", err)
	}

	f := snippets.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// VSCode format includes "scope" field
	if !strings.Contains(string(result), `"scope"`) {
		t.Errorf("expected VSCode format with scope field, got:\n%s", result)
	}

	goldenRelPath := filepath.Join(fixturePath, "expected.json")
	testutil.UpdateGoldenFile(t, goldenRelPath, result)

	expected := testutil.LoadFixtureFile(t, goldenRelPath)
	gotStr := strings.ReplaceAll(string(result), "\r\n", "\n")
	expectedStr := strings.ReplaceAll(string(expected), "\r\n", "\n")
	if gotStr != expectedStr {
		t.Errorf("output mismatch for fixture %q.\n\nGot:\n%s\n\nExpected:\n%s", fixtureName, gotStr, expectedStr)
	}
}
