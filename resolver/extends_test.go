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

func TestResolveGroupExtensions_WithExtensions(t *testing.T) {
	// Tests that deepCopyMap, deepCopyAny, and deepCopySlice correctly deep copy
	// token extensions containing nested maps and slices
	mfs := testutil.NewFixtureFS(t, "fixtures/v2025_10/extends-with-extensions", "/test")
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

	// Should have: base-primary, theme-primary (inherited), theme-secondary
	expected := []string{
		"base-primary",
		"theme-primary",
		"theme-secondary",
	}

	names := extractNames(result)
	if !slices.Equal(names, expected) {
		t.Errorf("expected tokens %v, got %v", expected, names)
	}

	// Find the inherited theme-primary token
	var themePrimary *token.Token
	var basePrimary *token.Token
	for _, tok := range result {
		switch tok.Name {
		case "theme-primary":
			themePrimary = tok
		case "base-primary":
			basePrimary = tok
		}
	}

	if themePrimary == nil {
		t.Fatal("expected to find theme-primary")
	}
	if basePrimary == nil {
		t.Fatal("expected to find base-primary")
	}

	// Verify extensions were deep copied (not shared references)
	if themePrimary.Extensions == nil {
		t.Fatal("expected theme-primary to have extensions from base-primary")
	}

	// Verify the nested slice was deep copied
	tags, ok := themePrimary.Extensions["com.example.tags"].([]any)
	if !ok {
		t.Fatalf("expected com.example.tags to be []any, got %T", themePrimary.Extensions["com.example.tags"])
	}
	// tags: ["brand", "primary"]
	if len(tags) != 2 || tags[0] != "brand" || tags[1] != "primary" {
		t.Errorf("expected tags [brand, primary], got %v", tags)
	}

	// Verify the nested map was deep copied
	metadata, ok := themePrimary.Extensions["com.example.metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected com.example.metadata to be map[string]any, got %T", themePrimary.Extensions["com.example.metadata"])
	}
	// metadata.source: "figma"
	if metadata["source"] != "figma" {
		t.Errorf("expected metadata.source 'figma', got %v", metadata["source"])
	}

	// Verify deep copy isolation: modifying the copy should not affect the original
	tags[0] = "modified"
	baseTags, ok := basePrimary.Extensions["com.example.tags"].([]any)
	if !ok {
		t.Fatalf("expected base com.example.tags to be []any, got %T", basePrimary.Extensions["com.example.tags"])
	}
	// base-primary.tags[0]: "brand" (should not be "modified")
	if baseTags[0] != "brand" {
		t.Errorf("deep copy failed: modifying inherited token affected base token, got %v", baseTags[0])
	}
}

func TestResolveGroupExtensions_InvalidJSONPointer(t *testing.T) {
	// $extends with invalid JSON pointer (no "#/" prefix) should be ignored
	tokens := []*token.Token{
		{
			Name:          "base-color",
			Path:          []string{"base", "color"},
			Value:         "#FF0000",
			Type:          token.TypeColor,
			SchemaVersion: schema.V2025_10,
		},
	}

	// $extends without "#/" prefix -> parseJSONPointer returns nil -> no extension
	data := []byte(`{
		"base": { "color": { "$type": "color", "$value": "#FF0000" } },
		"theme": { "$extends": "base", "bg": { "$type": "color", "$value": "#00FF00" } }
	}`)

	result, err := resolver.ResolveGroupExtensions(tokens, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return exactly the same tokens (invalid pointer ignored)
	if len(result) != len(tokens) {
		t.Errorf("expected %d tokens (invalid pointer ignored), got %d", len(tokens), len(result))
	}
}

func TestResolveGroupExtensions_EmptyJSONPointerPath(t *testing.T) {
	// $extends with "#/" but empty path after prefix -> parseJSONPointer returns nil
	tokens := []*token.Token{
		{
			Name:          "base-color",
			Path:          []string{"base", "color"},
			Value:         "#FF0000",
			Type:          token.TypeColor,
			SchemaVersion: schema.V2025_10,
		},
	}

	data := []byte(`{
		"base": { "color": { "$type": "color", "$value": "#FF0000" } },
		"theme": { "$extends": "#/", "bg": { "$type": "color", "$value": "#00FF00" } }
	}`)

	result, err := resolver.ResolveGroupExtensions(tokens, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return exactly the same tokens (empty path ignored)
	if len(result) != len(tokens) {
		t.Errorf("expected %d tokens (empty pointer path ignored), got %d", len(tokens), len(result))
	}
}

func TestResolveGroupExtensions_EmptyTokens(t *testing.T) {
	// Empty tokens slice -> no-op
	result, err := resolver.ResolveGroupExtensions([]*token.Token{}, []byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 tokens, got %d", len(result))
	}
}

func TestResolveGroupExtensions_NilExtensions(t *testing.T) {
	// Token with nil Extensions -> deepCopyMap(nil) returns nil
	tokens := []*token.Token{
		{
			Name:          "base-color",
			Path:          []string{"base", "color"},
			Value:         "#FF0000",
			Type:          token.TypeColor,
			SchemaVersion: schema.V2025_10,
			Extensions:    nil,
		},
	}

	data := []byte(`{
		"base": { "color": { "$type": "color", "$value": "#FF0000" } },
		"theme": { "$extends": "#/base", "bg": { "$type": "color", "$value": "#00FF00" } }
	}`)

	result, err := resolver.ResolveGroupExtensions(tokens, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have base-color + inherited theme-color
	expected := []string{"base-color", "theme-color"}
	names := extractNames(result)
	if !slices.Equal(names, expected) {
		t.Errorf("expected tokens %v, got %v", expected, names)
	}

	// Inherited token should also have nil Extensions
	for _, tok := range result {
		if tok.Name == "theme-color" {
			if tok.Extensions != nil {
				t.Errorf("expected nil Extensions for inherited token with nil source, got %v", tok.Extensions)
			}
		}
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
