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

func TestFormat_V2025_10_Colors(t *testing.T) {
	runFixtureTestV2025(t, "v2025-10-colors", css.Options{})
}

// Regression test for issue #17: exact scenario from the bug report.
func TestFormat_V2025_10_Issue17(t *testing.T) {
	runFixtureTestV2025(t, "v2025-10-issue-17", css.Options{})
}

// runFixtureTest runs a fixture-based test for the CSS formatter using draft schema.
func runFixtureTest(t *testing.T, fixtureName string, cssOpts css.Options) {
	t.Helper()
	runFixtureTestWithSchema(t, fixtureName, cssOpts, schema.Draft)
}

// runFixtureTestV2025 runs a fixture-based test for the CSS formatter using v2025.10 schema.
func runFixtureTestV2025(t *testing.T, fixtureName string, cssOpts css.Options) {
	t.Helper()
	runFixtureTestWithSchema(t, fixtureName, cssOpts, schema.V2025_10)
}

// runFixtureTestWithSchema runs a fixture-based test for the CSS formatter.
func runFixtureTestWithSchema(t *testing.T, fixtureName string, cssOpts css.Options, schemaVersion schema.Version) {
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

func TestToCSSValue_StructuredColor(t *testing.T) {
	tests := []struct {
		name     string
		value    map[string]any
		expected string
	}{
		{
			name: "srgb with hex",
			value: map[string]any{
				"colorSpace": "srgb",
				"components": []any{1.0, 0.42, 0.21},
				"alpha":      1.0,
				"hex":        "#FF6B36",
			},
			expected: "#FF6B36",
		},
		{
			name: "srgb without hex converts to hex",
			value: map[string]any{
				"colorSpace": "srgb",
				"components": []any{1.0, 0.5, 0.25},
				"alpha":      1.0,
			},
			expected: "#FF8040",
		},
		{
			name: "srgb with alpha uses color function",
			value: map[string]any{
				"colorSpace": "srgb",
				"components": []any{1.0, 0.5, 0.25},
				"alpha":      0.5,
			},
			expected: "color(srgb 1 0.5 0.25 / 0.5)",
		},
		{
			name: "oklch",
			value: map[string]any{
				"colorSpace": "oklch",
				"components": []any{0.988281, 0.0046875, 20.0},
				"alpha":      1.0,
			},
			expected: "oklch(0.9883 0.004687 20)",
		},
		{
			name: "oklch with alpha",
			value: map[string]any{
				"colorSpace": "oklch",
				"components": []any{0.7, 0.15, 180.0},
				"alpha":      0.8,
			},
			expected: "oklch(0.7 0.15 180 / 0.8)",
		},
		{
			name: "oklab",
			value: map[string]any{
				"colorSpace": "oklab",
				"components": []any{0.5, 0.1, -0.1},
				"alpha":      1.0,
			},
			expected: "oklab(0.5 0.1 -0.1)",
		},
		{
			name: "hsl",
			value: map[string]any{
				"colorSpace": "hsl",
				"components": []any{210.0, 50.0, 60.0},
				"alpha":      1.0,
			},
			expected: "hsl(210 50 60)",
		},
		{
			name: "hwb",
			value: map[string]any{
				"colorSpace": "hwb",
				"components": []any{210.0, 20.0, 30.0},
				"alpha":      1.0,
			},
			expected: "hwb(210 20 30)",
		},
		{
			name: "lab",
			value: map[string]any{
				"colorSpace": "lab",
				"components": []any{50.0, 20.0, -30.0},
				"alpha":      1.0,
			},
			expected: "lab(50 20 -30)",
		},
		{
			name: "lch",
			value: map[string]any{
				"colorSpace": "lch",
				"components": []any{50.0, 30.0, 270.0},
				"alpha":      1.0,
			},
			expected: "lch(50 30 270)",
		},
		{
			name: "display-p3",
			value: map[string]any{
				"colorSpace": "display-p3",
				"components": []any{1.0, 0.5, 0.25},
				"alpha":      1.0,
			},
			expected: "color(display-p3 1 0.5 0.25)",
		},
		{
			name: "display-p3 with alpha",
			value: map[string]any{
				"colorSpace": "display-p3",
				"components": []any{1.0, 0.5, 0.25},
				"alpha":      0.75,
			},
			expected: "color(display-p3 1 0.5 0.25 / 0.75)",
		},
		{
			name: "a98-rgb",
			value: map[string]any{
				"colorSpace": "a98-rgb",
				"components": []any{0.8, 0.4, 0.2},
				"alpha":      1.0,
			},
			expected: "color(a98-rgb 0.8 0.4 0.2)",
		},
		{
			name: "prophoto-rgb",
			value: map[string]any{
				"colorSpace": "prophoto-rgb",
				"components": []any{0.9, 0.5, 0.3},
				"alpha":      1.0,
			},
			expected: "color(prophoto-rgb 0.9 0.5 0.3)",
		},
		{
			name: "rec2020",
			value: map[string]any{
				"colorSpace": "rec2020",
				"components": []any{0.7, 0.4, 0.2},
				"alpha":      1.0,
			},
			expected: "color(rec2020 0.7 0.4 0.2)",
		},
		{
			name: "xyz-d50",
			value: map[string]any{
				"colorSpace": "xyz-d50",
				"components": []any{0.4, 0.3, 0.2},
				"alpha":      1.0,
			},
			expected: "color(xyz-d50 0.4 0.3 0.2)",
		},
		{
			name: "xyz-d65",
			value: map[string]any{
				"colorSpace": "xyz-d65",
				"components": []any{0.4, 0.3, 0.2},
				"alpha":      1.0,
			},
			expected: "color(xyz-d65 0.4 0.3 0.2)",
		},
		{
			name: "srgb-linear",
			value: map[string]any{
				"colorSpace": "srgb-linear",
				"components": []any{0.5, 0.3, 0.1},
				"alpha":      1.0,
			},
			expected: "color(srgb-linear 0.5 0.3 0.1)",
		},
		{
			name: "none component",
			value: map[string]any{
				"colorSpace": "oklch",
				"components": []any{0.5, "none", 180.0},
				"alpha":      1.0,
			},
			expected: "oklch(0.5 none 180)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := css.ToCSSValue(token.TypeColor, tt.value)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestToCSSValue_StructuredDimension(t *testing.T) {
	tests := []struct {
		name     string
		value    map[string]any
		expected string
	}{
		{
			name:     "px dimension",
			value:    map[string]any{"value": 4.0, "unit": "px"},
			expected: "4px",
		},
		{
			name:     "rem dimension",
			value:    map[string]any{"value": 1.5, "unit": "rem"},
			expected: "1.5rem",
		},
		{
			name:     "em dimension",
			value:    map[string]any{"value": 2.0, "unit": "em"},
			expected: "2em",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := css.ToCSSValue(token.TypeDimension, tt.value)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestToCSSValue_StringColor(t *testing.T) {
	result := css.ToCSSValue(token.TypeColor, "#FF6B35")
	if result != "#FF6B35" {
		t.Errorf("expected \"#FF6B35\", got %q", result)
	}
}

func TestToCSSValue_StringDimension(t *testing.T) {
	result := css.ToCSSValue(token.TypeDimension, "16px")
	if result != "16px" {
		t.Errorf("expected \"16px\", got %q", result)
	}
}

func TestToCSSValue_DimensionNilValue(t *testing.T) {
	// CodeRabbit review: nil value in dimension map should not produce "nilpx"
	value := map[string]any{"value": nil, "unit": "px"}
	result := css.ToCSSValue(token.TypeDimension, value)
	if result == "nilpx" || result == "<nil>px" {
		t.Errorf("nil dimension value produced invalid CSS: %q", result)
	}
}

func TestToCSSValue_DimensionMissingUnit(t *testing.T) {
	// Structured dimension without unit should fall through gracefully
	value := map[string]any{"value": 4.0}
	result := css.ToCSSValue(token.TypeDimension, value)
	if strings.Contains(result, "map[") {
		t.Errorf("dimension without unit rendered as Go map literal: %q", result)
	}
}
