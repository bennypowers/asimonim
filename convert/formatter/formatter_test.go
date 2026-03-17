/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package formatter_test

import (
	"strings"
	"testing"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/token"
)

func TestFormatHeader_Empty(t *testing.T) {
	result := formatter.FormatHeader("", formatter.CStyleComments)
	if result != "" {
		t.Errorf("expected empty string for empty header, got %q", result)
	}
}

func TestFormatHeader_SingleLine_WithLinePrefix(t *testing.T) {
	result := formatter.FormatHeader("Copyright 2026", formatter.SCSSComments)
	expected := "// Copyright 2026\n\n"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatHeader_MultiLine_CStyle(t *testing.T) {
	result := formatter.FormatHeader("Copyright 2026\nMIT License", formatter.CStyleComments)
	if !strings.HasPrefix(result, "/*\n") {
		t.Error("expected C-style block comment start")
	}
	if !strings.Contains(result, " * Copyright 2026\n") {
		t.Error("expected line with asterisk prefix")
	}
	if !strings.Contains(result, " * MIT License\n") {
		t.Error("expected second line with asterisk prefix")
	}
	if !strings.HasSuffix(result, "*/\n\n") {
		t.Error("expected block comment end")
	}
}

func TestFormatHeader_MultiLine_XML(t *testing.T) {
	result := formatter.FormatHeader("Copyright 2026\nMIT License", formatter.XMLComments)
	if !strings.HasPrefix(result, "<!--\n") {
		t.Error("expected XML comment start")
	}
	if !strings.Contains(result, "  Copyright 2026\n") {
		t.Error("expected line with XML-style indent")
	}
	if !strings.Contains(result, "  MIT License\n") {
		t.Error("expected second line with XML-style indent")
	}
	if !strings.HasSuffix(result, "-->\n\n") {
		t.Error("expected XML comment end")
	}
}

func TestFormatHeader_TrailingNewlines(t *testing.T) {
	result := formatter.FormatHeader("Copyright 2026\n\n\n", formatter.SCSSComments)
	expected := "// Copyright 2026\n\n"
	if result != expected {
		t.Errorf("expected trailing newlines to be trimmed, got %q", result)
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"color-primary", "colorPrimary"},
		{"color_primary", "colorPrimary"},
		{"color.primary", "colorPrimary"},
		{"ColorPrimary", "colorPrimary"},
		{"color-primary-dark", "colorPrimaryDark"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatter.ToCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToCamelCase(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"colorPrimary", "color-primary"},
		{"color_primary", "color-primary"},
		{"color.primary", "color-primary"},
		{"color-primary", "color-primary"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatter.ToKebabCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToKebabCase(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"colorPrimary", "color_primary"},
		{"color-primary", "color_primary"},
		{"color.primary", "color_primary"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatter.ToSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToSnakeCase(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestApplyPrefix(t *testing.T) {
	tests := []struct {
		name      string
		prefix    string
		delimiter string
		expected  string
	}{
		{"color-primary", "", "-", "color-primary"},
		{"color-primary", "rh", "-", "rh-color-primary"},
		{"color_primary", "rh", "_", "rh_color_primary"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_"+tt.prefix, func(t *testing.T) {
			result := formatter.ApplyPrefix(tt.name, tt.prefix, tt.delimiter)
			if result != tt.expected {
				t.Errorf("ApplyPrefix(%q, %q, %q) = %q, expected %q",
					tt.name, tt.prefix, tt.delimiter, result, tt.expected)
			}
		})
	}
}

func TestGroupByType(t *testing.T) {
	tokens := []*token.Token{
		{Name: "color-primary", Type: "color"},
		{Name: "color-secondary", Type: "color"},
		{Name: "spacing-small", Type: "dimension"},
		{Name: "font-body", Type: "fontFamily"},
	}

	groups := formatter.GroupByType(tokens)

	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}
	if len(groups["color"]) != 2 {
		t.Errorf("expected 2 color tokens, got %d", len(groups["color"]))
	}
	if len(groups["dimension"]) != 1 {
		t.Errorf("expected 1 dimension token, got %d", len(groups["dimension"]))
	}
	if len(groups["fontFamily"]) != 1 {
		t.Errorf("expected 1 fontFamily token, got %d", len(groups["fontFamily"]))
	}
}

func TestApplyPrefixCamel(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		expected string
	}{
		// empty prefix returns name unchanged
		{"colorPrimary", "", "colorPrimary"},
		// empty name returns camelCase prefix
		{"", "my-prefix", "myPrefix"},
		// normal case: prefix + capitalized name
		{"colorPrimary", "rh", "rhColorPrimary"},
		// hyphenated prefix
		{"colorPrimary", "my-app", "myAppColorPrimary"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_"+tt.prefix, func(t *testing.T) {
			result := formatter.ApplyPrefixCamel(tt.name, tt.prefix)
			if result != tt.expected {
				t.Errorf("ApplyPrefixCamel(%q, %q) = %q, expected %q",
					tt.name, tt.prefix, result, tt.expected)
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"color-primary", "ColorPrimary"},
		{"color_primary", "ColorPrimary"},
		{"color.primary", "ColorPrimary"},
		{"colorPrimary", "ColorPrimary"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatter.ToPascalCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToPascalCase(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToTitleCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"color-primary", "Color Primary"},
		{"color_primary", "Color Primary"},
		{"colorPrimary", "Color Primary"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatter.ToTitleCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToTitleCase(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSplitIntoWords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		// consecutive separators produce no empty words
		{"consecutive hyphens", "color--primary", []string{"color", "primary"}},
		// leading separator
		{"leading hyphen", "-color", []string{"color"}},
		// trailing separator
		{"trailing hyphen", "color-", []string{"color"}},
		// mixed separators
		{"mixed separators", "color-_primary.dark", []string{"color", "primary", "dark"}},
		// spaces as separators
		{"spaces", "color primary", []string{"color", "primary"}},
		// single word
		{"single word", "color", []string{"color"}},
		// empty string
		{"empty", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.SplitIntoWords(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("SplitIntoWords(%q) = %v (len %d), expected %v (len %d)",
					tt.input, result, len(result), tt.expected, len(tt.expected))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("SplitIntoWords(%q)[%d] = %q, expected %q",
						tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestMarshalFallback(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected string
	}{
		{
			"simple map",
			map[string]any{"color": "#ff0000"},
			`{"color":"#ff0000"}`,
		},
		{
			"nested map",
			map[string]any{"value": float64(4), "unit": "px"},
			`{"unit":"px","value":4}`,
		},
		{
			"empty map",
			map[string]any{},
			`{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.MarshalFallback(tt.input)
			if result != tt.expected {
				t.Errorf("MarshalFallback(%v) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEscapeXML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// ampersand
		{"A&B", "A&amp;B"},
		// less than
		{"A<B", "A&lt;B"},
		// greater than
		{"A>B", "A&gt;B"},
		// double quote
		{`A"B`, "A&quot;B"},
		// single quote
		{"A'B", "A&apos;B"},
		// multiple special characters
		{`<div class="a">&`, `&lt;div class=&quot;a&quot;&gt;&amp;`},
		// no special characters
		{"hello", "hello"},
		// empty string
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatter.EscapeXML(tt.input)
			if result != tt.expected {
				t.Errorf("EscapeXML(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestResolvedValue(t *testing.T) {
	t.Run("nil token", func(t *testing.T) {
		result := formatter.ResolvedValue(nil)
		if result != nil {
			t.Errorf("ResolvedValue(nil) = %v, expected nil", result)
		}
	})

	t.Run("token with ResolvedValue", func(t *testing.T) {
		// ResolvedValue takes precedence over RawValue and Value
		tok := &token.Token{
			Value:         "#000",
			RawValue:      "#111",
			ResolvedValue: "#222",
		}
		result := formatter.ResolvedValue(tok)
		if result != "#222" {
			t.Errorf("ResolvedValue() = %v, expected %q", result, "#222")
		}
	})

	t.Run("token with only RawValue", func(t *testing.T) {
		// RawValue takes precedence over Value when ResolvedValue is nil
		tok := &token.Token{
			Value:    "#000",
			RawValue: "#111",
		}
		result := formatter.ResolvedValue(tok)
		if result != "#111" {
			t.Errorf("ResolvedValue() = %v, expected %q", result, "#111")
		}
	})

	t.Run("token with only Value", func(t *testing.T) {
		// Value is the fallback when both ResolvedValue and RawValue are nil
		tok := &token.Token{
			Value: "#000",
		}
		result := formatter.ResolvedValue(tok)
		if result != "#000" {
			t.Errorf("ResolvedValue() = %v, expected %q", result, "#000")
		}
	})
}

func TestSortTokens(t *testing.T) {
	tokens := []*token.Token{
		{Name: "zebra"},
		{Name: "alpha"},
		{Name: "middle"},
	}

	sorted := formatter.SortTokens(tokens)

	// Verify sorted order
	if sorted[0].Name != "alpha" {
		t.Errorf("sorted[0].Name = %q, expected %q", sorted[0].Name, "alpha")
	}
	if sorted[1].Name != "middle" {
		t.Errorf("sorted[1].Name = %q, expected %q", sorted[1].Name, "middle")
	}
	if sorted[2].Name != "zebra" {
		t.Errorf("sorted[2].Name = %q, expected %q", sorted[2].Name, "zebra")
	}

	// Verify original slice is not modified
	if tokens[0].Name != "zebra" {
		t.Errorf("original tokens[0].Name = %q, expected %q (original should be unchanged)",
			tokens[0].Name, "zebra")
	}
}

func TestFormatHeader_SingleLine_BlockComment(t *testing.T) {
	// XMLComments has no LinePrefix, so a single line uses block comment style
	result := formatter.FormatHeader("Copyright 2026", formatter.XMLComments)
	expected := "<!-- Copyright 2026 -->\n\n"
	if result != expected {
		t.Errorf("FormatHeader single line block comment = %q, expected %q", result, expected)
	}
}
