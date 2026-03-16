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
	}

	f := android.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// Android XML should contain proper color values
	if !strings.Contains(output, "#FF6B36") {
		t.Errorf("expected #FF6B36 for srgb-hex, got:\n%s", output)
	}
	if !strings.Contains(output, "oklch(0.7 0.15 180)") {
		t.Errorf("expected oklch CSS value for oklch color, got:\n%s", output)
	}
	if !strings.Contains(output, "color(display-p3 1 0.5 0.25)") {
		t.Errorf("expected color() function for display-p3, got:\n%s", output)
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

	// Ensure no Go map literals in output
	if strings.Contains(output, "map[") {
		t.Errorf("output contains Go map literal:\n%s", output)
	}
}
