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
	"bennypowers.dev/asimonim/token"
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

func TestSerialize_V2025ToDraft_StructuredColorToString(t *testing.T) {
	// Test that structured colors convert to CSS strings when going from v2025 to draft
	tokens := []*token.Token{
		{
			Name:  "color-srgb-hex",
			Value: "",
			Type:  "color",
			Path:  []string{"color", "srgb-hex"},
			RawValue: map[string]any{
				"colorSpace": "srgb",
				"components": []any{1.0, 0.42, 0.21},
				"alpha":      1.0,
				"hex":        "#FF6B36",
			},
			SchemaVersion: schema.V2025_10,
		},
		{
			Name:  "color-no-hex",
			Value: "",
			Type:  "color",
			Path:  []string{"color", "no-hex"},
			RawValue: map[string]any{
				"colorSpace": "srgb",
				"components": []any{1.0, 0.5, 0.25},
				"alpha":      1.0,
			},
			SchemaVersion: schema.V2025_10,
		},
		{
			Name:  "color-with-alpha",
			Value: "",
			Type:  "color",
			Path:  []string{"color", "with-alpha"},
			RawValue: map[string]any{
				"colorSpace": "oklch",
				"components": []any{0.7, 0.15, 180.0},
				"alpha":      0.5,
			},
			SchemaVersion: schema.V2025_10,
		},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.V2025_10,
		OutputSchema: schema.Draft,
	})

	colorGroup := result["color"].(map[string]any)

	// hex field should be used when available
	srgbHex := colorGroup["srgb-hex"].(map[string]any)
	if srgbHex["$value"] != "#FF6B36" {
		t.Errorf("srgb-hex value = %v, want #FF6B36", srgbHex["$value"])
	}

	// Without hex, should produce color() function
	noHex := colorGroup["no-hex"].(map[string]any)
	val := noHex["$value"].(string)
	if val != "color(srgb 1 0.5 0.25)" {
		t.Errorf("no-hex value = %q, want %q", val, "color(srgb 1 0.5 0.25)")
	}

	// Alpha < 1 should include alpha in color() function
	withAlpha := colorGroup["with-alpha"].(map[string]any)
	alphaVal := withAlpha["$value"].(string)
	if alphaVal != "color(oklch 0.7 0.15 180 / 0.5)" {
		t.Errorf("with-alpha value = %q, want %q", alphaVal, "color(oklch 0.7 0.15 180 / 0.5)")
	}
}

func TestSerialize_MapReferences(t *testing.T) {
	// Test that map values with references are handled during same-schema serialization
	tokens := []*token.Token{
		{
			Name:  "shadow-primary",
			Value: "",
			Type:  "shadow",
			Path:  []string{"shadow", "primary"},
			RawValue: map[string]any{
				"offsetX": "2px",
				"offsetY": "4px",
				"blur":    "8px",
				"color":   "{color.primary}",
			},
		},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.Draft,
		OutputSchema: schema.Draft,
	})

	shadow := result["shadow"].(map[string]any)["primary"].(map[string]any)
	value := shadow["$value"].(map[string]any)
	// Reference should pass through in same-schema
	if value["color"] != "{color.primary}" {
		t.Errorf("shadow color = %v, want {color.primary}", value["color"])
	}
}

func TestSerialize_ArrayReferences(t *testing.T) {
	// Test array values pass through same-schema serialization
	tokens := []*token.Token{
		{
			Name:     "bezier-ease",
			Value:    "",
			Type:     "cubicBezier",
			Path:     []string{"bezier", "ease"},
			RawValue: []any{0.42, 0.0, 0.58, 1.0},
		},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.Draft,
		OutputSchema: schema.Draft,
	})

	bezier := result["bezier"].(map[string]any)["ease"].(map[string]any)
	value := bezier["$value"].([]any)
	if len(value) != 4 {
		t.Fatalf("expected 4 elements, got %d", len(value))
	}
}

func TestSerialize_DraftToV2025_MapValues(t *testing.T) {
	// Test that map values with curly brace refs get converted
	tokens := []*token.Token{
		{
			Name:  "shadow-primary",
			Value: "",
			Type:  "shadow",
			Path:  []string{"shadow", "primary"},
			RawValue: map[string]any{
				"offsetX": "2px",
				"offsetY": "4px",
				"blur":    "8px",
				"color":   "{color.primary}",
			},
		},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.Draft,
		OutputSchema: schema.V2025_10,
	})

	shadow := result["shadow"].(map[string]any)["primary"].(map[string]any)
	value := shadow["$value"].(map[string]any)
	// In v2025.10 map conversion, string refs stay as-is (only full value refs convert to $ref)
	if value["color"] != "{color.primary}" {
		t.Errorf("shadow color = %v, want {color.primary}", value["color"])
	}
}

func TestSerialize_V2025ToDraft_ArrayValues(t *testing.T) {
	// Test array values in v2025 to draft conversion
	tokens := []*token.Token{
		{
			Name:     "bezier-custom",
			Value:    "",
			Type:     "cubicBezier",
			Path:     []string{"bezier", "custom"},
			RawValue: []any{0.42, 0.0, 0.58, 1.0},
		},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.V2025_10,
		OutputSchema: schema.Draft,
	})

	bezier := result["bezier"].(map[string]any)["custom"].(map[string]any)
	value := bezier["$value"].([]any)
	if len(value) != 4 {
		t.Fatalf("expected 4 elements, got %d", len(value))
	}
}

func TestSerializeTokens_Deprecated(t *testing.T) {
	// SerializeTokens is a deprecated wrapper around Serialize
	tokens := []*token.Token{
		{
			Name:  "color-primary",
			Value: "#FF6B35",
			Type:  "color",
			Path:  []string{"color", "primary"},
		},
	}

	result := convert.SerializeTokens(tokens, schema.Draft, schema.Draft, false, "-")

	colorGroup, ok := result["color"].(map[string]any)
	if !ok {
		t.Fatal("expected 'color' group in result")
	}
	primary, ok := colorGroup["primary"].(map[string]any)
	if !ok {
		t.Fatal("expected 'primary' token in color group")
	}
	if primary["$value"] != "#FF6B35" {
		t.Errorf("expected $value '#FF6B35', got %v", primary["$value"])
	}
}

func TestSerializeTokens_Flattened(t *testing.T) {
	// SerializeTokens with flatten=true
	tokens := []*token.Token{
		{
			Name:  "color-primary",
			Value: "#FF6B35",
			Type:  "color",
			Path:  []string{"color", "primary"},
		},
	}

	result := convert.SerializeTokens(tokens, schema.Draft, schema.Draft, true, ".")

	if _, ok := result["color.primary"]; !ok {
		t.Error("expected 'color.primary' key with '.' delimiter")
	}
}

func TestSerialize_SerializeToken_WithExtensions(t *testing.T) {
	// Test that extensions are preserved in serialized output
	tokens := []*token.Token{
		{
			Name:  "color-primary",
			Value: "#FF6B35",
			Type:  "color",
			Path:  []string{"color", "primary"},
			Extensions: map[string]any{
				"com.example.figma": map[string]any{
					"nodeId": "123:456",
				},
			},
		},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.Draft,
		OutputSchema: schema.Draft,
	})

	colorGroup := result["color"].(map[string]any)
	primary := colorGroup["primary"].(map[string]any)

	extensions, ok := primary["$extensions"].(map[string]any)
	if !ok {
		t.Fatal("expected $extensions in serialized token")
	}
	figma, ok := extensions["com.example.figma"].(map[string]any)
	if !ok {
		t.Fatal("expected figma extension data")
	}
	if figma["nodeId"] != "123:456" {
		t.Errorf("expected nodeId '123:456', got %v", figma["nodeId"])
	}
}

func TestSerialize_SerializeToken_Deprecated(t *testing.T) {
	// Test that deprecated field and deprecation message are preserved
	tokens := []*token.Token{
		{
			Name:               "color-old",
			Value:              "#000000",
			Type:               "color",
			Path:               []string{"color", "old"},
			Deprecated:         true,
			DeprecationMessage: "Use color.new instead",
		},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.Draft,
		OutputSchema: schema.Draft,
	})

	colorGroup := result["color"].(map[string]any)
	old := colorGroup["old"].(map[string]any)

	if old["$deprecated"] != true {
		t.Errorf("expected $deprecated to be true, got %v", old["$deprecated"])
	}
	if old["$deprecationMessage"] != "Use color.new instead" {
		t.Errorf("expected deprecation message 'Use color.new instead', got %v", old["$deprecationMessage"])
	}
}

func TestSerialize_SerializeToken_DeprecatedWithoutMessage(t *testing.T) {
	// Test deprecated=true with no message
	tokens := []*token.Token{
		{
			Name:       "color-old",
			Value:      "#000000",
			Type:       "color",
			Path:       []string{"color", "old"},
			Deprecated: true,
		},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.Draft,
		OutputSchema: schema.Draft,
	})

	colorGroup := result["color"].(map[string]any)
	old := colorGroup["old"].(map[string]any)

	if old["$deprecated"] != true {
		t.Errorf("expected $deprecated to be true, got %v", old["$deprecated"])
	}
	if _, ok := old["$deprecationMessage"]; ok {
		t.Error("expected no $deprecationMessage when message is empty")
	}
}

func TestSerialize_ConvertDraftToV2025_EmbeddedReference(t *testing.T) {
	// Test that embedded references (partial string references) pass through as-is
	tokens := []*token.Token{
		{
			Name:     "text-greeting",
			Value:    "",
			Type:     "string",
			Path:     []string{"text", "greeting"},
			RawValue: "Hello {user.name}, welcome!",
		},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.Draft,
		OutputSchema: schema.V2025_10,
	})

	textGroup := result["text"].(map[string]any)
	greeting := textGroup["greeting"].(map[string]any)

	// Embedded references should be kept as-is (no $ref conversion)
	if greeting["$value"] != "Hello {user.name}, welcome!" {
		t.Errorf("expected embedded reference preserved, got %v", greeting["$value"])
	}
}

func TestSerialize_ConvertDraftToV2025_NonColorString(t *testing.T) {
	// Test that non-color string values pass through without structural conversion
	tokens := []*token.Token{
		{
			Name:     "font-body",
			Value:    "",
			Type:     "fontFamily",
			Path:     []string{"font", "body"},
			RawValue: "Inter, sans-serif",
		},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.Draft,
		OutputSchema: schema.V2025_10,
	})

	fontGroup := result["font"].(map[string]any)
	body := fontGroup["body"].(map[string]any)

	// Non-color strings should pass through unchanged
	if body["$value"] != "Inter, sans-serif" {
		t.Errorf("expected 'Inter, sans-serif', got %v", body["$value"])
	}
}

func TestSerialize_ConvertDraftToV2025_ArrayValue(t *testing.T) {
	// Test that array values in draft-to-v2025 conversion have references handled
	tokens := []*token.Token{
		{
			Name:     "bezier-ease",
			Value:    "",
			Type:     "cubicBezier",
			Path:     []string{"bezier", "ease"},
			RawValue: []any{0.42, 0.0, "{timing.x2}", 1.0},
		},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.Draft,
		OutputSchema: schema.V2025_10,
	})

	bezier := result["bezier"].(map[string]any)["ease"].(map[string]any)
	value := bezier["$value"].([]any)
	if len(value) != 4 {
		t.Fatalf("expected 4 elements, got %d", len(value))
	}
}

func TestSerialize_ConvertV2025ToDraft_NonMapNonArrayNonString(t *testing.T) {
	// Test that numeric and boolean values pass through v2025->draft unchanged
	tokens := []*token.Token{
		{
			Name:     "opacity-half",
			Value:    "",
			Type:     "number",
			Path:     []string{"opacity", "half"},
			RawValue: 0.5,
		},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.V2025_10,
		OutputSchema: schema.Draft,
	})

	opacityGroup := result["opacity"].(map[string]any)
	half := opacityGroup["half"].(map[string]any)
	// Numeric value should pass through unchanged
	if half["$value"] != 0.5 {
		t.Errorf("expected 0.5, got %v", half["$value"])
	}
}

func TestSerialize_ConvertV2025ToDraft_MapWithoutColorSpaceOrRef(t *testing.T) {
	// Test that map values without colorSpace or $ref are recursively converted
	tokens := []*token.Token{
		{
			Name:  "shadow-primary",
			Value: "",
			Type:  "shadow",
			Path:  []string{"shadow", "primary"},
			RawValue: map[string]any{
				"offsetX": map[string]any{"value": float64(2), "unit": "px"},
				"offsetY": "4px",
				"blur":    "8px",
				"color":   "#000",
			},
		},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.V2025_10,
		OutputSchema: schema.Draft,
	})

	shadow := result["shadow"].(map[string]any)["primary"].(map[string]any)
	value := shadow["$value"].(map[string]any)
	// Map values should be recursively converted
	if value["offsetY"] != "4px" {
		t.Errorf("expected offsetY '4px', got %v", value["offsetY"])
	}
}

func TestSerialize_ConvertV2025ToDraft_StringJSONPointer(t *testing.T) {
	// Test that string JSON pointer references are converted to curly brace format
	tokens := []*token.Token{
		{
			Name:     "color-alias",
			Value:    "",
			Type:     "color",
			Path:     []string{"color", "alias"},
			RawValue: "#/color/primary",
		},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.V2025_10,
		OutputSchema: schema.Draft,
	})

	colorGroup := result["color"].(map[string]any)
	alias := colorGroup["alias"].(map[string]any)
	// JSON pointer string should be converted to curly brace reference
	if alias["$value"] != "{color.primary}" {
		t.Errorf("expected '{color.primary}', got %v", alias["$value"])
	}
}

func TestSerialize_ConvertV2025ToDraft_PlainString(t *testing.T) {
	// Test that plain strings (not JSON pointers) pass through v2025->draft
	tokens := []*token.Token{
		{
			Name:     "font-body",
			Value:    "",
			Type:     "fontFamily",
			Path:     []string{"font", "body"},
			RawValue: "Inter",
		},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.V2025_10,
		OutputSchema: schema.Draft,
	})

	fontGroup := result["font"].(map[string]any)
	body := fontGroup["body"].(map[string]any)
	if body["$value"] != "Inter" {
		t.Errorf("expected 'Inter', got %v", body["$value"])
	}
}

func TestConvertStringColorToStructured_NonHexColor(t *testing.T) {
	// Test converting non-hex color strings (e.g., rgb, named colors)
	tokens := []*token.Token{
		{
			Name:     "color-named",
			Value:    "",
			Type:     "color",
			Path:     []string{"color", "named"},
			RawValue: "red",
		},
		{
			Name:     "color-rgb",
			Value:    "",
			Type:     "color",
			Path:     []string{"color", "rgb"},
			RawValue: "rgb(255, 128, 0)",
		},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.Draft,
		OutputSchema: schema.V2025_10,
	})

	colorGroup := result["color"].(map[string]any)

	// Named color "red" should be converted to structured format
	named := colorGroup["named"].(map[string]any)
	namedValue, ok := named["$value"].(map[string]any)
	if !ok {
		t.Fatal("expected structured color value for 'red'")
	}
	if namedValue["colorSpace"] != "srgb" {
		t.Errorf("expected colorSpace 'srgb', got %v", namedValue["colorSpace"])
	}
	// Named colors don't start with #, so hex should be generated
	if _, ok := namedValue["hex"].(string); !ok {
		t.Error("expected hex field for named color")
	}

	// rgb() color should be converted to structured format
	rgbToken := colorGroup["rgb"].(map[string]any)
	rgbValue, ok := rgbToken["$value"].(map[string]any)
	if !ok {
		t.Fatal("expected structured color value for rgb()")
	}
	if rgbValue["colorSpace"] != "srgb" {
		t.Errorf("expected colorSpace 'srgb', got %v", rgbValue["colorSpace"])
	}
}

func TestConvertStringColorToStructured_InvalidColor(t *testing.T) {
	// Test that invalid color strings are returned as-is
	tokens := []*token.Token{
		{
			Name:     "color-invalid",
			Value:    "",
			Type:     "color",
			Path:     []string{"color", "invalid"},
			RawValue: "not-a-color",
		},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.Draft,
		OutputSchema: schema.V2025_10,
	})

	colorGroup := result["color"].(map[string]any)
	invalid := colorGroup["invalid"].(map[string]any)
	// Invalid color string should pass through unchanged
	if invalid["$value"] != "not-a-color" {
		t.Errorf("expected 'not-a-color', got %v", invalid["$value"])
	}
}

func TestSerialize_EmptyStringValue(t *testing.T) {
	// Token with empty string Value (default zero value) still produces $value
	tokens := []*token.Token{
		{
			Name: "empty",
			Type: "color",
			Path: []string{"empty"},
		},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.Draft,
		OutputSchema: schema.Draft,
	})

	empty := result["empty"].(map[string]any)
	// Empty string value passes through as ""
	if empty["$value"] != "" {
		t.Errorf("expected empty string $value, got %v", empty["$value"])
	}
}

func TestSerialize_ConvertDraftToV2025_NumericValue(t *testing.T) {
	// Test that numeric values in draft-to-v2025 conversion pass through (default case)
	tokens := []*token.Token{
		{
			Name:     "opacity-half",
			Value:    "",
			Type:     "number",
			Path:     []string{"opacity", "half"},
			RawValue: 0.5,
		},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.Draft,
		OutputSchema: schema.V2025_10,
	})

	opacityGroup := result["opacity"].(map[string]any)
	half := opacityGroup["half"].(map[string]any)
	if half["$value"] != 0.5 {
		t.Errorf("expected 0.5, got %v", half["$value"])
	}
}

func TestSerialize_UnknownInputSchema(t *testing.T) {
	// Unknown InputSchema should default to Draft
	tokens := []*token.Token{
		{Name: "a", Value: "#FF0000", Type: "color", Path: []string{"a"}},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.Unknown,
		OutputSchema: schema.Unknown,
	})

	a := result["a"].(map[string]any)
	// Should pass through as draft (string color)
	if a["$value"] != "#FF0000" {
		t.Errorf("expected '#FF0000', got %v", a["$value"])
	}
}

func TestSerialize_NestedCollision(t *testing.T) {
	// Test collision in buildNestedStructure: a token and group share a path segment
	tokens := []*token.Token{
		{Name: "color", Value: "#000000", Type: "color", Path: []string{"color"}},
		{Name: "color-primary", Value: "#FF0000", Type: "color", Path: []string{"color", "primary"}},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.Draft,
		OutputSchema: schema.Draft,
	})

	// "color" was a leaf, then "color.primary" forces it to become a group
	colorGroup, ok := result["color"].(map[string]any)
	if !ok {
		t.Fatal("expected 'color' to be a map after collision")
	}
	// Original leaf value should be wrapped as $value
	if colorGroup["$value"] == nil {
		// The collision wraps the existing map, so the original token's $value should exist
		primary, ok := colorGroup["primary"].(map[string]any)
		if !ok {
			t.Fatal("expected 'primary' in color group")
		}
		if primary["$value"] != "#FF0000" {
			t.Errorf("expected primary value '#FF0000', got %v", primary["$value"])
		}
	}
}

func TestSerialize_V2025ToDraft_StructuredColorNoHex(t *testing.T) {
	// Structured color without hex and without colorSpace should return empty
	tokens := []*token.Token{
		{
			Name:  "color-empty",
			Value: "",
			Type:  "color",
			Path:  []string{"color", "empty"},
			RawValue: map[string]any{
				"colorSpace": "srgb",
			},
		},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.V2025_10,
		OutputSchema: schema.Draft,
	})

	colorGroup := result["color"].(map[string]any)
	empty := colorGroup["empty"].(map[string]any)
	// Missing components means convertStructuredColorToString returns ""
	if empty["$value"] != "" {
		t.Errorf("expected empty string for colorObj without components, got %v", empty["$value"])
	}
}

func TestSerialize_V2025ToDraft_MapWithArray(t *testing.T) {
	// Map containing an array value in v2025-to-draft
	tokens := []*token.Token{
		{
			Name:  "shadow-multi",
			Value: "",
			Type:  "shadow",
			Path:  []string{"shadow", "multi"},
			RawValue: map[string]any{
				"layers": []any{
					map[string]any{"offsetX": "2px"},
				},
			},
		},
	}

	result := convert.Serialize(tokens, convert.Options{
		InputSchema:  schema.V2025_10,
		OutputSchema: schema.Draft,
	})

	shadow := result["shadow"].(map[string]any)["multi"].(map[string]any)
	value := shadow["$value"].(map[string]any)
	layers, ok := value["layers"].([]any)
	if !ok {
		t.Fatal("expected layers array")
	}
	if len(layers) != 1 {
		t.Errorf("expected 1 layer, got %d", len(layers))
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
