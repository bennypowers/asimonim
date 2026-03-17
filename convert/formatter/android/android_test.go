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
