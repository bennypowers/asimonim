/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package convert_test

import (
	"slices"
	"strings"
	"testing"

	"bennypowers.dev/asimonim/convert"
	"bennypowers.dev/asimonim/parser"
	"bennypowers.dev/asimonim/resolver"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/testutil"
	"bennypowers.dev/asimonim/token"
)

func loadTestTokens(t *testing.T) []*token.Token {
	t.Helper()
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
	return tokens
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected convert.Format
		wantErr  bool
	}{
		{"dtcg", convert.FormatDTCG, false},
		{"", convert.FormatDTCG, false},
		{"json", convert.FormatFlatJSON, false},
		{"flat", convert.FormatFlatJSON, false},
		{"flat-json", convert.FormatFlatJSON, false},
		{"xml", convert.FormatAndroid, false},
		{"android", convert.FormatAndroid, false},
		{"swift", convert.FormatSwift, false},
		{"ios", convert.FormatSwift, false},
		{"typescript", convert.FormatTypeScript, false},
		{"ts", convert.FormatTypeScript, false},
		{"cts", convert.FormatCTS, false},
		{"commonjs", convert.FormatCTS, false},
		{"scss", convert.FormatSCSS, false},
		{"sass", convert.FormatSCSS, false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := convert.ParseFormat(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestFormatTokens_FlatJSON(t *testing.T) {
	tokens := loadTestTokens(t)
	opts := convert.DefaultOptions()

	output, err := convert.FormatTokens(tokens, convert.FormatFlatJSON, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := string(output)

	// Check for flat key-value structure
	if !strings.Contains(result, `"color-primary"`) {
		t.Error("expected flat key 'color-primary'")
	}
	if !strings.Contains(result, `"spacing-small"`) {
		t.Error("expected flat key 'spacing-small'")
	}
	// Should NOT have nested structure markers
	if strings.Contains(result, `"$value"`) {
		t.Error("flat JSON should not contain $value")
	}
}

func TestFormatTokens_XML(t *testing.T) {
	tokens := loadTestTokens(t)
	opts := convert.DefaultOptions()

	output, err := convert.FormatTokens(tokens, convert.FormatAndroid, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := string(output)

	// Check XML structure
	if !strings.Contains(result, `<?xml version="1.0"`) {
		t.Error("expected XML declaration")
	}
	if !strings.Contains(result, `<resources>`) {
		t.Error("expected resources tag")
	}
	if !strings.Contains(result, `<color name="color_primary">`) {
		t.Error("expected color element with snake_case name")
	}
	if !strings.Contains(result, `<dimen name="spacing_small">`) {
		t.Error("expected dimen element for dimension type")
	}
	if !strings.Contains(result, `</resources>`) {
		t.Error("expected closing resources tag")
	}
}

func TestFormatTokens_Swift(t *testing.T) {
	tokens := loadTestTokens(t)
	opts := convert.DefaultOptions()

	output, err := convert.FormatTokens(tokens, convert.FormatSwift, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := string(output)

	// Check Swift structure
	if !strings.Contains(result, "import SwiftUI") {
		t.Error("expected SwiftUI import")
	}
	if !strings.Contains(result, "public enum DesignTokens") {
		t.Error("expected DesignTokens enum")
	}
	if !strings.Contains(result, "public enum Color") {
		t.Error("expected Color enum for color tokens")
	}
	if !strings.Contains(result, "public enum Dimension") {
		t.Error("expected Dimension enum for dimension tokens")
	}
	if !strings.Contains(result, "Color(.sRGB, red:") {
		t.Error("expected native Color(.sRGB, red:...) for color values")
	}
	if !strings.Contains(result, "CGFloat(") {
		t.Error("expected CGFloat for dimension values")
	}
}

func TestFormatTokens_TypeScript(t *testing.T) {
	tokens := loadTestTokens(t)
	opts := convert.DefaultOptions()

	output, err := convert.FormatTokens(tokens, convert.FormatTypeScript, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := string(output)

	// Check TypeScript structure
	if !strings.Contains(result, "export const") {
		t.Error("expected export const")
	}
	if !strings.Contains(result, "colorPrimary =") {
		t.Error("expected camelCase variable name 'colorPrimary'")
	}
	if !strings.Contains(result, "as const") {
		t.Error("expected 'as const' assertion")
	}
	// Check for JSDoc comment
	if !strings.Contains(result, "/** Primary brand color */") {
		t.Error("expected JSDoc comment for description")
	}
}

func TestFormatTokens_SCSS(t *testing.T) {
	tokens := loadTestTokens(t)
	opts := convert.DefaultOptions()

	output, err := convert.FormatTokens(tokens, convert.FormatSCSS, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := string(output)

	// Check SCSS structure
	if !strings.Contains(result, "$color-primary:") {
		t.Error("expected kebab-case variable name '$color-primary'")
	}
	if !strings.Contains(result, "$spacing-small:") {
		t.Error("expected kebab-case variable name '$spacing-small'")
	}
	if !strings.Contains(result, "#FF6B35") {
		t.Error("expected color value")
	}
	// Check for comment groups
	if !strings.Contains(result, "// Color") {
		t.Error("expected group comment")
	}
}

func TestFormatTokens_DTCG(t *testing.T) {
	tokens := loadTestTokens(t)
	opts := convert.DefaultOptions()

	output, err := convert.FormatTokens(tokens, convert.FormatDTCG, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := string(output)

	// Check DTCG structure
	if !strings.Contains(result, `"$value"`) {
		t.Error("expected $value field")
	}
	if !strings.Contains(result, `"$type"`) {
		t.Error("expected $type field")
	}
	if !strings.Contains(result, `"color"`) {
		t.Error("expected nested color group")
	}
}

func TestFormatTokens_CTS(t *testing.T) {
	tokens := loadTestTokens(t)
	opts := convert.DefaultOptions()

	output, err := convert.FormatTokens(tokens, convert.FormatCTS, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := string(output)

	// Check CommonJS TypeScript structure
	if !strings.Contains(result, "exports.") {
		t.Error("expected exports. for CommonJS")
	}
	if !strings.Contains(result, "colorPrimary =") {
		t.Error("expected camelCase variable name 'colorPrimary'")
	}
	if !strings.Contains(result, "as const") {
		t.Error("expected 'as const' assertion")
	}
	// Check for JSDoc comment
	if !strings.Contains(result, "/** Primary brand color */") {
		t.Error("expected JSDoc comment for description")
	}
}

func TestValidFormats(t *testing.T) {
	formats := convert.ValidFormats()

	expected := []string{"dtcg", "json", "android", "swift", "typescript", "cts", "scss", "typescript-map"}
	if len(formats) != len(expected) {
		t.Errorf("expected %d formats, got %d", len(expected), len(formats))
	}

	for _, exp := range expected {
		if !slices.Contains(formats, exp) {
			t.Errorf("expected format %q not found", exp)
		}
	}
}
