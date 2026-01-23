/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package typescriptmap_test

import (
	"strings"
	"testing"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/convert/formatter/typescriptmap"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/token"
)

func TestFormat_Basic(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:        "color-primary",
			Path:        []string{"color", "primary"},
			Type:        token.TypeColor,
			Value:       "#FF6B35",
			Description: "Primary brand color",
		},
		{
			Name:  "spacing-small",
			Path:  []string{"spacing", "small"},
			Type:  token.TypeDimension,
			Value: "4px",
		},
	}

	f := typescriptmap.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// Check type definitions
	if !strings.Contains(output, "export interface Color") {
		t.Error("expected Color interface definition")
	}
	if !strings.Contains(output, "export interface DesignToken<V>") {
		t.Error("expected DesignToken interface definition")
	}

	// Check TokenName union type
	if !strings.Contains(output, "export type TokenName =") {
		t.Error("expected TokenName union type")
	}
	if !strings.Contains(output, `| "color-primary"`) {
		t.Error("expected color-primary in TokenName")
	}
	if !strings.Contains(output, `| "spacing-small"`) {
		t.Error("expected spacing-small in TokenName")
	}

	// Check TokenMap class
	if !strings.Contains(output, "export class TokenMap") {
		t.Error("expected TokenMap class")
	}
	if !strings.Contains(output, `get(name: "color-primary"): DesignToken<string>`) {
		t.Error("expected typed get() overload for color-primary")
	}
	if !strings.Contains(output, `get(name: "spacing-small"): DesignToken<string>`) {
		t.Error("expected typed get() overload for spacing-small")
	}

	// Check default export
	if !strings.Contains(output, "export const tokens = new TokenMap()") {
		t.Error("expected default tokens export")
	}
}

func TestFormat_WithPrefix(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:  "color-primary",
			Path:  []string{"color", "primary"},
			Type:  token.TypeColor,
			Value: "#FF6B35",
		},
	}

	f := typescriptmap.New()
	result, err := f.Format(tokens, formatter.Options{Prefix: "rh"})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// Check prefix is applied
	if !strings.Contains(output, `| "rh-color-primary"`) {
		t.Error("expected prefix in TokenName")
	}
	if !strings.Contains(output, `get(name: "rh-color-primary")`) {
		t.Error("expected prefix in get() overload")
	}
}

func TestFormat_StructuredColor(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:          "color-brand",
			Path:          []string{"color", "brand"},
			Type:          token.TypeColor,
			SchemaVersion: schema.V2025_10,
			ResolvedValue: map[string]any{
				"colorSpace": "srgb",
				"components": []any{1.0, 0.42, 0.21},
				"alpha":      1.0,
				"hex":        "#FF6B36",
			},
		},
	}

	f := typescriptmap.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// Structured color should have Color type
	if !strings.Contains(output, `get(name: "color-brand"): DesignToken<Color>`) {
		t.Error("expected Color type for structured color")
	}
	// Value should include colorSpace
	if !strings.Contains(output, `"colorSpace": "srgb"`) {
		t.Error("expected colorSpace in value")
	}
}

func TestFormat_JSDoc(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:        "color-primary",
			Path:        []string{"color", "primary"},
			Type:        token.TypeColor,
			Value:       "#FF6B35",
			Description: "Primary brand color",
		},
	}

	f := typescriptmap.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// Check JSDoc comment
	if !strings.Contains(output, "/** Primary brand color */") {
		t.Error("expected JSDoc comment for token with description")
	}
}

func TestFormat_NumberType(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:          "weight-bold",
			Path:          []string{"weight", "bold"},
			Type:          token.TypeFontWeight,
			ResolvedValue: 700.0,
		},
	}

	f := typescriptmap.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// Number types should have number type
	if !strings.Contains(output, `get(name: "weight-bold"): DesignToken<number>`) {
		t.Error("expected number type for fontWeight")
	}
}

func TestFormat_CubicBezier(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:          "easing-smooth",
			Path:          []string{"easing", "smooth"},
			Type:          token.TypeCubicBezier,
			ResolvedValue: []any{0.25, 0.1, 0.25, 1.0},
		},
	}

	f := typescriptmap.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// CubicBezier should have tuple type
	if !strings.Contains(output, `get(name: "easing-smooth"): DesignToken<[number, number, number, number]>`) {
		t.Error("expected tuple type for cubicBezier")
	}
}
