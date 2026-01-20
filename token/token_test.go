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

func TestNewMap(t *testing.T) {
	tokens := []*token.Token{
		{Name: "color-primary", Value: "#FF0000"},
		{Name: "color-secondary", Value: "#00FF00"},
		{Name: "spacing-small", Value: "8px"},
	}

	t.Run("creates map with correct length", func(t *testing.T) {
		m := token.NewMap(tokens, "")
		if m.Len() != 3 {
			t.Errorf("Map.Len() = %d, want 3", m.Len())
		}
	})

	t.Run("applies prefix to tokens", func(t *testing.T) {
		m := token.NewMap(tokens, "my-prefix")
		tok, ok := m.Get("color-primary")
		if !ok {
			t.Fatal("expected to find token")
		}
		if tok.Prefix != "my-prefix" {
			t.Errorf("token.Prefix = %q, want %q", tok.Prefix, "my-prefix")
		}
	})

	t.Run("does not modify original tokens", func(t *testing.T) {
		_ = token.NewMap(tokens, "my-prefix")
		if tokens[0].Prefix != "" {
			t.Errorf("original token was modified, Prefix = %q", tokens[0].Prefix)
		}
	})
}

func TestMap_Get(t *testing.T) {
	tokens := []*token.Token{
		{Name: "color-primary", Value: "#FF0000"},
		{Name: "color-secondary", Value: "#00FF00"},
	}

	t.Run("lookup by short name without prefix", func(t *testing.T) {
		m := token.NewMap(tokens, "")
		tok, ok := m.Get("color-primary")
		if !ok {
			t.Fatal("expected to find token")
		}
		if tok.Value != "#FF0000" {
			t.Errorf("tok.Value = %q, want %q", tok.Value, "#FF0000")
		}
	})

	t.Run("lookup by full CSS name without prefix", func(t *testing.T) {
		m := token.NewMap(tokens, "")
		tok, ok := m.Get("--color-primary")
		if !ok {
			t.Fatal("expected to find token")
		}
		if tok.Value != "#FF0000" {
			t.Errorf("tok.Value = %q, want %q", tok.Value, "#FF0000")
		}
	})

	t.Run("lookup by short name with prefix", func(t *testing.T) {
		m := token.NewMap(tokens, "rh")
		tok, ok := m.Get("color-primary")
		if !ok {
			t.Fatal("expected to find token by short name")
		}
		if tok.Value != "#FF0000" {
			t.Errorf("tok.Value = %q, want %q", tok.Value, "#FF0000")
		}
	})

	t.Run("lookup by full CSS name with prefix", func(t *testing.T) {
		m := token.NewMap(tokens, "rh")
		tok, ok := m.Get("--rh-color-primary")
		if !ok {
			t.Fatal("expected to find token by full CSS name")
		}
		if tok.Value != "#FF0000" {
			t.Errorf("tok.Value = %q, want %q", tok.Value, "#FF0000")
		}
	})

	t.Run("lookup by dot-path", func(t *testing.T) {
		tokensWithPath := []*token.Token{
			{Name: "color-brand-primary", Value: "#FF0000"},
		}
		m := token.NewMap(tokensWithPath, "")
		tok, ok := m.Get("color.brand.primary")
		if !ok {
			t.Fatal("expected to find token by dot-path")
		}
		if tok.Value != "#FF0000" {
			t.Errorf("tok.Value = %q, want %q", tok.Value, "#FF0000")
		}
	})

	t.Run("returns false for missing token", func(t *testing.T) {
		m := token.NewMap(tokens, "")
		_, ok := m.Get("nonexistent")
		if ok {
			t.Error("expected not to find nonexistent token")
		}
	})
}

func TestMap_All(t *testing.T) {
	tokens := []*token.Token{
		{Name: "a", Value: "1"},
		{Name: "b", Value: "2"},
		{Name: "c", Value: "3"},
	}
	m := token.NewMap(tokens, "")
	all := m.All()

	if len(all) != 3 {
		t.Errorf("len(All()) = %d, want 3", len(all))
	}

	// Verify all tokens are present (order not guaranteed)
	values := make(map[string]bool)
	for _, tok := range all {
		values[tok.Value] = true
	}
	for _, expected := range []string{"1", "2", "3"} {
		if !values[expected] {
			t.Errorf("missing token with value %q", expected)
		}
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

func TestTypeToCSSSyntax(t *testing.T) {
	tests := []struct {
		tokenType string
		expected  string
	}{
		{token.TypeColor, "<color>"},
		{token.TypeDimension, "<length>"},
		{token.TypeNumber, "<number>"},
		{token.TypeString, "<custom-ident>"},
		{token.TypeFontFamily, "<custom-ident>+"},
		{token.TypeFontWeight, "<number>"},
		{token.TypeDuration, "<time>"},
		{token.TypeCubicBezier, "<easing-function>"},
		{token.TypeShadow, "<shadow>"},
		{token.TypeBorder, "<line-width> || <line-style> || <color>"},
		{token.TypeGradient, "<image>"},
		{token.TypeTypography, "<custom-ident>"},
		{token.TypeStrokeStyle, "<line-style>"},
		{token.TypeTransition, "<time> || <easing-function>"},
		{"unknownType", "<custom-ident>"},
		{"", "<custom-ident>"},
	}

	for _, tt := range tests {
		t.Run(tt.tokenType, func(t *testing.T) {
			if got := token.TypeToCSSSyntax(tt.tokenType); got != tt.expected {
				t.Errorf("TypeToCSSSyntax(%q) = %q, want %q", tt.tokenType, got, tt.expected)
			}
		})
	}
}

func TestToken_CSSSyntax(t *testing.T) {
	tests := []struct {
		name     string
		token    token.Token
		expected string
	}{
		{
			name:     "color token",
			token:    token.Token{Type: token.TypeColor},
			expected: "<color>",
		},
		{
			name:     "dimension token",
			token:    token.Token{Type: token.TypeDimension},
			expected: "<length>",
		},
		{
			name:     "empty type",
			token:    token.Token{Type: ""},
			expected: "<custom-ident>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.CSSSyntax(); got != tt.expected {
				t.Errorf("Token.CSSSyntax() = %q, want %q", got, tt.expected)
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
			expected: "#FF8040", // sRGB auto-converts to hex
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
