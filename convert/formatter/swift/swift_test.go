/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package swift_test

import (
	"flag"
	"os"
	"strings"
	"testing"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/convert/formatter/swift"
	"github.com/stretchr/testify/require"

	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/testutil"
	"bennypowers.dev/asimonim/token"
)

func TestFormat_V2025_10_StructuredColors(t *testing.T) {
	allTokens := testutil.ParseFixtureTokens(t, "fixtures/v2025_10/all-color-spaces", schema.V2025_10)

	tokens := []*token.Token{
		testutil.TokenByPath(t, allTokens, "color.srgb-hex"),    // srgb, [1, 0.42, 0.21], hex: "#FF6B36"
		testutil.TokenByPath(t, allTokens, "color.oklch-alpha"), // oklch, [0.7, 0.15, 180], alpha: 0.8
		testutil.TokenByPath(t, allTokens, "color.display-p3"),  // display-p3, [1, 0.5, 0.25]
		testutil.TokenByPath(t, allTokens, "color.srgb-no-hex"), // srgb, [1, 0.5, 0.25]
	}

	f := swift.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// Swift should convert structured colors to Color() initializers
	expectations := []string{
		"Color(.sRGB, red: 1, green: 0.42, blue: 0.21)",                    // srgb-hex: srgb [1, 0.42, 0.21]
		"Color(.sRGB, red: 0.7, green: 0.15, blue: 180, opacity: 0.8)",     // oklch-alpha: oklch [0.7, 0.15, 180] alpha 0.8
		"Color(.displayP3, red: 1, green: 0.5, blue: 0.25)",                // display-p3: [1, 0.5, 0.25]
	}

	for _, expected := range expectations {
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

	f := swift.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// spacing.small: {value: 4, unit: "px"} → CGFloat(4)
	if !strings.Contains(output, "CGFloat(4)") {
		t.Errorf("expected CGFloat(4) for px dimension, got:\n%s", output)
	}
	// spacing.medium: {value: 1.5, unit: "rem"} → CGFloat(1.5)
	if !strings.Contains(output, "CGFloat(1.5)") {
		t.Errorf("expected CGFloat(1.5) for rem dimension, got:\n%s", output)
	}

	if strings.Contains(output, "map[") {
		t.Errorf("output contains Go map literal:\n%s", output)
	}
}

// Regression test: nil dimension value should not produce invalid Swift
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

	f := swift.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)
	if strings.Contains(output, "CGFloat(<nil>)") || strings.Contains(output, "CGFloat(nil)") {
		t.Errorf("nil dimension value produced invalid Swift: %s", output)
	}
}

// Regression test: Swift comment injection via unit string
func TestFormat_DimensionCommentInjection(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:          "spacing.evil",
			Path:          []string{"spacing", "evil"},
			Type:          token.TypeDimension,
			SchemaVersion: schema.V2025_10,
			RawValue:      map[string]any{"value": 16.0, "unit": "px */ inject /*"},
		},
	}

	f := swift.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)
	// The */ in the unit should be sanitized to prevent comment breakout
	if strings.Count(output, "*/") > 1 {
		t.Errorf("unit string broke out of Swift comment: %s", output)
	}

	// Also test /* injection
	tokens[0].RawValue = map[string]any{"value": 16.0, "unit": "/* bad"}
	result, err = f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	output = string(result)
	if strings.Count(output, "/*") > 1 {
		t.Errorf("unit string opened an extra block comment: %s", output)
	}
}

func TestFormat_UngroupedTokens(t *testing.T) {
	// Tokens with no type end up in the "Other" enum section
	tokens := []*token.Token{
		{
			Name:     "misc.something",
			Path:     []string{"misc", "something"},
			Type:     "", // no type
			RawValue: "hello",
		},
		{
			Name:     "misc.another",
			Path:     []string{"misc", "another"},
			Type:     "", // no type
			RawValue: float64(42),
		},
	}

	f := swift.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// Should contain the "Other" section header
	if !strings.Contains(output, "// MARK: - Other") {
		t.Errorf("expected Other section header, got:\n%s", output)
	}
	if !strings.Contains(output, "public enum Other") {
		t.Errorf("expected Other enum, got:\n%s", output)
	}
	// Values should be rendered as quoted strings via fallback
	if !strings.Contains(output, "miscSomething") {
		t.Errorf("expected miscSomething variable name, got:\n%s", output)
	}
	if !strings.Contains(output, "miscAnother") {
		t.Errorf("expected miscAnother variable name, got:\n%s", output)
	}
}

func TestFormat_DurationValues(t *testing.T) {
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
			Name:     "timing.bare",
			Path:     []string{"timing", "bare"},
			Type:     token.TypeDuration,
			RawValue: "0.3",
		},
	}

	f := swift.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// 200ms → TimeInterval(0.2) (converted from ms to s)
	if !strings.Contains(output, "TimeInterval(0.2)") {
		t.Errorf("expected TimeInterval(0.2) for 200ms, got:\n%s", output)
	}
	// 1.5s → TimeInterval(1.5)
	if !strings.Contains(output, "TimeInterval(1.5)") {
		t.Errorf("expected TimeInterval(1.5) for 1.5s, got:\n%s", output)
	}
	// bare number → TimeInterval(0.3)
	if !strings.Contains(output, "TimeInterval(0.3)") {
		t.Errorf("expected TimeInterval(0.3) for bare duration, got:\n%s", output)
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
		{
			Name:     "font.weight-light",
			Path:     []string{"font", "weight-light"},
			Type:     token.TypeFontWeight,
			RawValue: float64(300),
		},
	}

	f := swift.New()
	result, err := f.Format(tokens, formatter.Options{})
	require.NoError(t, err)

	// "number" maps to "NumberTokens" to avoid shadowing Swift's built-in Number
	assertGolden(t, result, "testdata/golden/number-fontweight-values.swift")
}

func TestFormat_FontFamilyValues(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:     "font.body",
			Path:     []string{"font", "body"},
			Type:     token.TypeFontFamily,
			RawValue: "Helvetica Neue",
		},
	}

	f := swift.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// FontFamily string → falls through to default quoted output
	if !strings.Contains(output, "public enum FontFamily") {
		t.Errorf("expected FontFamily enum section, got:\n%s", output)
	}
	if !strings.Contains(output, `"Helvetica Neue"`) {
		t.Errorf("expected quoted font family value, got:\n%s", output)
	}
}

func TestFormat_StringValues(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:     "label.greeting",
			Path:     []string{"label", "greeting"},
			Type:     token.TypeString,
			RawValue: "Hello World",
		},
	}

	f := swift.New()
	result, err := f.Format(tokens, formatter.Options{})
	require.NoError(t, err)

	// "string" type maps to "StringTokens" to avoid shadowing Swift's built-in String
	assertGolden(t, result, "testdata/golden/string-values.swift")
}

func TestFormat_MoreColorSpaces(t *testing.T) {
	allTokens := testutil.ParseFixtureTokens(t, "fixtures/v2025_10/all-color-spaces", schema.V2025_10)

	tokens := []*token.Token{
		testutil.TokenByPath(t, allTokens, "color.lab"),         // lab [50, 20, -30]
		testutil.TokenByPath(t, allTokens, "color.lch"),         // lch [50, 30, 270]
		testutil.TokenByPath(t, allTokens, "color.oklab"),       // oklab [0.5, 0.1, -0.1]
		testutil.TokenByPath(t, allTokens, "color.xyz-d50"),     // xyz-d50 [0.4, 0.3, 0.2]
		testutil.TokenByPath(t, allTokens, "color.xyz-d65"),     // xyz-d65 [0.4, 0.3, 0.2]
		testutil.TokenByPath(t, allTokens, "color.srgb-linear"), // srgb-linear [0.5, 0.3, 0.1]
	}

	f := swift.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// lab, lch, oklab → .sRGB (mapped to sRGB since Swift doesn't have native support)
	// lab: [50, 20, -30] uses .sRGB color space
	if !strings.Contains(output, "Color(.sRGB, red: 50, green: 20, blue: -30)") {
		t.Errorf("expected lab color with .sRGB, got:\n%s", output)
	}
	// lch: [50, 30, 270] uses .sRGB color space
	if !strings.Contains(output, "Color(.sRGB, red: 50, green: 30, blue: 270)") {
		t.Errorf("expected lch color with .sRGB, got:\n%s", output)
	}
	// oklab: [0.5, 0.1, -0.1] uses .sRGB color space
	if !strings.Contains(output, "Color(.sRGB, red: 0.5, green: 0.1, blue: -0.1)") {
		t.Errorf("expected oklab color with .sRGB, got:\n%s", output)
	}
	// xyz-d50: [0.4, 0.3, 0.2] → .genericXYZ
	if !strings.Contains(output, "Color(.genericXYZ, red: 0.4, green: 0.3, blue: 0.2)") {
		t.Errorf("expected xyz-d50 with .genericXYZ, got:\n%s", output)
	}
	// xyz-d65: [0.4, 0.3, 0.2] → .genericXYZ
	if !strings.Contains(output, "Color(.genericXYZ, red: 0.4, green: 0.3, blue: 0.2)") {
		t.Errorf("expected xyz-d65 with .genericXYZ, got:\n%s", output)
	}
	// srgb-linear: [0.5, 0.3, 0.1] → .linearSRGB
	if !strings.Contains(output, "Color(.linearSRGB, red: 0.5, green: 0.3, blue: 0.1)") {
		t.Errorf("expected srgb-linear with .linearSRGB, got:\n%s", output)
	}
}

func TestFormat_StringColorValue(t *testing.T) {
	// String color (draft-style) should be parsed by csscolorparser
	tokens := []*token.Token{
		{
			Name:          "color.primary",
			Path:          []string{"color", "primary"},
			Type:          token.TypeColor,
			SchemaVersion: schema.Draft,
			RawValue:      "#ff0000",
		},
		{
			Name:          "color.unparseable",
			Path:          []string{"color", "unparseable"},
			Type:          token.TypeColor,
			SchemaVersion: schema.Draft,
			RawValue:      "not-a-color",
		},
	}

	f := swift.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// #ff0000 → Color(.sRGB, red: 1, green: 0, blue: 0)
	if !strings.Contains(output, "Color(.sRGB, red: 1, green: 0, blue: 0)") {
		t.Errorf("expected parsed sRGB color for #ff0000, got:\n%s", output)
	}
	// "not-a-color" → quoted string fallback
	if !strings.Contains(output, `"not-a-color"`) {
		t.Errorf("expected quoted fallback for unparseable color, got:\n%s", output)
	}
}

func TestFormat_CustomPrefix(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:     "color.primary",
			Path:     []string{"color", "primary"},
			Type:     token.TypeColor,
			RawValue: "#ff0000",
		},
	}

	f := swift.New()
	result, err := f.Format(tokens, formatter.Options{Prefix: "my-app"})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// Prefix should produce "MyAppTokens" enum
	if !strings.Contains(output, "public enum MyAppTokens") {
		t.Errorf("expected MyAppTokens enum with prefix, got:\n%s", output)
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

	f := swift.New()
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

	f := swift.New()
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

func TestFormat_DimensionStringValue(t *testing.T) {
	// Draft-style dimension with string value like "16px"
	tokens := []*token.Token{
		{
			Name:          "spacing.base",
			Path:          []string{"spacing", "base"},
			Type:          token.TypeDimension,
			SchemaVersion: schema.Draft,
			RawValue:      "16px",
		},
		{
			Name:          "spacing.em",
			Path:          []string{"spacing", "em"},
			Type:          token.TypeDimension,
			SchemaVersion: schema.Draft,
			RawValue:      "2em",
		},
		{
			Name:          "spacing.rem",
			Path:          []string{"spacing", "rem"},
			Type:          token.TypeDimension,
			SchemaVersion: schema.Draft,
			RawValue:      "1.5rem",
		},
	}

	f := swift.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// "16px" → CGFloat(16)
	if !strings.Contains(output, "CGFloat(16)") {
		t.Errorf("expected CGFloat(16) for 16px string, got:\n%s", output)
	}
	// "2em" → CGFloat(2)
	if !strings.Contains(output, "CGFloat(2)") {
		t.Errorf("expected CGFloat(2) for 2em string, got:\n%s", output)
	}
	// "1.5rem" → CGFloat(1.5)
	if !strings.Contains(output, "CGFloat(1.5)") {
		t.Errorf("expected CGFloat(1.5) for 1.5rem string, got:\n%s", output)
	}
}

func TestFormat_StructuredColorFewerThan3Components(t *testing.T) {
	// Structured color with fewer than 3 components should produce Color.clear
	tokens := []*token.Token{
		{
			Name:          "color.broken",
			Path:          []string{"color", "broken"},
			Type:          token.TypeColor,
			SchemaVersion: schema.V2025_10,
			RawValue: map[string]any{
				"colorSpace": "srgb",
				"components": []any{0.5, 0.3},
				"alpha":      1.0,
			},
		},
	}

	f := swift.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// Should produce Color.clear as fallback
	if !strings.Contains(output, "Color.clear") {
		t.Errorf("expected Color.clear for fewer than 3 components, got:\n%s", output)
	}
}

func TestFormat_DimensionWithUnitComment(t *testing.T) {
	allTokens := testutil.ParseFixtureTokens(t, "fixtures/v2025_10/all-color-spaces", schema.V2025_10)

	tok := testutil.TokenByPath(t, allTokens, "spacing.small") // {value: 4, unit: "px"}

	f := swift.New()
	result, err := f.Format([]*token.Token{tok}, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// spacing.small: {value: 4, unit: "px"} -> CGFloat(4) /* px */
	if !strings.Contains(output, "CGFloat(4) /* px */") {
		t.Errorf("expected CGFloat(4) /* px */ with unit comment, got:\n%s", output)
	}
}

// assertGolden compares result against a golden file, or updates the
// golden file when -update is passed.
func assertGolden(t *testing.T, result []byte, goldenPath string) {
	t.Helper()

	updateFlag := flag.Lookup("update")
	if updateFlag != nil && updateFlag.Value.String() == "true" {
		if err := os.MkdirAll("testdata/golden", 0o755); err != nil {
			t.Fatalf("failed to create golden dir: %v", err)
		}
		err := os.WriteFile(goldenPath, result, 0o644)
		require.NoError(t, err)
		return
	}

	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "golden file %s not found; run with -update to create", goldenPath)
	require.Equal(t, string(expected), string(result))
}
