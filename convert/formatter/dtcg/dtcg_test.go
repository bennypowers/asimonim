/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package dtcg_test

import (
	"encoding/json"
	"testing"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/convert/formatter/dtcg"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/testutil"
	"bennypowers.dev/asimonim/token"
)

func TestFormat(t *testing.T) {
	serialize := func(tokens []*token.Token) map[string]any {
		result := make(map[string]any)
		for _, tok := range tokens {
			result[tok.Name] = map[string]any{
				"$value": tok.Value,
				"$type":  tok.Type,
			}
		}
		return result
	}

	allTokens := testutil.ParseFixtureTokens(t, "fixtures/v2025_10/all-color-spaces", schema.V2025_10)
	tokens := []*token.Token{
		testutil.TokenByPath(t, allTokens, "color.srgb-hex"),
		testutil.TokenByPath(t, allTokens, "spacing.small"),
	}

	f := dtcg.New(serialize)
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	// color.srgb-hex: structured sRGB color -> $value and $type preserved
	colorTok, ok := parsed["color-srgb-hex"].(map[string]any)
	if !ok {
		t.Fatal("expected color-srgb-hex in output")
	}
	if colorTok["$type"] != "color" {
		t.Errorf("color type = %v, want color", colorTok["$type"])
	}

	// spacing.small: {value: 4, unit: "px"} -> $value and $type: "dimension"
	spacingTok, ok := parsed["spacing-small"].(map[string]any)
	if !ok {
		t.Fatal("expected spacing-small in output")
	}
	if spacingTok["$type"] != "dimension" {
		t.Errorf("spacing type = %v, want dimension", spacingTok["$type"])
	}
}
