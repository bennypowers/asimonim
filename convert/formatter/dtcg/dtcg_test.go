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

	tokens := []*token.Token{
		{Name: "color-primary", Value: "#FF6B35", Type: "color"},
		{Name: "spacing-small", Value: "4px", Type: "dimension"},
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

	// Check color token
	colorTok, ok := parsed["color-primary"].(map[string]any)
	if !ok {
		t.Fatal("expected color-primary in output")
	}
	if colorTok["$value"] != "#FF6B35" {
		t.Errorf("color value = %v, want #FF6B35", colorTok["$value"])
	}
	if colorTok["$type"] != "color" {
		t.Errorf("color type = %v, want color", colorTok["$type"])
	}

	// Check spacing token
	spacingTok, ok := parsed["spacing-small"].(map[string]any)
	if !ok {
		t.Fatal("expected spacing-small in output")
	}
	if spacingTok["$value"] != "4px" {
		t.Errorf("spacing value = %v, want 4px", spacingTok["$value"])
	}
}
