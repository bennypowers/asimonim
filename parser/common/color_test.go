/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package common_test

import (
	"testing"

	"bennypowers.dev/asimonim/parser/common"
	"bennypowers.dev/asimonim/schema"
)

func TestParseColorValue_Draft(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
		wantErr  bool
	}{
		{
			name:     "hex color",
			input:    "#FF6B36",
			expected: "#FF6B36",
		},
		{
			name:     "rgb color",
			input:    "rgb(255, 107, 54)",
			expected: "rgb(255, 107, 54)",
		},
		{
			name:     "hsl color",
			input:    "hsl(120, 50%, 50%)",
			expected: "hsl(120, 50%, 50%)",
		},
		{
			name:     "named color",
			input:    "rebeccapurple",
			expected: "rebeccapurple",
		},
		{
			name:    "non-string value fails",
			input:   42,
			wantErr: true,
		},
		{
			name:    "map value fails for draft",
			input:   map[string]any{"colorSpace": "srgb"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := common.ParseColorValue(tt.input, schema.Draft)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ToCSS() != tt.expected {
				t.Errorf("ToCSS() = %q, want %q", result.ToCSS(), tt.expected)
			}
			if result.Version() != schema.Draft {
				t.Errorf("Version() = %v, want %v", result.Version(), schema.Draft)
			}
			if !result.IsValid() {
				t.Errorf("IsValid() = false, want true")
			}
		})
	}
}

func TestParseColorValue_V2025_10(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected string
		wantErr  bool
	}{
		{
			name: "srgb with hex field",
			input: map[string]any{
				"colorSpace": "srgb",
				"components": []any{1.0, 0.42, 0.21},
				"hex":        "#FF6B36",
			},
			expected: "#FF6B36",
		},
		{
			name: "srgb without hex field",
			input: map[string]any{
				"colorSpace": "srgb",
				"components": []any{1.0, 0.5, 0.25},
			},
			expected: "color(srgb 1 0.5 0.25)",
		},
		{
			name: "srgb with alpha",
			input: map[string]any{
				"colorSpace": "srgb",
				"components": []any{1.0, 0.5, 0.25},
				"alpha":      0.5,
			},
			expected: "color(srgb 1 0.5 0.25 / 0.5)",
		},
		{
			name: "srgb with full alpha (omitted)",
			input: map[string]any{
				"colorSpace": "srgb",
				"components": []any{1.0, 0.5, 0.25},
				"alpha":      1.0,
			},
			expected: "color(srgb 1 0.5 0.25)",
		},
		{
			name: "display-p3 color",
			input: map[string]any{
				"colorSpace": "display-p3",
				"components": []any{0.8, 0.2, 0.4},
			},
			expected: "color(display-p3 0.8 0.2 0.4)",
		},
		{
			name: "oklch color",
			input: map[string]any{
				"colorSpace": "oklch",
				"components": []any{0.7, 0.15, 180.0},
			},
			expected: "color(oklch 0.7 0.15 180)",
		},
		{
			name: "oklab color",
			input: map[string]any{
				"colorSpace": "oklab",
				"components": []any{0.6, 0.1, -0.1},
			},
			expected: "color(oklab 0.6 0.1 -0.1)",
		},
		{
			name: "lab color",
			input: map[string]any{
				"colorSpace": "lab",
				"components": []any{50.0, 25.0, -25.0},
			},
			expected: "color(lab 50 25 -25)",
		},
		{
			name: "lch color",
			input: map[string]any{
				"colorSpace": "lch",
				"components": []any{50.0, 30.0, 270.0},
			},
			expected: "color(lch 50 30 270)",
		},
		{
			name: "a98-rgb color",
			input: map[string]any{
				"colorSpace": "a98-rgb",
				"components": []any{0.5, 0.5, 0.5},
			},
			expected: "color(a98-rgb 0.5 0.5 0.5)",
		},
		{
			name: "prophoto-rgb color",
			input: map[string]any{
				"colorSpace": "prophoto-rgb",
				"components": []any{0.4, 0.3, 0.2},
			},
			expected: "color(prophoto-rgb 0.4 0.3 0.2)",
		},
		{
			name: "rec2020 color",
			input: map[string]any{
				"colorSpace": "rec2020",
				"components": []any{0.3, 0.6, 0.9},
			},
			expected: "color(rec2020 0.3 0.6 0.9)",
		},
		{
			name: "xyz-d50 color",
			input: map[string]any{
				"colorSpace": "xyz-d50",
				"components": []any{0.4, 0.5, 0.6},
			},
			expected: "color(xyz-d50 0.4 0.5 0.6)",
		},
		{
			name: "xyz-d65 color",
			input: map[string]any{
				"colorSpace": "xyz-d65",
				"components": []any{0.2, 0.3, 0.4},
			},
			expected: "color(xyz-d65 0.2 0.3 0.4)",
		},
		{
			name: "srgb-linear color",
			input: map[string]any{
				"colorSpace": "srgb-linear",
				"components": []any{0.5, 0.5, 0.5},
			},
			expected: "color(srgb-linear 0.5 0.5 0.5)",
		},
		{
			name: "hsl uses native function",
			input: map[string]any{
				"colorSpace": "hsl",
				"components": []any{120.0, 0.5, 0.5},
			},
			expected: "hsl(120 0.5 0.5)",
		},
		{
			name: "hsl with alpha",
			input: map[string]any{
				"colorSpace": "hsl",
				"components": []any{120.0, 0.5, 0.5},
				"alpha":      0.8,
			},
			expected: "hsl(120 0.5 0.5 / 0.8)",
		},
		{
			name: "hwb uses native function",
			input: map[string]any{
				"colorSpace": "hwb",
				"components": []any{180.0, 0.2, 0.3},
			},
			expected: "hwb(180 0.2 0.3)",
		},
		{
			name: "hwb with alpha",
			input: map[string]any{
				"colorSpace": "hwb",
				"components": []any{180.0, 0.2, 0.3},
				"alpha":      0.5,
			},
			expected: "hwb(180 0.2 0.3 / 0.5)",
		},
		{
			name: "component with none keyword",
			input: map[string]any{
				"colorSpace": "oklch",
				"components": []any{0.7, "none", 180.0},
			},
			expected: "color(oklch 0.7 none 180)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := common.ParseColorValue(tt.input, schema.V2025_10)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ToCSS() != tt.expected {
				t.Errorf("ToCSS() = %q, want %q", result.ToCSS(), tt.expected)
			}
			if result.Version() != schema.V2025_10 {
				t.Errorf("Version() = %v, want %v", result.Version(), schema.V2025_10)
			}
			if !result.IsValid() {
				t.Errorf("IsValid() = false, want true")
			}
		})
	}
}

func TestParseColorValue_V2025_10_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{
			name:  "string value fails",
			input: "#FF0000",
		},
		{
			name: "missing colorSpace",
			input: map[string]any{
				"components": []any{1.0, 0.5, 0.25},
			},
		},
		{
			name: "missing components",
			input: map[string]any{
				"colorSpace": "srgb",
			},
		},
		{
			name: "invalid component type",
			input: map[string]any{
				"colorSpace": "srgb",
				"components": []any{1.0, "invalid", 0.25},
			},
		},
		{
			name: "components not array",
			input: map[string]any{
				"colorSpace": "srgb",
				"components": "not-an-array",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := common.ParseColorValue(tt.input, schema.V2025_10)
			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

func TestValidColorSpaces(t *testing.T) {
	expectedSpaces := []string{
		"srgb", "srgb-linear", "display-p3", "a98-rgb",
		"prophoto-rgb", "rec2020", "xyz-d50", "xyz-d65",
		"lab", "lch", "oklab", "oklch", "hsl", "hwb",
	}

	for _, space := range expectedSpaces {
		if !common.ValidColorSpaces[space] {
			t.Errorf("color space %q not in ValidColorSpaces", space)
		}
	}

	// Verify count
	if len(common.ValidColorSpaces) != 14 {
		t.Errorf("expected 14 color spaces, got %d", len(common.ValidColorSpaces))
	}
}

func TestStringColorValue_IsValid(t *testing.T) {
	valid := &common.StringColorValue{Value: "#FF0000", Schema: schema.Draft}
	if !valid.IsValid() {
		t.Error("expected IsValid() = true for non-empty value")
	}

	invalid := &common.StringColorValue{Value: "", Schema: schema.Draft}
	if invalid.IsValid() {
		t.Error("expected IsValid() = false for empty value")
	}
}

func TestObjectColorValue_IsValid(t *testing.T) {
	valid := &common.ObjectColorValue{
		ColorSpace: "srgb",
		Components: []any{1.0, 0.5, 0.25},
		Schema:     schema.V2025_10,
	}
	if !valid.IsValid() {
		t.Error("expected IsValid() = true for valid color")
	}

	noSpace := &common.ObjectColorValue{
		ColorSpace: "",
		Components: []any{1.0, 0.5, 0.25},
		Schema:     schema.V2025_10,
	}
	if noSpace.IsValid() {
		t.Error("expected IsValid() = false for empty colorSpace")
	}

	noComponents := &common.ObjectColorValue{
		ColorSpace: "srgb",
		Components: nil,
		Schema:     schema.V2025_10,
	}
	if noComponents.IsValid() {
		t.Error("expected IsValid() = false for empty components")
	}
}
