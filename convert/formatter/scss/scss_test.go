/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package scss_test

import (
	"strings"
	"testing"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/convert/formatter/scss"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/token"
)

func TestFormat_V2025_10_StructuredColors(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:          "color.srgb-hex",
			Path:          []string{"color", "srgb-hex"},
			Type:          token.TypeColor,
			SchemaVersion: schema.V2025_10,
			RawValue: map[string]any{
				"colorSpace": "srgb",
				"components": []any{1.0, 0.42, 0.21},
				"alpha":      1.0,
				"hex":        "#FF6B36",
			},
		},
		{
			Name:          "color.oklch",
			Path:          []string{"color", "oklch"},
			Type:          token.TypeColor,
			SchemaVersion: schema.V2025_10,
			RawValue: map[string]any{
				"colorSpace": "oklch",
				"components": []any{0.7, 0.15, 180.0},
				"alpha":      1.0,
			},
		},
		{
			Name:          "color.oklch-alpha",
			Path:          []string{"color", "oklch-alpha"},
			Type:          token.TypeColor,
			SchemaVersion: schema.V2025_10,
			RawValue: map[string]any{
				"colorSpace": "oklch",
				"components": []any{0.7, 0.15, 180.0},
				"alpha":      0.8,
			},
		},
		{
			Name:          "color.display-p3",
			Path:          []string{"color", "display-p3"},
			Type:          token.TypeColor,
			SchemaVersion: schema.V2025_10,
			RawValue: map[string]any{
				"colorSpace": "display-p3",
				"components": []any{1.0, 0.5, 0.25},
				"alpha":      1.0,
			},
		},
		{
			Name:          "color.hsl",
			Path:          []string{"color", "hsl"},
			Type:          token.TypeColor,
			SchemaVersion: schema.V2025_10,
			RawValue: map[string]any{
				"colorSpace": "hsl",
				"components": []any{210.0, 50.0, 60.0},
				"alpha":      1.0,
			},
		},
		{
			Name:          "color.srgb-no-hex",
			Path:          []string{"color", "srgb-no-hex"},
			Type:          token.TypeColor,
			SchemaVersion: schema.V2025_10,
			RawValue: map[string]any{
				"colorSpace": "srgb",
				"components": []any{1.0, 0.5, 0.25},
				"alpha":      1.0,
			},
		},
		{
			Name:          "color.none-component",
			Path:          []string{"color", "none-component"},
			Type:          token.TypeColor,
			SchemaVersion: schema.V2025_10,
			RawValue: map[string]any{
				"colorSpace": "oklch",
				"components": []any{0.5, "none", 180.0},
				"alpha":      1.0,
			},
		},
	}

	f := scss.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	expectations := map[string]string{
		"$color-srgb-hex":        "#FF6B36",
		"$color-oklch":           "oklch(0.7 0.15 180)",
		"$color-oklch-alpha":     "oklch(0.7 0.15 180 / 0.8)",
		"$color-display-p3":      "color(display-p3 1 0.5 0.25)",
		"$color-hsl":             "hsl(210 50 60)",
		"$color-srgb-no-hex":     "#FF8040",
		"$color-none-component":  "oklch(0.5 none 180)",
	}

	for varName, expectedValue := range expectations {
		expected := varName + ": " + expectedValue + ";"
		if !strings.Contains(output, expected) {
			t.Errorf("expected output to contain %q, got:\n%s", expected, output)
		}
	}

	// Ensure no Go map literals in output
	if strings.Contains(output, "map[") {
		t.Errorf("output contains Go map literal:\n%s", output)
	}
}

func TestFormat_V2025_10_StructuredDimensions(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:          "spacing.small",
			Path:          []string{"spacing", "small"},
			Type:          token.TypeDimension,
			SchemaVersion: schema.V2025_10,
			RawValue:      map[string]any{"value": 4.0, "unit": "px"},
		},
		{
			Name:          "spacing.medium",
			Path:          []string{"spacing", "medium"},
			Type:          token.TypeDimension,
			SchemaVersion: schema.V2025_10,
			RawValue:      map[string]any{"value": 1.5, "unit": "rem"},
		},
	}

	f := scss.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	if !strings.Contains(output, "$spacing-small: 4px;") {
		t.Errorf("expected $spacing-small: 4px;, got:\n%s", output)
	}
	if !strings.Contains(output, "$spacing-medium: 1.5rem;") {
		t.Errorf("expected $spacing-medium: 1.5rem;, got:\n%s", output)
	}
}

// Regression test: nil dimension value should produce JSON fallback, not "nilpx"
func TestFormat_DimensionNilValue(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:          "spacing.broken",
			Path:          []string{"spacing", "broken"},
			Type:          token.TypeDimension,
			SchemaVersion: schema.V2025_10,
			RawValue:      map[string]any{"value": nil, "unit": "px"},
		},
	}

	f := scss.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)
	if strings.Contains(output, "nilpx") || strings.Contains(output, "<nil>px") {
		t.Errorf("nil dimension value produced invalid SCSS: %s", output)
	}
	// Should contain JSON fallback serialization
	if !strings.Contains(output, `{"unit":"px","value":null}`) {
		t.Errorf("expected JSON fallback for nil dimension, got:\n%s", output)
	}
}
