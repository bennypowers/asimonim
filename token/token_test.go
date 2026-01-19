/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package token_test

import (
	"testing"

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
