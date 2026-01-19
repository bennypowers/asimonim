/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package convert_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"bennypowers.dev/asimonim/convert"
	"bennypowers.dev/asimonim/parser"
	"bennypowers.dev/asimonim/resolver"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/testutil"
)

func TestSerialize_FlattenSimple(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/convert/flatten", "/test")

	p := parser.NewJSONParser()
	tokens, err := p.ParseFile(mfs, "/test/tokens.json", parser.Options{
		SchemaVersion: schema.Draft,
		SkipPositions: true,
	})
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if err := resolver.ResolveAliases(tokens, schema.Draft); err != nil {
		t.Fatalf("failed to resolve aliases: %v", err)
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.Draft,
		OutputSchema: schema.Draft,
		Flatten:      true,
		Delimiter:    "-",
	})

	got, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}

	expected := testutil.LoadFixtureFile(t, "fixtures/convert/flatten/expected.json")

	// Parse both for comparison (order-independent)
	var gotMap, expectedMap map[string]any
	if err := json.Unmarshal(got, &gotMap); err != nil {
		t.Fatalf("failed to unmarshal got: %v", err)
	}
	if err := json.Unmarshal(expected, &expectedMap); err != nil {
		t.Fatalf("failed to unmarshal expected: %v", err)
	}

	// Deep compare the maps
	if !reflect.DeepEqual(expectedMap, gotMap) {
		// Provide detailed diff for debugging
		for key := range expectedMap {
			if _, ok := gotMap[key]; !ok {
				t.Errorf("expected key %q not found in result", key)
			} else if !reflect.DeepEqual(expectedMap[key], gotMap[key]) {
				t.Errorf("value mismatch for key %q:\n  expected: %v\n  got: %v", key, expectedMap[key], gotMap[key])
			}
		}
		for key := range gotMap {
			if _, ok := expectedMap[key]; !ok {
				t.Errorf("unexpected key %q in result", key)
			}
		}
	}
}

func TestSerialize_NestedPreserve(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/convert/flatten", "/test")

	p := parser.NewJSONParser()
	tokens, err := p.ParseFile(mfs, "/test/tokens.json", parser.Options{
		SchemaVersion: schema.Draft,
		SkipPositions: true,
	})
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if err := resolver.ResolveAliases(tokens, schema.Draft); err != nil {
		t.Fatalf("failed to resolve aliases: %v", err)
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.Draft,
		OutputSchema: schema.Draft,
		Flatten:      false,
	})

	// Should have nested structure with "color" and "spacing" groups
	if _, ok := result["color"]; !ok {
		t.Error("expected 'color' group in nested result")
	}
	if _, ok := result["spacing"]; !ok {
		t.Error("expected 'spacing' group in nested result")
	}

	// Should NOT have flattened keys
	if _, ok := result["color-primary"]; ok {
		t.Error("unexpected flattened key 'color-primary' in nested result")
	}
}

func TestSerialize_DraftToV2025(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/convert/draft-to-stable", "/test")

	p := parser.NewJSONParser()
	tokens, err := p.ParseFile(mfs, "/test/tokens.json", parser.Options{
		SchemaVersion: schema.Draft,
		SkipPositions: true,
	})
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if err := resolver.ResolveAliases(tokens, schema.Draft); err != nil {
		t.Fatalf("failed to resolve aliases: %v", err)
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.Draft,
		OutputSchema: schema.V2025_10,
	})

	// Check $schema is present
	schemaURL, ok := result["$schema"].(string)
	if !ok {
		t.Error("expected $schema field in v2025_10 output")
	}
	if schemaURL != schema.V2025_10.URL() {
		t.Errorf("expected schema URL %s, got %s", schema.V2025_10.URL(), schemaURL)
	}

	// Check color structure - should be nested under color > primary
	colorGroup, ok := result["color"].(map[string]any)
	if !ok {
		t.Fatal("expected 'color' group in result")
	}

	primary, ok := colorGroup["primary"].(map[string]any)
	if !ok {
		t.Fatal("expected 'primary' token in color group")
	}

	// Primary color should have structured value
	value, ok := primary["$value"].(map[string]any)
	if !ok {
		t.Error("expected structured color value for primary")
	} else {
		if _, ok := value["colorSpace"].(string); !ok {
			t.Error("expected colorSpace in structured color")
		}
		if _, ok := value["components"].([]any); !ok {
			t.Error("expected components in structured color")
		}
	}

	// Secondary should have $ref
	secondary, ok := colorGroup["secondary"].(map[string]any)
	if !ok {
		t.Fatal("expected 'secondary' token in color group")
	}

	secValue, ok := secondary["$value"].(map[string]any)
	if !ok {
		t.Fatal("expected $value in secondary token")
	}

	ref, ok := secValue["$ref"].(string)
	if !ok {
		t.Error("expected $ref in secondary value")
	} else if ref != "#/color/primary" {
		t.Errorf("expected $ref '#/color/primary', got %s", ref)
	}
}

func TestSerialize_V2025ToDraft(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/convert/stable-to-draft", "/test")

	p := parser.NewJSONParser()
	tokens, err := p.ParseFile(mfs, "/test/tokens.json", parser.Options{
		SchemaVersion: schema.V2025_10,
		SkipPositions: true,
	})
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if err := resolver.ResolveAliases(tokens, schema.V2025_10); err != nil {
		t.Fatalf("failed to resolve aliases: %v", err)
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.V2025_10,
		OutputSchema: schema.Draft,
	})

	// Should NOT have $schema field
	if _, ok := result["$schema"]; ok {
		t.Error("unexpected $schema field in draft output")
	}

	// Check color structure
	colorGroup, ok := result["color"].(map[string]any)
	if !ok {
		t.Fatal("expected 'color' group in result")
	}

	brandGroup, ok := colorGroup["brand"].(map[string]any)
	if !ok {
		t.Fatal("expected 'brand' group in color group")
	}

	primary, ok := brandGroup["primary"].(map[string]any)
	if !ok {
		t.Fatal("expected 'primary' token")
	}

	// Primary should have string value (converted from structured)
	value, ok := primary["$value"].(string)
	if !ok {
		t.Error("expected string color value for primary")
	} else if value != "#FF6B35" {
		t.Errorf("expected '#FF6B35', got %s", value)
	}

	// Secondary should have curly brace reference
	secondary, ok := brandGroup["secondary"].(map[string]any)
	if !ok {
		t.Fatal("expected 'secondary' token")
	}

	secValue, ok := secondary["$value"].(string)
	if !ok {
		t.Error("expected string value for secondary")
	} else if secValue != "{color.brand.primary}" {
		t.Errorf("expected '{color.brand.primary}', got %s", secValue)
	}
}

func TestSerialize_CombineFiles(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/convert/combine", "/test")

	p := parser.NewJSONParser()

	// Parse first file
	tokens1, err := p.ParseFile(mfs, "/test/colors.json", parser.Options{
		SchemaVersion: schema.Draft,
		SkipPositions: true,
	})
	if err != nil {
		t.Fatalf("failed to parse colors.json: %v", err)
	}

	// Parse second file
	tokens2, err := p.ParseFile(mfs, "/test/spacing.json", parser.Options{
		SchemaVersion: schema.Draft,
		SkipPositions: true,
	})
	if err != nil {
		t.Fatalf("failed to parse spacing.json: %v", err)
	}

	// Combine tokens
	allTokens := append(tokens1, tokens2...)
	if err := resolver.ResolveAliases(allTokens, schema.Draft); err != nil {
		t.Fatalf("failed to resolve aliases: %v", err)
	}

	result := convert.Serialize(allTokens, convert.Options{
		InputSchema:  schema.Draft,
		OutputSchema: schema.Draft,
	})

	// Should have both groups
	if _, ok := result["color"]; !ok {
		t.Error("expected 'color' group in combined result")
	}
	if _, ok := result["spacing"]; !ok {
		t.Error("expected 'spacing' group in combined result")
	}
}

func TestSerialize_CustomDelimiter(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/convert/flatten", "/test")

	p := parser.NewJSONParser()
	tokens, err := p.ParseFile(mfs, "/test/tokens.json", parser.Options{
		SchemaVersion: schema.Draft,
		SkipPositions: true,
	})
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if err := resolver.ResolveAliases(tokens, schema.Draft); err != nil {
		t.Fatalf("failed to resolve aliases: %v", err)
	}

	// Use underscore as delimiter
	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.Draft,
		OutputSchema: schema.Draft,
		Flatten:      true,
		Delimiter:    "_",
	})

	// Check that keys use underscore delimiter
	if _, ok := result["color_primary"]; !ok {
		t.Error("expected 'color_primary' key with underscore delimiter")
	}
	if _, ok := result["spacing_small"]; !ok {
		t.Error("expected 'spacing_small' key with underscore delimiter")
	}

	// Should NOT have hyphen-separated keys
	if _, ok := result["color-primary"]; ok {
		t.Error("unexpected 'color-primary' key with underscore delimiter")
	}
}

func TestSerialize_BasicDraftRoundtrip(t *testing.T) {
	// Test that basic tokens roundtrip through serialization unchanged
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")

	p := parser.NewJSONParser()
	tokens, err := p.ParseFile(mfs, "/test/tokens.json", parser.Options{
		SchemaVersion: schema.Draft,
		SkipPositions: true,
	})
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if err := resolver.ResolveAliases(tokens, schema.Draft); err != nil {
		t.Fatalf("failed to resolve aliases: %v", err)
	}

	// Serialize with same schema - values should pass through
	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.Draft,
		OutputSchema: schema.Draft,
	})

	// Basic check that serialization works
	if result == nil {
		t.Error("expected non-nil result")
	}

	// Should have color group
	colorGroup, ok := result["color"].(map[string]any)
	if !ok {
		t.Fatal("expected 'color' group")
	}

	// Verify primary token structure preserved
	primary, ok := colorGroup["primary"].(map[string]any)
	if !ok {
		t.Fatal("expected 'primary' token in color group")
	}
	if primary["$value"] != "#FF6B35" {
		t.Errorf("expected primary $value '#FF6B35', got %v", primary["$value"])
	}
	if primary["$description"] != "Primary brand color" {
		t.Errorf("expected primary $description 'Primary brand color', got %v", primary["$description"])
	}

	// Should have spacing group
	if _, ok := result["spacing"]; !ok {
		t.Error("expected 'spacing' group")
	}
}

func TestSerialize_DefaultOptions(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")

	p := parser.NewJSONParser()
	tokens, err := p.ParseFile(mfs, "/test/tokens.json", parser.Options{
		SchemaVersion: schema.Draft,
		SkipPositions: true,
	})
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Test DefaultOptions returns sensible defaults
	opts := convert.DefaultOptions()
	if opts.InputSchema != schema.Draft {
		t.Errorf("expected Draft input schema, got %v", opts.InputSchema)
	}
	if opts.OutputSchema != schema.Unknown {
		t.Errorf("expected Unknown output schema, got %v", opts.OutputSchema)
	}
	if opts.Flatten {
		t.Error("expected Flatten to be false by default")
	}
	if opts.Delimiter != "-" {
		t.Errorf("expected '-' delimiter, got %q", opts.Delimiter)
	}

	// Serialize with default options should work
	result := convert.Serialize(tokens, opts)
	if result == nil {
		t.Error("expected non-nil result with default options")
	}
}
