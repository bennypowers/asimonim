/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package token_test

import (
	"testing"

	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/token"
)

func TestToken_CSSVariableName(t *testing.T) {
	tests := []struct {
		name     string
		token    token.Token
		expected string
	}{
		{
			name:     "simple name",
			token:    token.Token{Name: "color-primary"},
			expected: "--color-primary",
		},
		{
			name:     "dotted name",
			token:    token.Token{Name: "color.primary"},
			expected: "--color-primary",
		},
		{
			name:     "with prefix",
			token:    token.Token{Name: "color-primary", Prefix: "rh"},
			expected: "--rh-color-primary",
		},
		{
			name:     "with dotted prefix",
			token:    token.Token{Name: "color-primary", Prefix: "my.prefix"},
			expected: "--my-prefix-color-primary",
		},
		{
			name:     "empty name",
			token:    token.Token{Name: ""},
			expected: "",
		},
		{
			name:     "complex path",
			token:    token.Token{Name: "color.brand.primary.base"},
			expected: "--color-brand-primary-base",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.CSSVariableName(); got != tt.expected {
				t.Errorf("Token.CSSVariableName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestToken_DotPath(t *testing.T) {
	tests := []struct {
		name     string
		path     []string
		expected string
	}{
		{
			name:     "simple path",
			path:     []string{"color", "primary"},
			expected: "color.primary",
		},
		{
			name:     "single element",
			path:     []string{"color"},
			expected: "color",
		},
		{
			name:     "empty path",
			path:     nil,
			expected: "",
		},
		{
			name:     "deep path",
			path:     []string{"color", "brand", "primary", "base"},
			expected: "color.brand.primary.base",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := token.Token{Path: tt.path}
			if got := tok.DotPath(); got != tt.expected {
				t.Errorf("Token.DotPath() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestToken_DisplayValue(t *testing.T) {
	tests := []struct {
		name     string
		token    token.Token
		expected string
	}{
		{
			name: "simple string value",
			token: token.Token{
				Value: "#ff0000",
				Type:  token.TypeColor,
			},
			expected: "#ff0000",
		},
		{
			name: "raw value string takes precedence over Value",
			token: token.Token{
				Value:    "{color.primary}",
				RawValue: "#00ff00",
				Type:     token.TypeColor,
			},
			expected: "#00ff00",
		},
		{
			name: "resolved value takes precedence",
			token: token.Token{
				Value:         "{color.primary}",
				RawValue:      nil,
				ResolvedValue: "#0000ff",
				IsResolved:    true,
				Type:          token.TypeColor,
			},
			expected: "#0000ff",
		},
		{
			name: "structured color with hex field",
			token: token.Token{
				Type: token.TypeColor,
				RawValue: map[string]any{
					"colorSpace": "srgb",
					"components": []any{1.0, 0.42, 0.21},
					"alpha":      1.0,
					"hex":        "#FF6B36",
				},
				SchemaVersion: schema.V2025_10,
			},
			expected: "#FF6B36",
		},
		{
			name: "structured color without hex field",
			token: token.Token{
				Type: token.TypeColor,
				RawValue: map[string]any{
					"colorSpace": "srgb",
					"components": []any{1.0, 0.5, 0.25},
				},
				SchemaVersion: schema.V2025_10,
			},
			expected: "color(srgb 1 0.5 0.25)",
		},
		{
			name: "resolved structured color",
			token: token.Token{
				Type:  token.TypeColor,
				Value: "{color.brand.primary}",
				ResolvedValue: map[string]any{
					"colorSpace": "srgb",
					"components": []any{0.0, 0.4, 0.8},
					"hex":        "#0066CC",
				},
				IsResolved:    true,
				SchemaVersion: schema.V2025_10,
			},
			expected: "#0066CC",
		},
		{
			name: "non-color map value (JSON serialized)",
			token: token.Token{
				Type: token.TypeTypography,
				RawValue: map[string]any{
					"fontFamily": "Arial",
					"fontSize":   "16px",
				},
			},
			expected: `{"fontFamily":"Arial","fontSize":"16px"}`,
		},
		{
			name: "array value without type (JSON serialized)",
			token: token.Token{
				Type:     "", // no type, falls back to JSON serialization
				RawValue: []any{0.42, 0.0, 0.58, 1.0},
			},
			expected: `[0.42,0,0.58,1]`,
		},
		{
			name: "empty value",
			token: token.Token{
				Value: "",
				Type:  token.TypeString,
			},
			expected: "",
		},
		{
			name: "nil raw value falls back to Value",
			token: token.Token{
				Value:    "fallback-value",
				RawValue: nil,
				Type:     token.TypeString,
			},
			expected: "fallback-value",
		},
		{
			name: "number value",
			token: token.Token{
				Type:     token.TypeNumber,
				RawValue: 42.5,
			},
			expected: "42.5",
		},
		// Dimension tests
		{
			name: "structured dimension with rem",
			token: token.Token{
				Type: token.TypeDimension,
				RawValue: map[string]any{
					"value": 0.5,
					"unit":  "rem",
				},
				SchemaVersion: schema.V2025_10,
			},
			expected: "0.5rem",
		},
		{
			name: "structured dimension with px",
			token: token.Token{
				Type: token.TypeDimension,
				RawValue: map[string]any{
					"value": 16,
					"unit":  "px",
				},
				SchemaVersion: schema.V2025_10,
			},
			expected: "16px",
		},
		{
			name: "structured dimension with zero",
			token: token.Token{
				Type: token.TypeDimension,
				RawValue: map[string]any{
					"value": 0,
					"unit":  "px",
				},
				SchemaVersion: schema.V2025_10,
			},
			expected: "0px",
		},
		// Duration tests
		{
			name: "structured duration with ms",
			token: token.Token{
				Type: token.TypeDuration,
				RawValue: map[string]any{
					"value": 100,
					"unit":  "ms",
				},
				SchemaVersion: schema.V2025_10,
			},
			expected: "100ms",
		},
		{
			name: "structured duration with seconds",
			token: token.Token{
				Type: token.TypeDuration,
				RawValue: map[string]any{
					"value": 1.5,
					"unit":  "s",
				},
				SchemaVersion: schema.V2025_10,
			},
			expected: "1.5s",
		},
		// Cubic bezier tests
		{
			name: "cubic bezier array",
			token: token.Token{
				Type:     token.TypeCubicBezier,
				RawValue: []any{0.42, 0.0, 0.58, 1.0},
			},
			expected: "cubic-bezier(0.42, 0, 0.58, 1)",
		},
		{
			name: "cubic bezier with negative y",
			token: token.Token{
				Type:     token.TypeCubicBezier,
				RawValue: []any{0.5, -0.5, 0.5, 1.5},
			},
			expected: "cubic-bezier(0.5, -0.5, 0.5, 1.5)",
		},
		// Font family tests
		{
			name: "font family single string",
			token: token.Token{
				Type:     token.TypeFontFamily,
				RawValue: "Arial",
			},
			expected: "Arial",
		},
		{
			name: "font family array",
			token: token.Token{
				Type:     token.TypeFontFamily,
				RawValue: []any{"Helvetica", "Arial", "sans-serif"},
			},
			expected: "Helvetica, Arial, sans-serif",
		},
		{
			name: "font family with spaces gets quoted",
			token: token.Token{
				Type:     token.TypeFontFamily,
				RawValue: []any{"Comic Sans MS", "Arial"},
			},
			expected: `"Comic Sans MS", Arial`,
		},
		// Shadow tests
		{
			name: "shadow with string values",
			token: token.Token{
				Type: token.TypeShadow,
				RawValue: map[string]any{
					"offsetX": "2px",
					"offsetY": "4px",
					"blur":    "8px",
					"spread":  "0px",
					"color":   "rgba(0, 0, 0, 0.2)",
				},
			},
			expected: "2px 4px 8px rgba(0, 0, 0, 0.2)",
		},
		{
			name: "shadow with structured dimension values",
			token: token.Token{
				Type: token.TypeShadow,
				RawValue: map[string]any{
					"offsetX": map[string]any{"value": 0, "unit": "px"},
					"offsetY": map[string]any{"value": 4, "unit": "px"},
					"blur":    map[string]any{"value": 16, "unit": "px"},
					"spread":  map[string]any{"value": 2, "unit": "px"},
					"color":   "#000000",
				},
				SchemaVersion: schema.V2025_10,
			},
			expected: "0px 4px 16px 2px #000000",
		},
		{
			name: "shadow array (layered shadows)",
			token: token.Token{
				Type: token.TypeShadow,
				RawValue: []any{
					map[string]any{
						"offsetX": "0px",
						"offsetY": "2px",
						"blur":    "4px",
						"color":   "rgba(0,0,0,0.1)",
					},
					map[string]any{
						"offsetX": "0px",
						"offsetY": "4px",
						"blur":    "8px",
						"color":   "rgba(0,0,0,0.2)",
					},
				},
			},
			expected: "0px 2px 4px rgba(0,0,0,0.1), 0px 4px 8px rgba(0,0,0,0.2)",
		},
		// Border tests
		{
			name: "border with string values",
			token: token.Token{
				Type: token.TypeBorder,
				RawValue: map[string]any{
					"width": "1px",
					"style": "solid",
					"color": "#000000",
				},
			},
			expected: "1px solid #000000",
		},
		{
			name: "border with structured dimension",
			token: token.Token{
				Type: token.TypeBorder,
				RawValue: map[string]any{
					"width": map[string]any{"value": 2, "unit": "px"},
					"style": "dashed",
					"color": "#ff0000",
				},
				SchemaVersion: schema.V2025_10,
			},
			expected: "2px dashed #ff0000",
		},
		// Transition tests
		{
			name: "transition with string values",
			token: token.Token{
				Type: token.TypeTransition,
				RawValue: map[string]any{
					"duration":       "300ms",
					"timingFunction": "ease-in-out",
				},
			},
			expected: "300ms ease-in-out",
		},
		{
			name: "transition with structured values",
			token: token.Token{
				Type: token.TypeTransition,
				RawValue: map[string]any{
					"duration":       map[string]any{"value": 200, "unit": "ms"},
					"timingFunction": []any{0.4, 0.0, 0.2, 1.0},
					"delay":          map[string]any{"value": 50, "unit": "ms"},
				},
				SchemaVersion: schema.V2025_10,
			},
			expected: "200ms cubic-bezier(0.4, 0, 0.2, 1) 50ms",
		},
		{
			name: "transition without delay",
			token: token.Token{
				Type: token.TypeTransition,
				RawValue: map[string]any{
					"duration":       map[string]any{"value": 150, "unit": "ms"},
					"timingFunction": []any{0.0, 0.0, 1.0, 1.0},
				},
				SchemaVersion: schema.V2025_10,
			},
			expected: "150ms cubic-bezier(0, 0, 1, 1)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.token.DisplayValue()
			if got != tt.expected {
				t.Errorf("Token.DisplayValue() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestTokenTypeConstants(t *testing.T) {
	// Verify type constants are defined correctly
	types := map[string]string{
		"TypeColor":       token.TypeColor,
		"TypeDimension":   token.TypeDimension,
		"TypeFontFamily":  token.TypeFontFamily,
		"TypeFontWeight":  token.TypeFontWeight,
		"TypeDuration":    token.TypeDuration,
		"TypeCubicBezier": token.TypeCubicBezier,
		"TypeNumber":      token.TypeNumber,
		"TypeString":      token.TypeString,
		"TypeStrokeStyle": token.TypeStrokeStyle,
		"TypeBorder":      token.TypeBorder,
		"TypeTransition":  token.TypeTransition,
		"TypeShadow":      token.TypeShadow,
		"TypeGradient":    token.TypeGradient,
		"TypeTypography":  token.TypeTypography,
	}

	expected := map[string]string{
		"TypeColor":       "color",
		"TypeDimension":   "dimension",
		"TypeFontFamily":  "fontFamily",
		"TypeFontWeight":  "fontWeight",
		"TypeDuration":    "duration",
		"TypeCubicBezier": "cubicBezier",
		"TypeNumber":      "number",
		"TypeString":      "string",
		"TypeStrokeStyle": "strokeStyle",
		"TypeBorder":      "border",
		"TypeTransition":  "transition",
		"TypeShadow":      "shadow",
		"TypeGradient":    "gradient",
		"TypeTypography":  "typography",
	}

	for name, got := range types {
		if want := expected[name]; got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}
