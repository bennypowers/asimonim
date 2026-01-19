/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package common_test

import (
	"testing"

	"bennypowers.dev/asimonim/parser/common"
)

func TestCurlyBraceRefPattern(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"{color.primary}", []string{"color.primary"}},
		{"{spacing.small}", []string{"spacing.small"}},
		{"prefix {color.primary} suffix", []string{"color.primary"}},
		{"{a} and {b}", []string{"a", "b"}},
		{"no references", nil},
		{"{nested.deep.path.value}", []string{"nested.deep.path.value"}},
		{"{}", nil}, // empty braces don't match
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			matches := common.CurlyBraceRefPattern.FindAllStringSubmatch(tt.input, -1)
			var got []string
			for _, m := range matches {
				if len(m) > 1 && m[1] != "" {
					got = append(got, m[1])
				}
			}
			if len(got) != len(tt.expected) {
				t.Errorf("CurlyBraceRefPattern matches = %v, want %v", got, tt.expected)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("CurlyBraceRefPattern match[%d] = %v, want %v", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestJSONPointerRefPattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"JSON format", `"$ref": "#/color/primary"`, "#/color/primary"},
		{"YAML unquoted key", `$ref: "#/color/primary"`, "#/color/primary"},
		{"YAML single quotes", `$ref: '#/color/primary'`, "#/color/primary"},
		{"with spaces", `"$ref" : "#/path/to/token"`, "#/path/to/token"},
		{"no match", `"value": "something"`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := common.JSONPointerRefPattern.FindStringSubmatch(tt.input)
			got := ""
			if len(matches) > 1 {
				got = matches[1]
			}
			if got != tt.expected {
				t.Errorf("JSONPointerRefPattern = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRootKeywordPattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"JSON format", `"$root": {`, true},
		{"YAML format", `$root:`, true},
		{"with spaces", `"$root" :`, true},
		{"no match", `"value": "something"`, false},
		{"partial match", `root:`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.RootKeywordPattern.MatchString(tt.input)
			if got != tt.expected {
				t.Errorf("RootKeywordPattern.MatchString(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSchemaFieldPattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"JSON format",
			`"$schema": "https://www.designtokens.org/schemas/draft.json"`,
			"https://www.designtokens.org/schemas/draft.json",
		},
		{
			"YAML double quotes",
			`$schema: "https://www.designtokens.org/schemas/2025.10.json"`,
			"https://www.designtokens.org/schemas/2025.10.json",
		},
		{
			"YAML single quotes",
			`$schema: 'https://example.com/schema.json'`,
			"https://example.com/schema.json",
		},
		{
			"with leading whitespace",
			`  "$schema": "https://example.com/schema.json"`,
			"https://example.com/schema.json",
		},
		{
			"no match",
			`"type": "color"`,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := common.SchemaFieldPattern.FindStringSubmatch(tt.input)
			got := ""
			if len(matches) > 1 {
				got = matches[1]
			}
			if got != tt.expected {
				t.Errorf("SchemaFieldPattern = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestValueFieldPattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"JSON format", `"$value": "#fff"`, true},
		{"YAML format", `$value: "#fff"`, true},
		{"with spaces", `"$value" :`, true},
		{"no match", `"value": "something"`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.ValueFieldPattern.MatchString(tt.input)
			if got != tt.expected {
				t.Errorf("ValueFieldPattern.MatchString(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTypeFieldPattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"JSON format", `"$type": "color"`, true},
		{"YAML format", `$type: color`, true},
		{"with spaces", `"$type" :`, true},
		{"no match", `"type": "something"`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.TypeFieldPattern.MatchString(tt.input)
			if got != tt.expected {
				t.Errorf("TypeFieldPattern.MatchString(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtendsFieldPattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"JSON format", `"$extends": "#/base/color"`, "#/base/color"},
		{"YAML unquoted", `$extends: "#/base/token"`, "#/base/token"},
		{"YAML single quotes", `$extends: '#/other'`, "#/other"},
		{"no match", `"extends": "something"`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := common.ExtendsFieldPattern.FindStringSubmatch(tt.input)
			got := ""
			if len(matches) > 1 {
				got = matches[1]
			}
			if got != tt.expected {
				t.Errorf("ExtendsFieldPattern = %v, want %v", got, tt.expected)
			}
		})
	}
}
