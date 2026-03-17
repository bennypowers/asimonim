/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package swift_test

import (
	"strings"
	"testing"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/convert/formatter/swift"
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
