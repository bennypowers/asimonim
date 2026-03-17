/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package android_test

import (
	"strings"
	"testing"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/convert/formatter/android"
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
			Name:          "color.srgb-alpha",
			Path:          []string{"color", "srgb-alpha"},
			Type:          token.TypeColor,
			SchemaVersion: schema.V2025_10,
			RawValue: map[string]any{
				"colorSpace": "srgb",
				"components": []any{1.0, 0.0, 0.0},
				"alpha":      0.5,
			},
		},
	}

	f := android.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// All colors must be hex — no CSS functions allowed in Android XML
	if strings.Contains(output, "oklch(") || strings.Contains(output, "color(") {
		t.Errorf("Android output contains CSS color functions:\n%s", output)
	}

	// sRGB with hex field should use it
	if !strings.Contains(output, "#FF6B36") {
		t.Errorf("expected #FF6B36 for srgb-hex, got:\n%s", output)
	}

	// sRGB without hex should convert to hex
	if !strings.Contains(output, "#FF8040") {
		t.Errorf("expected #FF8040 for srgb-no-hex, got:\n%s", output)
	}

	// All values must be hex format
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "<color") && strings.Contains(line, ">") {
			// Extract value between > and <
			start := strings.Index(line, ">") + 1
			end := strings.LastIndex(line, "<")
			if start > 0 && end > start {
				val := line[start:end]
				if !strings.HasPrefix(val, "#") {
					t.Errorf("non-hex color value in Android XML: %q", val)
				}
			}
		}
	}

	// Alpha should produce #AARRGGBB format
	if !strings.Contains(output, "#80FF0000") {
		t.Errorf("expected #80FF0000 for srgb with alpha 0.5, got:\n%s", output)
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
	}

	f := android.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	if !strings.Contains(output, "4px") {
		t.Errorf("expected 4px for structured dimension, got:\n%s", output)
	}

	if strings.Contains(output, "map[") {
		t.Errorf("output contains Go map literal:\n%s", output)
	}
}
