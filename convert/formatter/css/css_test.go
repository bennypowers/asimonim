/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package css_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/convert/formatter/css"
	"bennypowers.dev/asimonim/parser"
	"bennypowers.dev/asimonim/resolver"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/testutil"
	"bennypowers.dev/asimonim/token"
)

func TestFormat_Plain(t *testing.T) {
	runFixtureTest(t, "plain", css.Options{})
}

func TestFormat_WithPrefix(t *testing.T) {
	runFixtureTest(t, "with-prefix", css.Options{})
}

func TestFormat_HostSelector(t *testing.T) {
	runFixtureTest(t, "host-selector", css.Options{Selector: css.SelectorHost})
}

func TestFormat_LitModule(t *testing.T) {
	runFixtureTest(t, "lit-module", css.Options{
		Selector: css.SelectorHost,
		Module:   css.ModuleLit,
	})
}

func TestFormat_LitModuleWithRoot(t *testing.T) {
	runFixtureTest(t, "lit-with-root", css.Options{
		Selector: css.SelectorRoot,
		Module:   css.ModuleLit,
	})
}

// runFixtureTest runs a fixture-based test for the CSS formatter.
func runFixtureTest(t *testing.T, fixtureName string, cssOpts css.Options) {
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

	// Check for options.json to load options
	fmtOpts := formatter.Options{}
	if optData, err := mfs.ReadFile("/test/options.json"); err == nil {
		var fileOpts struct {
			Prefix      string `json:"prefix"`
			Delimiter   string `json:"delimiter"`
			CSSSelector string `json:"cssSelector"`
			CSSModule   string `json:"cssModule"`
		}
		if err := json.Unmarshal(optData, &fileOpts); err == nil {
			if fileOpts.Prefix != "" {
				fmtOpts.Prefix = fileOpts.Prefix
			}
			if fileOpts.Delimiter != "" {
				fmtOpts.Delimiter = fileOpts.Delimiter
			}
			if fileOpts.CSSSelector != "" {
				cssOpts.Selector = css.Selector(fileOpts.CSSSelector)
			}
			if fileOpts.CSSModule != "" {
				cssOpts.Module = css.Module(fileOpts.CSSModule)
			}
		}
	}

	f := css.NewWithOptions(cssOpts)
	result, err := f.Format(tokens, fmtOpts)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Determine expected file extension
	expectedExt := ".css"
	if cssOpts.Module == css.ModuleLit {
		expectedExt = ".ts"
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

// Unit tests for ToCSSValue function

func TestToCSSValue_CubicBezier(t *testing.T) {
	value := []any{0.25, 0.1, 0.25, 1.0}
	result := css.ToCSSValue(token.TypeCubicBezier, value)

	expected := "cubic-bezier(0.25, 0.1, 0.25, 1)"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestToCSSValue_FontFamily(t *testing.T) {
	result := css.ToCSSValue(token.TypeFontFamily, "Open Sans")
	expected := `"Open Sans"`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}

	// Already quoted
	result = css.ToCSSValue(token.TypeFontFamily, `"Roboto"`)
	expected = `"Roboto"`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestToCSSValue_Number(t *testing.T) {
	// Integer-like float
	result := css.ToCSSValue(token.TypeNumber, 400.0)
	if result != "400" {
		t.Errorf("expected \"400\", got %q", result)
	}

	// Actual float
	result = css.ToCSSValue(token.TypeNumber, 1.5)
	if result != "1.5" {
		t.Errorf("expected \"1.5\", got %q", result)
	}
}

func TestToCSSValue_Duration(t *testing.T) {
	// Milliseconds
	result := css.ToCSSValue("", "200ms")
	if result != "200ms" {
		t.Errorf("expected \"200ms\", got %q", result)
	}

	// Seconds
	result = css.ToCSSValue("", "0.5s")
	if result != "0.5s" {
		t.Errorf("expected \"0.5s\", got %q", result)
	}
}
