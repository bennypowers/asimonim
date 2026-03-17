/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package flatjson_test

import (
	"encoding/json"
	"testing"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/convert/formatter/flatjson"
	"bennypowers.dev/asimonim/token"
)

func TestFormat(t *testing.T) {
	tokens := []*token.Token{
		{Name: "color-primary", Path: []string{"color", "primary"}, Value: "#FF6B35"},
		{Name: "spacing-small", Path: []string{"spacing", "small"}, Value: "4px"},
	}

	f := flatjson.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	// Default delimiter is "-"
	if parsed["color-primary"] != "#FF6B35" {
		t.Errorf("color-primary = %v, want #FF6B35", parsed["color-primary"])
	}
	if parsed["spacing-small"] != "4px" {
		t.Errorf("spacing-small = %v, want 4px", parsed["spacing-small"])
	}
}

func TestFormat_WithPrefix(t *testing.T) {
	tokens := []*token.Token{
		{Name: "color-primary", Path: []string{"color", "primary"}, Value: "#FF6B35"},
	}

	f := flatjson.New()
	result, err := f.Format(tokens, formatter.Options{Prefix: "rh"})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if parsed["rh-color-primary"] != "#FF6B35" {
		t.Errorf("expected rh-color-primary key, got keys: %v", parsed)
	}
}

func TestFormat_CustomDelimiter(t *testing.T) {
	tokens := []*token.Token{
		{Name: "color-primary", Path: []string{"color", "primary"}, Value: "#FF6B35"},
	}

	f := flatjson.New()
	result, err := f.Format(tokens, formatter.Options{Delimiter: "_"})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if parsed["color_primary"] != "#FF6B35" {
		t.Errorf("expected color_primary key, got keys: %v", parsed)
	}
}

func TestFormat_UsesResolvedValue(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:          "color-secondary",
			Path:          []string{"color", "secondary"},
			Value:         "{color.primary}",
			ResolvedValue: "#FF6B35",
		},
	}

	f := flatjson.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	// ResolvedValue should be used over Value
	if parsed["color-secondary"] != "#FF6B35" {
		t.Errorf("expected resolved value #FF6B35, got %v", parsed["color-secondary"])
	}
}
