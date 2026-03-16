/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package swift_test

import (
	"strings"
	"testing"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/convert/formatter/swift"
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
	}

	f := swift.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// Swift should convert structured colors to Color() initializers
	expectations := []string{
		"Color(.sRGB, red: 1, green: 0.42, blue: 0.21)",      // srgb
		"Color(.sRGB, red: 0.7, green: 0.15, blue: 180)",     // oklch falls back to sRGB
		"Color(.sRGB, red: 0.7, green: 0.15, blue: 180, opacity: 0.8)", // oklch with alpha
		"Color(.displayP3, red: 1, green: 0.5, blue: 0.25)",  // display-p3
	}

	for _, expected := range expectations {
		if !strings.Contains(output, expected) {
			t.Errorf("expected output to contain %q, got:\n%s", expected, output)
		}
	}

	// Ensure no Go map literals in output
	if strings.Contains(output, "map[") {
		t.Errorf("output contains Go map literal:\n%s", output)
	}
}

// Regression test: nil dimension value should not produce invalid Swift
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

	f := swift.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)
	if strings.Contains(output, "CGFloat(<nil>)") || strings.Contains(output, "CGFloat(nil)") {
		t.Errorf("nil dimension value produced invalid Swift: %s", output)
	}
}

// Regression test: Swift comment injection via unit string
func TestFormat_DimensionCommentInjection(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:          "spacing.evil",
			Path:          []string{"spacing", "evil"},
			Type:          token.TypeDimension,
			SchemaVersion: schema.V2025_10,
			RawValue:      map[string]any{"value": 16.0, "unit": "px */ inject /*"},
		},
	}

	f := swift.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)
	// The */ in the unit should be sanitized to prevent comment breakout
	if strings.Count(output, "*/") > 1 {
		t.Errorf("unit string broke out of Swift comment: %s", output)
	}

	// Also test /* injection
	tokens[0].RawValue = map[string]any{"value": 16.0, "unit": "/* bad"}
	result, err = f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	output = string(result)
	if strings.Count(output, "/*") > 1 {
		t.Errorf("unit string opened an extra block comment: %s", output)
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

	f := swift.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	if !strings.Contains(output, "CGFloat(4)") {
		t.Errorf("expected CGFloat(4) for px dimension, got:\n%s", output)
	}
	if !strings.Contains(output, "CGFloat(1.5)") {
		t.Errorf("expected CGFloat(1.5) for rem dimension, got:\n%s", output)
	}

	// Ensure no Go map literals in output
	if strings.Contains(output, "map[") {
		t.Errorf("output contains Go map literal:\n%s", output)
	}
}
