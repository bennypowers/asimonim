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
