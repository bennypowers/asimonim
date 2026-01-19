/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package parser_test

import (
	"testing"

	"bennypowers.dev/asimonim/parser"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/testutil"
)

func TestJSONParser_Parse(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")

	p := parser.NewJSONParser()
	tokens, err := p.ParseFile(mfs, "/test/tokens.json", parser.Options{
		SchemaVersion: schema.Draft,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tokens) != 5 {
		t.Errorf("expected 5 tokens, got %d", len(tokens))
	}

	// Check that we found expected tokens
	names := make(map[string]bool)
	for _, tok := range tokens {
		names[tok.Name] = true
	}

	expected := []string{"color-primary", "color-secondary", "spacing-small", "spacing-medium", "spacing-large"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected token %s not found", name)
		}
	}
}

func TestJSONParser_V2025_10(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/v2025_10/structured-colors", "/test")

	p := parser.NewJSONParser()
	tokens, err := p.ParseFile(mfs, "/test/tokens.json", parser.Options{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens, got %d", len(tokens))
	}
}

func TestJSONParser_SkipPositions(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
	data, err := mfs.ReadFile("/test/tokens.json")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	p := parser.NewJSONParser()

	// Parse with position tracking
	withPositions, err := p.Parse(data, parser.Options{
		SchemaVersion: schema.Draft,
	})
	if err != nil {
		t.Fatalf("unexpected error with positions: %v", err)
	}

	// Parse without position tracking (fast mode)
	withoutPositions, err := p.Parse(data, parser.Options{
		SchemaVersion: schema.Draft,
		SkipPositions: true,
	})
	if err != nil {
		t.Fatalf("unexpected error without positions: %v", err)
	}

	// Should have same number of tokens
	if len(withPositions) != len(withoutPositions) {
		t.Fatalf("expected %d tokens, got %d", len(withPositions), len(withoutPositions))
	}

	// Build map for comparison
	posMap := make(map[string]*struct {
		name, value, typ, desc string
	})
	for _, tok := range withPositions {
		posMap[tok.Name] = &struct {
			name, value, typ, desc string
		}{tok.Name, tok.Value, tok.Type, tok.Description}
	}

	// Compare all tokens (except Line/Character which should be 0 in fast mode)
	for _, tok := range withoutPositions {
		expected, ok := posMap[tok.Name]
		if !ok {
			t.Errorf("token %s not found in position-tracked results", tok.Name)
			continue
		}
		if tok.Name != expected.name {
			t.Errorf("name mismatch: got %s, want %s", tok.Name, expected.name)
		}
		if tok.Value != expected.value {
			t.Errorf("value mismatch for %s: got %s, want %s", tok.Name, tok.Value, expected.value)
		}
		if tok.Type != expected.typ {
			t.Errorf("type mismatch for %s: got %s, want %s", tok.Name, tok.Type, expected.typ)
		}
		if tok.Description != expected.desc {
			t.Errorf("description mismatch for %s: got %s, want %s", tok.Name, tok.Description, expected.desc)
		}
		// Fast mode should have zero positions
		if tok.Line != 0 || tok.Character != 0 {
			t.Errorf("expected zero positions in fast mode for %s, got Line=%d Character=%d", tok.Name, tok.Line, tok.Character)
		}
	}
}
