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
