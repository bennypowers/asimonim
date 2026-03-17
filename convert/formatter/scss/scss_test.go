/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package scss_test

import (
	"strings"
	"testing"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/convert/formatter/scss"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/testutil"
	"bennypowers.dev/asimonim/token"
)

func TestFormat_V2025_10_StructuredColors(t *testing.T) {
	allTokens := testutil.ParseFixtureTokens(t, "fixtures/v2025_10/all-color-spaces", schema.V2025_10)

	tokens := []*token.Token{
		testutil.TokenByPath(t, allTokens, "color.srgb-hex"),       // srgb, hex: "#FF6B36"
		testutil.TokenByPath(t, allTokens, "color.srgb-no-hex"),    // srgb, [1, 0.5, 0.25]
		testutil.TokenByPath(t, allTokens, "color.oklch-alpha"),    // oklch, [0.7, 0.15, 180], alpha: 0.8
		testutil.TokenByPath(t, allTokens, "color.display-p3"),     // display-p3, [1, 0.5, 0.25]
		testutil.TokenByPath(t, allTokens, "color.hsl"),            // hsl, [210, 50, 60]
		testutil.TokenByPath(t, allTokens, "color.none-component"), // oklch, [0.5, "none", 180]
	}

	f := scss.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	expectations := map[string]string{
		"$color-srgb-hex":       "#FF6B36",                       // srgb [1, 0.42, 0.21] hex "#FF6B36"
		"$color-srgb-no-hex":    "#FF8040",                       // srgb [1, 0.5, 0.25] → hex
		"$color-oklch-alpha":    "oklch(0.7 0.15 180 / 0.8)",     // oklch [0.7, 0.15, 180] alpha 0.8
		"$color-display-p3":     "color(display-p3 1 0.5 0.25)",  // display-p3 [1, 0.5, 0.25]
		"$color-hsl":            "hsl(210 50 60)",                 // hsl [210, 50, 60]
		"$color-none-component": "oklch(0.5 none 180)",           // oklch [0.5, "none", 180]
	}

	for varName, expectedValue := range expectations {
		expected := varName + ": " + expectedValue + ";"
		if !strings.Contains(output, expected) {
			t.Errorf("expected output to contain %q, got:\n%s", expected, output)
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

	f := scss.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// spacing.small: {value: 4, unit: "px"} → 4px
	if !strings.Contains(output, "$spacing-small: 4px;") {
		t.Errorf("expected $spacing-small: 4px;, got:\n%s", output)
	}
	// spacing.medium: {value: 1.5, unit: "rem"} → 1.5rem
	if !strings.Contains(output, "$spacing-medium: 1.5rem;") {
		t.Errorf("expected $spacing-medium: 1.5rem;, got:\n%s", output)
	}
}

// Regression test: nil dimension value should produce JSON fallback, not "nilpx"
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

	f := scss.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)
	if strings.Contains(output, "nilpx") || strings.Contains(output, "<nil>px") {
		t.Errorf("nil dimension value produced invalid SCSS: %s", output)
	}
	// Should contain JSON fallback serialization
	if !strings.Contains(output, `{"unit":"px","value":null}`) {
		t.Errorf("expected JSON fallback for nil dimension, got:\n%s", output)
	}
}

func TestFormat_StringColorValue(t *testing.T) {
	// Draft-style string color values should pass through as-is
	tokens := []*token.Token{
		{
			Name:          "color.primary",
			Path:          []string{"color", "primary"},
			Type:          token.TypeColor,
			SchemaVersion: schema.Draft,
			RawValue:      "#ff0000",
		},
		{
			Name:          "color.named",
			Path:          []string{"color", "named"},
			Type:          token.TypeColor,
			SchemaVersion: schema.Draft,
			RawValue:      "rebeccapurple",
		},
	}

	f := scss.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// String colors pass through via fmt.Sprintf("%v", value)
	if !strings.Contains(output, "$color-primary: #ff0000;") {
		t.Errorf("expected $color-primary: #ff0000;, got:\n%s", output)
	}
	if !strings.Contains(output, "$color-named: rebeccapurple;") {
		t.Errorf("expected $color-named: rebeccapurple;, got:\n%s", output)
	}
}

func TestFormat_FontFamilyQuoting(t *testing.T) {
	// FontFamily string values should be quoted
	tokens := []*token.Token{
		{
			Name:     "font.body",
			Path:     []string{"font", "body"},
			Type:     token.TypeFontFamily,
			RawValue: "Helvetica Neue",
		},
		{
			Name:     "font.mono",
			Path:     []string{"font", "mono"},
			Type:     token.TypeFontFamily,
			RawValue: "monospace",
		},
	}

	f := scss.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// FontFamily values should be quoted
	if !strings.Contains(output, `$font-body: "Helvetica Neue";`) {
		t.Errorf("expected quoted font family, got:\n%s", output)
	}
	if !strings.Contains(output, `$font-mono: "monospace";`) {
		t.Errorf("expected quoted mono font family, got:\n%s", output)
	}
}

func TestFormat_DurationPatternMatching(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:     "timing.fast",
			Path:     []string{"timing", "fast"},
			Type:     token.TypeDuration,
			RawValue: "200ms",
		},
		{
			Name:     "timing.slow",
			Path:     []string{"timing", "slow"},
			Type:     token.TypeDuration,
			RawValue: "1.5s",
		},
		{
			Name:     "timing.zero",
			Path:     []string{"timing", "zero"},
			Type:     token.TypeDuration,
			RawValue: "0s",
		},
	}

	f := scss.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// ms duration passes through the string suffix check
	if !strings.Contains(output, "$timing-fast: 200ms;") {
		t.Errorf("expected $timing-fast: 200ms;, got:\n%s", output)
	}
	// seconds duration matches secondsDurationPattern
	if !strings.Contains(output, "$timing-slow: 1.5s;") {
		t.Errorf("expected $timing-slow: 1.5s;, got:\n%s", output)
	}
	// zero seconds matches secondsDurationPattern
	if !strings.Contains(output, "$timing-zero: 0s;") {
		t.Errorf("expected $timing-zero: 0s;, got:\n%s", output)
	}
}

func TestFormat_NumberAndFontWeightValues(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:     "size.scale",
			Path:     []string{"size", "scale"},
			Type:     token.TypeNumber,
			RawValue: float64(1.5),
		},
		{
			Name:     "size.integer",
			Path:     []string{"size", "integer"},
			Type:     token.TypeNumber,
			RawValue: float64(42),
		},
		{
			Name:     "font.weight-bold",
			Path:     []string{"font", "weight-bold"},
			Type:     token.TypeFontWeight,
			RawValue: float64(700),
		},
	}

	f := scss.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// float64 1.5 → "1.5"
	if !strings.Contains(output, "$size-scale: 1.5;") {
		t.Errorf("expected $size-scale: 1.5;, got:\n%s", output)
	}
	// integer 42 → "42" (not "42.0")
	if !strings.Contains(output, "$size-integer: 42;") {
		t.Errorf("expected $size-integer: 42;, got:\n%s", output)
	}
	// fontWeight 700 → "700"
	if !strings.Contains(output, "$font-weight-bold: 700;") {
		t.Errorf("expected $font-weight-bold: 700;, got:\n%s", output)
	}
}

func TestFormat_MapAndSliceFallback(t *testing.T) {
	// Map/slice values for types without specific handling should serialize as JSON
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

	f := scss.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// No Go map literals
	if strings.Contains(output, "map[") {
		t.Errorf("output contains Go map literal:\n%s", output)
	}
	// Slice should serialize as JSON array
	if !strings.Contains(output, "[0.25,0.1,0.25,1]") {
		t.Errorf("expected JSON array for cubic bezier, got:\n%s", output)
	}
}

func TestFormat_CustomHeader(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:     "color.primary",
			Path:     []string{"color", "primary"},
			Type:     token.TypeColor,
			RawValue: "#ff0000",
		},
	}

	f := scss.New()
	result, err := f.Format(tokens, formatter.Options{Header: "Custom header"})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	if !strings.Contains(output, "Custom header") {
		t.Errorf("expected custom header in output, got:\n%s", output)
	}
	// Default header should NOT appear
	if strings.Contains(output, "Generated by asimonim") {
		t.Errorf("default header should not appear when custom header is set, got:\n%s", output)
	}
}

func TestFormat_TokenWithDescription(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:        "color.primary",
			Path:        []string{"color", "primary"},
			Type:        token.TypeColor,
			RawValue:    "#ff0000",
			Description: "Primary brand color",
		},
	}

	f := scss.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// Description should appear as /// doc comment
	if !strings.Contains(output, "/// Primary brand color") {
		t.Errorf("expected description doc comment, got:\n%s", output)
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

	f := scss.New()
	result, err := f.Format(tokens, formatter.Options{Prefix: "app"})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	if !strings.Contains(output, "$app-color-primary: #ff0000;") {
		t.Errorf("expected prefixed variable name, got:\n%s", output)
	}
}

func TestFormat_StringValuesWithCSSUnits(t *testing.T) {
	// String values that look like CSS values should pass through unquoted
	tokens := []*token.Token{
		{
			Name:     "size.base",
			Path:     []string{"size", "base"},
			Type:     "", // no type, hits the string suffix checks
			RawValue: "16px",
		},
		{
			Name:     "size.relative",
			Path:     []string{"size", "relative"},
			Type:     "",
			RawValue: "2rem",
		},
		{
			Name:     "size.em",
			Path:     []string{"size", "em"},
			Type:     "",
			RawValue: "1.5em",
		},
		{
			Name:     "size.pct",
			Path:     []string{"size", "pct"},
			Type:     "",
			RawValue: "50%",
		},
		{
			Name:     "color.hex",
			Path:     []string{"color", "hex"},
			Type:     "",
			RawValue: "#abc123",
		},
	}

	f := scss.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// All CSS-like strings should pass through unquoted
	if !strings.Contains(output, "$size-base: 16px;") {
		t.Errorf("expected $size-base: 16px;, got:\n%s", output)
	}
	if !strings.Contains(output, "$size-relative: 2rem;") {
		t.Errorf("expected $size-relative: 2rem;, got:\n%s", output)
	}
	if !strings.Contains(output, "$size-em: 1.5em;") {
		t.Errorf("expected $size-em: 1.5em;, got:\n%s", output)
	}
	if !strings.Contains(output, "$size-pct: 50%;") {
		t.Errorf("expected $size-pct: 50%%;, got:\n%s", output)
	}
	if !strings.Contains(output, "$color-hex: #abc123;") {
		t.Errorf("expected $color-hex: #abc123;, got:\n%s", output)
	}
}
