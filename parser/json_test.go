/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
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

func TestJSONParser_V2025_10_CurlyRefs(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/v2025_10/curly-refs", "/test")

	p := parser.NewJSONParser()
	tokens, err := p.ParseFile(mfs, "/test/tokens.json", parser.Options{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tokens) != 3 {
		t.Errorf("expected 3 tokens, got %d", len(tokens))
	}

	// Check that curly-brace refs are preserved
	tokenByName := make(map[string]*struct {
		value, desc string
	})
	for _, tok := range tokens {
		tokenByName[tok.Name] = &struct {
			value, desc string
		}{tok.Value, tok.Description}
	}

	// Secondary should have curly-brace ref
	if sec := tokenByName["color-brand-secondary"]; sec != nil {
		if sec.value != "{color.brand.primary}" {
			t.Errorf("expected secondary value to be curly-brace ref, got %s", sec.value)
		}
	} else {
		t.Error("expected token color-brand-secondary not found")
	}

	// Action should have chained curly-brace ref
	if action := tokenByName["color-semantic-action"]; action != nil {
		if action.value != "{color.brand.secondary}" {
			t.Errorf("expected action value to be curly-brace ref, got %s", action.value)
		}
	} else {
		t.Error("expected token color-semantic-action not found")
	}
}

func TestJSONParser_ParseYAML(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple-yaml", "/test")

	p := parser.NewJSONParser()
	tokens, err := p.ParseFile(mfs, "/test/tokens.yaml", parser.Options{
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

	// Check that $type inheritance works
	for _, tok := range tokens {
		if tok.Type == "" {
			t.Errorf("expected token %s to have a type", tok.Name)
		}
	}
}

func TestJSONParser_AutoDetectSchema(t *testing.T) {
	t.Run("detects v2025.10 from $schema field", func(t *testing.T) {
		mfs := testutil.NewFixtureFS(t, "fixtures/v2025_10/structured-colors", "/test")

		p := parser.NewJSONParser()
		// Don't set SchemaVersion - let it auto-detect
		tokens, err := p.ParseFile(mfs, "/test/tokens.json", parser.Options{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, tok := range tokens {
			if tok.SchemaVersion != schema.V2025_10 {
				t.Errorf("expected token %s to have schema version v2025.10, got %s", tok.Name, tok.SchemaVersion)
			}
		}
	})

	t.Run("explicit SchemaVersion overrides auto-detection", func(t *testing.T) {
		mfs := testutil.NewFixtureFS(t, "fixtures/v2025_10/structured-colors", "/test")

		p := parser.NewJSONParser()
		// Explicitly set Draft - should override the $schema field
		tokens, err := p.ParseFile(mfs, "/test/tokens.json", parser.Options{
			SchemaVersion: schema.Draft,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, tok := range tokens {
			if tok.SchemaVersion != schema.Draft {
				t.Errorf("expected token %s to have schema version draft (override), got %s", tok.Name, tok.SchemaVersion)
			}
		}
	})

	t.Run("defaults to draft when no $schema field", func(t *testing.T) {
		mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")

		p := parser.NewJSONParser()
		tokens, err := p.ParseFile(mfs, "/test/tokens.json", parser.Options{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, tok := range tokens {
			if tok.SchemaVersion != schema.Draft {
				t.Errorf("expected token %s to have schema version draft, got %s", tok.Name, tok.SchemaVersion)
			}
		}
	})
}

func TestJSONParser_NumericValues(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/numeric-values", "/test")

	p := parser.NewJSONParser()
	tokens, err := p.ParseFile(mfs, "/test/tokens.json", parser.Options{
		SchemaVersion: schema.Draft,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tokens) != 4 {
		t.Errorf("expected 4 tokens, got %d", len(tokens))
	}

	// Build lookup by name
	tokenByName := make(map[string]string)
	for _, tok := range tokens {
		tokenByName[tok.Name] = tok.Value
	}

	// Numeric $value fields must populate Token.Value as a string
	tests := []struct {
		name     string
		expected string
	}{
		{"font-weight-regular", "400"},
		{"font-weight-medium", "500"},
		{"font-weight-bold", "700"},
		{"opacity-half", "0.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := tokenByName[tt.name]
			if !ok {
				t.Fatalf("token %s not found", tt.name)
			}
			if val != tt.expected {
				t.Errorf("Token.Value = %q, want %q", val, tt.expected)
			}
		})
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

func TestJSONParser_AutoDetectSchemaFromJSONC(t *testing.T) {
	// The block comment makes this invalid YAML, but valid JSONC. Schema
	// detection must inspect the normalized JSON so the 2025.10 $ref is
	// recognized instead of falling back to Draft.
	data := []byte(`{
		/* 2025.10 $ref tokens require schema detection */
		"color": {
			"primary": {"$type": "color", "$value": "#ff0000"},
			"alias": {"$type": "color", "$ref": "#/color/primary"}
		}
	}`)

	tokens, err := parser.NewJSONParser().Parse(data, parser.Options{SkipPositions: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var aliasFound bool
	for _, tok := range tokens {
		if tok.Name == "color-alias" {
			aliasFound = true
			if tok.SchemaVersion != schema.V2025_10 {
				t.Errorf("color-alias schema = %s, want %s", tok.SchemaVersion, schema.V2025_10)
			}
		}
	}
	if !aliasFound {
		t.Fatal("color-alias token not found; schema auto-detection failed for JSONC input")
	}
}

func TestJSONParser_ParseJSONCWithLeadingComment(t *testing.T) {
	// A leading // comment must not cause the document to be classified as
	// YAML. The parser should recognize JSONC and parse it.
	data := []byte(`// tokens for the 2025.10 schema
{
	"$schema": "https://www.designtokens.org/schemas/2025.10.json",
	"spacing": {
		"small": {"$type": "number", "$value": 4}
	}
}`)

	tokens, err := parser.NewJSONParser().Parse(data, parser.Options{SkipPositions: true})
	if err != nil {
		t.Fatalf("unexpected error parsing JSONC with leading comment: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("got %d tokens, want 1", len(tokens))
	}
	if tokens[0].Name != "spacing-small" {
		t.Errorf("token name = %q, want %q", tokens[0].Name, "spacing-small")
	}
	if tokens[0].SchemaVersion != schema.V2025_10 {
		t.Errorf("schema = %s, want %s", tokens[0].SchemaVersion, schema.V2025_10)
	}
}

func TestJSONParser_ParseBlockCommentYAMLFallback(t *testing.T) {
	// This is valid YAML but not valid JSON. It must not be reclassified as
	// JSON merely because a leading block comment is skipped.
	data := []byte(`/* c */ {foo: bar}`)

	tokens, err := parser.NewJSONParser().Parse(data, parser.Options{SkipPositions: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("got %d tokens, want 0", len(tokens))
	}
}

func TestJSONParser_ParseYAMLFlowMapping(t *testing.T) {
	data := []byte("color:\n  primary: {$type: color, $value: '#fff'}\n")

	tokens, err := parser.NewJSONParser().Parse(data, parser.Options{SkipPositions: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("got %d tokens, want 1", len(tokens))
	}
	if tokens[0].Name != "color-primary" {
		t.Errorf("token name = %q, want %q", tokens[0].Name, "color-primary")
	}
	if tokens[0].Value != "#fff" {
		t.Errorf("token value = %q, want %q", tokens[0].Value, "#fff")
	}
}

func TestJSONParser_ParseUTF8BOM(t *testing.T) {
	bom := []byte{0xEF, 0xBB, 0xBF}

	t.Run("plain JSON", func(t *testing.T) {
		payload := []byte(`{"spacing":{"small":{"$type":"number","$value":4}}}`)
		data := append(append([]byte{}, bom...), payload...)

		tokens, err := parser.NewJSONParser().Parse(data, parser.Options{SkipPositions: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tokens) != 1 {
			t.Fatalf("got %d tokens, want 1", len(tokens))
		}
		if tokens[0].Name != "spacing-small" {
			t.Errorf("token name = %q, want %q", tokens[0].Name, "spacing-small")
		}
	})

	t.Run("JSONC with leading comment", func(t *testing.T) {
		payload := []byte("// comment\n{\"spacing\":{\"small\":{\"$type\":\"number\",\"$value\":4}}}")
		data := append(append([]byte{}, bom...), payload...)

		tokens, err := parser.NewJSONParser().Parse(data, parser.Options{SkipPositions: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tokens) != 1 {
			t.Fatalf("got %d tokens, want 1", len(tokens))
		}
		if tokens[0].Name != "spacing-small" {
			t.Errorf("token name = %q, want %q", tokens[0].Name, "spacing-small")
		}
	})
}
