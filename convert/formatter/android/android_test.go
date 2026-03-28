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
	"bennypowers.dev/asimonim/testutil"
	"bennypowers.dev/asimonim/token"
)

func TestFormat_V2025_10_StructuredColors(t *testing.T) {
	allTokens := testutil.ParseFixtureTokens(t, "fixtures/v2025_10/all-color-spaces", schema.V2025_10)

	// Select representative color tokens for Android hex conversion tests
	tokens := []*token.Token{
		testutil.TokenByPath(t, allTokens, "color.srgb-hex"),       // srgb, hex: "#FF6B36"
		testutil.TokenByPath(t, allTokens, "color.srgb-no-hex"),    // srgb, [1, 0.5, 0.25] → #FF8040
		testutil.TokenByPath(t, allTokens, "color.srgb-alpha"),     // srgb, [1, 0.5, 0.25], alpha: 0.5
		testutil.TokenByPath(t, allTokens, "color.oklch"),          // oklch, [0.988281, 0.0046875, 20]
		testutil.TokenByPath(t, allTokens, "color.display-p3"),     // display-p3, [1, 0.5, 0.25]
		testutil.TokenByPath(t, allTokens, "color.a98-rgb"),        // a98-rgb, [0.8, 0.4, 0.2]
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

	// Assert exact hex values for each color space conversion
	for _, tc := range []struct {
		name string
		hex  string
	}{
		{"srgb-hex", "#FF6B36"},        // srgb [1, 0.42, 0.21] hex "#FF6B36"
		{"srgb-no-hex", "#FF8040"},     // srgb [1, 0.5, 0.25] → #FF8040
		{"srgb-alpha", "#80FF8040"},    // srgb [1, 0.5, 0.25] alpha 0.5 → #AARRGGBB
		{"display-p3", "#FF7626"},      // display-p3 [1, 0.5, 0.25] → sRGB
		{"a98-rgb", "#E7662B"},         // a98-rgb [0.8, 0.4, 0.2] → sRGB
	} {
		if !strings.Contains(output, tc.hex) {
			t.Errorf("expected %s for %s, got:\n%s", tc.hex, tc.name, output)
		}
	}

	if strings.Contains(output, "map[") {
		t.Errorf("output contains Go map literal:\n%s", output)
	}
}

func TestFormat_V2025_10_StructuredDimensions(t *testing.T) {
	allTokens := testutil.ParseFixtureTokens(t, "fixtures/v2025_10/all-color-spaces", schema.V2025_10)

	tokens := []*token.Token{
		testutil.TokenByPath(t, allTokens, "spacing.small"),  // {value: 4, unit: "px"}
		testutil.TokenByPath(t, allTokens, "spacing.medium"), // {value: 1.5, unit: "rem"}
	}

	f := android.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// spacing.small: {value: 4, unit: "px"} → 4px
	if !strings.Contains(output, "4px") {
		t.Errorf("expected 4px for structured dimension, got:\n%s", output)
	}

	// spacing.medium: {value: 1.5, unit: "rem"} → 1.5rem
	if !strings.Contains(output, "1.5rem") {
		t.Errorf("expected 1.5rem for structured dimension, got:\n%s", output)
	}

	if strings.Contains(output, "map[") {
		t.Errorf("output contains Go map literal:\n%s", output)
	}
}

func TestFormat_StringAndNumberTokens(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:     "label.greeting",
			Path:     []string{"label", "greeting"},
			Type:     token.TypeString,
			RawValue: "Hello World",
		},
		{
			Name:     "size.scale",
			Path:     []string{"size", "scale"},
			Type:     token.TypeNumber,
			RawValue: float64(42),
		},
		{
			Name:     "font.body",
			Path:     []string{"font", "body"},
			Type:     token.TypeFontFamily,
			RawValue: "Helvetica Neue",
		},
	}

	f := android.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// string type → <string> element
	if !strings.Contains(output, `<string name="label_greeting">Hello World</string>`) {
		t.Errorf("expected <string> element for string token, got:\n%s", output)
	}
	// number type → <integer> element
	if !strings.Contains(output, `<integer name="size_scale">42</integer>`) {
		t.Errorf("expected <integer> element for number token, got:\n%s", output)
	}
	// fontFamily type → <string> element
	if !strings.Contains(output, `<string name="font_body">Helvetica Neue</string>`) {
		t.Errorf("expected <string> element for fontFamily token, got:\n%s", output)
	}
}

func TestFormat_UnknownTokenType(t *testing.T) {
	// Tokens with unknown type should default to <string> element
	tokens := []*token.Token{
		{
			Name:     "custom.value",
			Path:     []string{"custom", "value"},
			Type:     "customType",
			RawValue: "some-value",
		},
	}

	f := android.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	if !strings.Contains(output, `<string name="custom_value">some-value</string>`) {
		t.Errorf("expected <string> element for unknown token type, got:\n%s", output)
	}
}

func TestFormat_WideGamutColorSpaces(t *testing.T) {
	allTokens := testutil.ParseFixtureTokens(t, "fixtures/v2025_10/all-color-spaces", schema.V2025_10)

	tokens := []*token.Token{
		testutil.TokenByPath(t, allTokens, "color.xyz-d50"),      // xyz-d50 [0.4, 0.3, 0.2]
		testutil.TokenByPath(t, allTokens, "color.prophoto-rgb"), // prophoto-rgb [0.9, 0.5, 0.3]
		testutil.TokenByPath(t, allTokens, "color.rec2020"),      // rec2020 [0.7, 0.4, 0.2]
		testutil.TokenByPath(t, allTokens, "color.xyz-d65"),      // xyz-d65 [0.4, 0.3, 0.2]
		testutil.TokenByPath(t, allTokens, "color.srgb-linear"),  // srgb-linear [0.5, 0.3, 0.1]
	}

	f := android.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// Assert exact hex values for each wide-gamut color space
	for _, tc := range []struct {
		name string
		hex  string
	}{
		{"xyz-d50", "#D67987"},         // xyz-d50 [0.4, 0.3, 0.2] → sRGB
		{"prophoto-rgb", "#FF7151"},    // prophoto-rgb [0.9, 0.5, 0.3] → sRGB
		{"rec2020", "#DC6735"},         // rec2020 [0.7, 0.4, 0.2] → sRGB
		{"xyz-d65", "#DF7773"},         // xyz-d65 [0.4, 0.3, 0.2] → sRGB
		{"srgb-linear", "#BC9559"},     // srgb-linear [0.5, 0.3, 0.1] → sRGB
	} {
		if !strings.Contains(output, tc.hex) {
			t.Errorf("expected %s for %s, got:\n%s", tc.hex, tc.name, output)
		}
	}

	// No CSS functions allowed
	if strings.Contains(output, "color(") || strings.Contains(output, "oklch(") {
		t.Errorf("Android output contains CSS color functions:\n%s", output)
	}

	// No Go map literals
	if strings.Contains(output, "map[") {
		t.Errorf("output contains Go map literal:\n%s", output)
	}
}

func TestFormat_DimensionNilValue(t *testing.T) {
	// Nil dimension value should produce JSON fallback
	tokens := []*token.Token{
		{
			Name:          "spacing.broken",
			Path:          []string{"spacing", "broken"},
			Type:          token.TypeDimension,
			SchemaVersion: schema.V2025_10,
			RawValue:      map[string]any{"value": nil, "unit": "px"},
		},
	}

	f := android.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// Should not produce "nilpx" or "<nil>px"
	if strings.Contains(output, "nilpx") || strings.Contains(output, "<nil>px") {
		t.Errorf("nil dimension value produced invalid output: %s", output)
	}
}

func TestFormat_MapAndSliceValues(t *testing.T) {
	// Map values for non-color/dimension types should serialize as JSON
	tokens := []*token.Token{
		{
			Name:     "shadow.base",
			Path:     []string{"shadow", "base"},
			Type:     token.TypeShadow,
			RawValue: map[string]any{"offsetX": "2px", "offsetY": "4px"},
		},
		{
			Name:     "bezier.ease",
			Path:     []string{"bezier", "ease"},
			Type:     token.TypeCubicBezier,
			RawValue: []any{0.25, 0.1, 0.25, 1.0},
		},
	}

	f := android.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// map value → JSON serialization, no "map[" Go literal
	if strings.Contains(output, "map[") {
		t.Errorf("output contains Go map literal:\n%s", output)
	}
	// slice value → JSON array
	if !strings.Contains(output, "[0.25,0.1,0.25,1]") {
		t.Errorf("expected JSON array for cubic bezier, got:\n%s", output)
	}
}

func TestFormat_WithPrefix(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:     "color.primary",
			Path:     []string{"color", "primary"},
			Type:     token.TypeColor,
			RawValue: "#ff0000",
		},
	}

	f := android.New()
	result, err := f.Format(tokens, formatter.Options{Prefix: "app"})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// Prefix should be applied with underscore delimiter
	if !strings.Contains(output, `name="app_color_primary"`) {
		t.Errorf("expected prefixed name, got:\n%s", output)
	}
}

func TestFormat_WithHeader(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:     "color.primary",
			Path:     []string{"color", "primary"},
			Type:     token.TypeColor,
			RawValue: "#ff0000",
		},
	}

	f := android.New()
	result, err := f.Format(tokens, formatter.Options{Header: "Custom header"})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	if !strings.Contains(output, "Custom header") {
		t.Errorf("expected custom header in output, got:\n%s", output)
	}
}

func TestFormat_LowValueColorSpaces(t *testing.T) {
	// Colors with very low or negative component values exercise edge cases
	// in go-colorful's transfer functions and clamping
	allTokens := testutil.ParseFixtureTokens(t, "fixtures/low-value-colors", schema.V2025_10)

	tokens := []*token.Token{
		testutil.TokenByPath(t, allTokens, "color.srgb-low"),      // srgb [0.01, 0.02, 0.03]
		testutil.TokenByPath(t, allTokens, "color.a98-negative"),  // a98-rgb [-0.1, 0.01, 0.5]
		testutil.TokenByPath(t, allTokens, "color.prophoto-low"),  // prophoto-rgb [0.02, 0.01, 0.03]
		testutil.TokenByPath(t, allTokens, "color.rec2020-low"),   // rec2020 [0.01, 0.02, 0.03]
	}

	f := android.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// Assert exact hex values for low-value/edge-case colors
	for _, tc := range []struct {
		name string
		hex  string
	}{
		{"srgb-low", "#030508"},       // srgb [0.01, 0.02, 0.03] → near-black
		{"a98-negative", "#000083"},   // a98-rgb [-0.1, 0.01, 0.5] → clamped negative
		{"prophoto-low", "#050207"},   // prophoto-rgb [0.02, 0.01, 0.03] → near-black
		{"rec2020-low", "#020F14"},    // rec2020 [0.01, 0.02, 0.03] → near-black
	} {
		if !strings.Contains(output, tc.hex) {
			t.Errorf("expected %s for %s, got:\n%s", tc.hex, tc.name, output)
		}
	}
}
