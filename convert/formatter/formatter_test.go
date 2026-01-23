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
