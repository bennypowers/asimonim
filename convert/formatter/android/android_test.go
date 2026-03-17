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

	// sRGB with hex field should use it: srgb [1, 0.42, 0.21] hex "#FF6B36"
	if !strings.Contains(output, "#FF6B36") {
		t.Errorf("expected #FF6B36 for srgb-hex, got:\n%s", output)
	}

	// sRGB without hex should convert to hex: srgb [1, 0.5, 0.25] → #FF8040
	if !strings.Contains(output, "#FF8040") {
		t.Errorf("expected #FF8040 for srgb-no-hex, got:\n%s", output)
	}

	// All values must be hex format
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "<color") && strings.Contains(line, ">") {
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

	// Alpha should produce #AARRGGBB: srgb [1, 0.5, 0.25] alpha 0.5
	if !strings.Contains(output, "#80FF8040") {
		t.Errorf("expected #80FF8040 for srgb-alpha, got:\n%s", output)
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

	// All must be hex format
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "<color") && strings.Contains(line, ">") {
			start := strings.Index(line, ">") + 1
			end := strings.LastIndex(line, "<")
			if start > 0 && end > start {
				val := line[start:end]
				if !strings.HasPrefix(val, "#") {
					t.Errorf("non-hex color value in Android XML: %q in line: %s", val, line)
				}
			}
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
	// Colors with very low component values exercise the linear transfer
	// function branches (srgbToLinear c <= 0.04045, prophotoToLinear c <= 16/512,
	// a98ToLinear c < 0, rec2020ToLinear c < beta*4.5)
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

	// All must produce hex output (no CSS functions)
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "<color") && strings.Contains(line, ">") {
			start := strings.Index(line, ">") + 1
			end := strings.LastIndex(line, "<")
			if start > 0 && end > start {
				val := line[start:end]
				// low value colors → near-black hex values
				if !strings.HasPrefix(val, "#") {
					t.Errorf("non-hex color value: %q in line: %s", val, line)
				}
			}
		}
	}
}
