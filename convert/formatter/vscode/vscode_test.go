/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package vscode_test

import (
	"encoding/json"
	"slices"
	"testing"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/convert/formatter/vscode"
	"bennypowers.dev/asimonim/token"
)

func TestFormat_Basic(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:        "color-primary",
			Path:        []string{"color", "primary"},
			Type:        token.TypeColor,
			Value:       "#FF6B35",
			Description: "Primary brand color",
		},
		{
			Name:  "spacing-small",
			Path:  []string{"spacing", "small"},
			Type:  token.TypeDimension,
			Value: "4px",
		},
	}

	f := vscode.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var snippets map[string]vscode.Snippet
	if err := json.Unmarshal(result, &snippets); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// Check color-primary snippet
	colorSnippet, ok := snippets["color-primary"]
	if !ok {
		t.Fatal("expected color-primary snippet")
	}

	if colorSnippet.Scope != "css,scss,less,stylus,postcss" {
		t.Errorf("expected scope css,scss,less,stylus,postcss, got %s", colorSnippet.Scope)
	}

	if len(colorSnippet.Body) != 1 || colorSnippet.Body[0] != "var(--color-primary)" {
		t.Errorf("expected body [var(--color-primary)], got %v", colorSnippet.Body)
	}

	if colorSnippet.Description != "Primary brand color" {
		t.Errorf("expected description 'Primary brand color', got %s", colorSnippet.Description)
	}

	// Check prefix includes token name and hex value (without #)
	hasName := false
	hasHex := false
	for _, prefix := range colorSnippet.Prefix {
		if prefix == "color-primary" {
			hasName = true
		}
		if prefix == "FF6B35" {
			hasHex = true
		}
	}
	if !hasName {
		t.Error("expected color-primary in prefixes")
	}
	if !hasHex {
		t.Error("expected hex value FF6B35 in prefixes")
	}

	// Check spacing-small snippet
	spacingSnippet, ok := snippets["spacing-small"]
	if !ok {
		t.Fatal("expected spacing-small snippet")
	}

	if spacingSnippet.Description != "" {
		t.Errorf("expected empty description, got %s", spacingSnippet.Description)
	}
}

func TestFormat_WithPrefix(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:  "color-primary",
			Path:  []string{"color", "primary"},
			Type:  token.TypeColor,
			Value: "#FF6B35",
		},
	}

	f := vscode.New()
	result, err := f.Format(tokens, formatter.Options{Prefix: "rh"})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var snippets map[string]vscode.Snippet
	if err := json.Unmarshal(result, &snippets); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// Check snippet has prefixed name
	snippet, ok := snippets["rh-color-primary"]
	if !ok {
		t.Fatal("expected rh-color-primary snippet")
	}

	if snippet.Body[0] != "var(--rh-color-primary)" {
		t.Errorf("expected body var(--rh-color-primary), got %s", snippet.Body[0])
	}
}

func TestFormat_CamelCasePrefix(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:  "font-size-large",
			Path:  []string{"font", "size", "large"},
			Type:  token.TypeDimension,
			Value: "24px",
		},
	}

	f := vscode.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var snippets map[string]vscode.Snippet
	if err := json.Unmarshal(result, &snippets); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	snippet, ok := snippets["font-size-large"]
	if !ok {
		t.Fatal("expected font-size-large snippet")
	}

	// Check for camelCase prefix
	if !slices.Contains(snippet.Prefix, "fontSizeLarge") {
		t.Errorf("expected camelCase prefix fontSizeLarge in %v", snippet.Prefix)
	}
}

func TestFormat_ValidJSON(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:        "color-primary",
			Path:        []string{"color", "primary"},
			Type:        token.TypeColor,
			Value:       "#FF6B35",
			Description: "Contains \"quotes\" and 'apostrophes'",
		},
	}

	f := vscode.New()
	result, err := f.Format(tokens, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Verify the output is valid JSON
	var snippets map[string]any
	if err := json.Unmarshal(result, &snippets); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
}
