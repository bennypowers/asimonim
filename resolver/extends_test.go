/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package resolver_test

import (
	"slices"
	"sort"
	"strings"
	"testing"

	"bennypowers.dev/asimonim/parser"
	"bennypowers.dev/asimonim/resolver"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/testutil"
	"bennypowers.dev/asimonim/token"
)

func TestResolveGroupExtensions_Simple(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/v2025_10/extends-simple", "/test")
	data, err := mfs.ReadFile("/test/tokens.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	p := parser.NewJSONParser()
	tokens, err := p.Parse(data, parser.Options{})
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	result, err := resolver.ResolveGroupExtensions(tokens, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"base-color-blue",
		"base-color-red",
		"theme-color-blue",
		"theme-color-green",
		"theme-color-red",
	}

	names := extractNames(result)
	if !slices.Equal(names, expected) {
		t.Errorf("expected tokens %v, got %v", expected, names)
	}
}

func TestResolveGroupExtensions_Chained(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/v2025_10/extends-chained", "/test")
	data, err := mfs.ReadFile("/test/tokens.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	p := parser.NewJSONParser()
	tokens, err := p.Parse(data, parser.Options{})
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	result, err := resolver.ResolveGroupExtensions(tokens, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"base-color-primary",
		"brand-color-accent",
		"brand-color-primary",
		"brand-color-secondary",
		"light-color-primary",
		"light-color-secondary",
	}

	names := extractNames(result)
	if !slices.Equal(names, expected) {
		t.Errorf("expected tokens %v, got %v", expected, names)
	}
}

func TestResolveGroupExtensions_Override(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/v2025_10/extends-override", "/test")
	data, err := mfs.ReadFile("/test/tokens.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	p := parser.NewJSONParser()
	tokens, err := p.Parse(data, parser.Options{})
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	result, err := resolver.ResolveGroupExtensions(tokens, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"base-color-primary",
		"base-color-secondary",
		"theme-color-primary",
		"theme-color-secondary",
	}

	names := extractNames(result)
	if !slices.Equal(names, expected) {
		t.Errorf("expected tokens %v, got %v", expected, names)
	}

	// Verify the override token has the child's value, not the parent's
	var themePrimary *token.Token
	for _, tok := range result {
		if tok.Name == "theme-color-primary" {
			themePrimary = tok
			break
		}
	}

	if themePrimary == nil {
		t.Fatal("expected to find theme-color-primary")
	}

	if themePrimary.Value != "#0000FF" {
		t.Errorf("expected theme-color-primary value #0000FF, got %s", themePrimary.Value)
	}

	if themePrimary.Description != "Theme primary override" {
		t.Errorf("expected description 'Theme primary override', got %s", themePrimary.Description)
	}
}

func TestResolveGroupExtensions_Circular(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/v2025_10/extends-circular", "/test")
	data, err := mfs.ReadFile("/test/tokens.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	p := parser.NewJSONParser()
	tokens, err := p.Parse(data, parser.Options{})
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	_, err = resolver.ResolveGroupExtensions(tokens, data)
	if err == nil {
		t.Fatal("expected error for circular extension")
	}

	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("expected error to contain 'circular', got: %v", err)
	}
}

func TestResolveGroupExtensions_Nested(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/v2025_10/extends-nested", "/test")
	data, err := mfs.ReadFile("/test/tokens.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	p := parser.NewJSONParser()
	tokens, err := p.Parse(data, parser.Options{})
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	result, err := resolver.ResolveGroupExtensions(tokens, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"colors-semantic-error",
		"colors-semantic-success",
		"theme-error",
		"theme-success",
		"theme-warning",
	}

	names := extractNames(result)
	if !slices.Equal(names, expected) {
		t.Errorf("expected tokens %v, got %v", expected, names)
	}
}

func TestResolveGroupExtensions_DraftSchema_NoOp(t *testing.T) {
	// Create tokens with Draft schema
	tokens := []*token.Token{
		{Name: "base-color", Value: "#FF0000", SchemaVersion: schema.Draft},
		{Name: "theme-color", Value: "#00FF00", SchemaVersion: schema.Draft},
	}

	// Even if data has $extends, Draft tokens should be unchanged
	data := []byte(`{
		"base": { "color": { "$value": "#FF0000" } },
		"theme": { "$extends": "#/base", "color": { "$value": "#00FF00" } }
	}`)

	result, err := resolver.ResolveGroupExtensions(tokens, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return exactly the same tokens
	if len(result) != len(tokens) {
		t.Errorf("expected %d tokens, got %d", len(tokens), len(result))
	}
}

// extractNames returns sorted token names from the result.
func extractNames(tokens []*token.Token) []string {
	names := make([]string, len(tokens))
	for i, tok := range tokens {
		names[i] = tok.Name
	}
	sort.Strings(names)
	return names
}
