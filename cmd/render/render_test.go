/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package render

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"bennypowers.dev/asimonim/testutil"
	"bennypowers.dev/asimonim/token"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Color Brand", "color-brand"},
		{"color-brand", "color-brand"},
		{"color.brand.primary", "color-brand-primary"},
		{"--color-brand-primary", "color-brand-primary"},
		{"Color  Brand", "color-brand"},
		{"UPPERCASE", "uppercase"},
		{"with_underscores", "with-underscores"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := slugify(tt.input)
			if result != tt.expected {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToTitleCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"color", "Color"},
		{"brand", "Brand"},
		{"primary", "Primary"},
		{"color-brand", "Color-Brand"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toTitleCase(tt.input)
			if result != tt.expected {
				t.Errorf("toTitleCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildHierarchy(t *testing.T) {
	rows := []Row{
		{Name: "--color-brand-primary", Path: []string{"color", "brand", "primary"}},
		{Name: "--color-brand-secondary", Path: []string{"color", "brand", "secondary"}},
		{Name: "--color-semantic-error", Path: []string{"color", "semantic", "error"}},
		{Name: "--spacing-small", Path: []string{"spacing", "small"}},
	}

	root := BuildHierarchy(rows)

	// Check root has two children: color and spacing
	if len(root.Children) != 2 {
		t.Errorf("expected 2 root children, got %d", len(root.Children))
	}

	colorNode := root.Children["color"]
	if colorNode == nil {
		t.Fatal("expected color node")
	}
	if len(colorNode.Children) != 2 {
		t.Errorf("expected 2 color children (brand, semantic), got %d", len(colorNode.Children))
	}

	brandNode := colorNode.Children["brand"]
	if brandNode == nil {
		t.Fatal("expected brand node")
	}
	if len(brandNode.Tokens) != 2 {
		t.Errorf("expected 2 tokens in brand, got %d", len(brandNode.Tokens))
	}

	spacingNode := root.Children["spacing"]
	if spacingNode == nil {
		t.Fatal("expected spacing node")
	}
	if len(spacingNode.Tokens) != 1 {
		t.Errorf("expected 1 token in spacing, got %d", len(spacingNode.Tokens))
	}
}

func TestExtractGroupMeta(t *testing.T) {
	data := []byte(`{
		"color": {
			"$type": "color",
			"$description": "Brand colors",
			"brand": {
				"$description": "Primary brand palette",
				"primary": { "$value": "#FF6B35" }
			}
		}
	}`)

	meta, err := ExtractGroupMeta(data)
	if err != nil {
		t.Fatalf("ExtractGroupMeta failed: %v", err)
	}

	colorMeta, ok := meta["color"]
	if !ok {
		t.Error("expected color metadata")
	}
	if colorMeta.Description != "Brand colors" {
		t.Errorf("expected 'Brand colors', got %q", colorMeta.Description)
	}
	if colorMeta.Type != "color" {
		t.Errorf("expected 'color' type, got %q", colorMeta.Type)
	}

	brandMeta, ok := meta["color.brand"]
	if !ok {
		t.Error("expected color.brand metadata")
	}
	if brandMeta.Description != "Primary brand palette" {
		t.Errorf("expected 'Primary brand palette', got %q", brandMeta.Description)
	}
}

func TestGenerateTOC(t *testing.T) {
	rows := []Row{
		{Name: "--color-brand-primary", Path: []string{"color", "brand", "primary"}},
		{Name: "--spacing-small", Path: []string{"spacing", "small"}},
	}

	root := BuildHierarchy(rows)
	toc := GenerateTOC(root, 3)

	if !strings.Contains(toc, "## Table Of Contents") {
		t.Error("TOC should contain header")
	}
	if !strings.Contains(toc, "[Color](#color)") {
		t.Error("TOC should contain Color link")
	}
	if !strings.Contains(toc, "[Brand](#color-brand)") {
		t.Error("TOC should contain Brand link")
	}
	if !strings.Contains(toc, "[Spacing](#spacing)") {
		t.Error("TOC should contain Spacing link")
	}
}

func TestFormatTokenName(t *testing.T) {
	tests := []struct {
		name       string
		row        Row
		showLinks  bool
		expected   string
	}{
		{
			name:      "plain name",
			row:       Row{Name: "--color-primary"},
			showLinks: false,
			expected:  "--color-primary",
		},
		{
			name:      "with link",
			row:       Row{Name: "--color-primary"},
			showLinks: true,
			expected:  "[--color-primary](#color-primary)",
		},
		{
			name:      "deprecated",
			row:       Row{Name: "--color-primary", Deprecated: true},
			showLinks: false,
			expected:  "~~--color-primary~~",
		},
		{
			name:      "deprecated with link",
			row:       Row{Name: "--color-primary", Deprecated: true},
			showLinks: true,
			expected:  "~~[--color-primary](#color-primary)~~",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTokenName(tt.row, tt.showLinks)
			if result != tt.expected {
				t.Errorf("formatTokenName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatDescription(t *testing.T) {
	tests := []struct {
		name     string
		row      Row
		expected string
	}{
		{
			name:     "plain description",
			row:      Row{Description: "A color token"},
			expected: "A color token",
		},
		{
			name:     "deprecated with message",
			row:      Row{Deprecated: true, DeprecationMessage: "Use danger instead"},
			expected: "*Deprecated: Use danger instead*",
		},
		{
			name:     "deprecated with description and message",
			row:      Row{Description: "Error color", Deprecated: true, DeprecationMessage: "Use danger instead"},
			expected: "Error color *Deprecated: Use danger instead*",
		},
		{
			name:     "deprecated without message",
			row:      Row{Deprecated: true},
			expected: "*Deprecated*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDescription(tt.row)
			if result != tt.expected {
				t.Errorf("formatDescription() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatRefChain(t *testing.T) {
	tests := []struct {
		name      string
		chain     []string
		showLinks bool
		expected  string
	}{
		{
			name:      "empty chain",
			chain:     nil,
			showLinks: false,
			expected:  "",
		},
		{
			name:      "single ref",
			chain:     []string{"--color-primary"},
			showLinks: false,
			expected:  "--color-primary",
		},
		{
			name:      "multiple refs",
			chain:     []string{"--color-primary", "--color-base"},
			showLinks: false,
			expected:  "--color-primary → --color-base",
		},
		{
			name:      "single ref with link",
			chain:     []string{"--color-primary"},
			showLinks: true,
			expected:  "[--color-primary](#color-primary)",
		},
		{
			name:      "multiple refs with links",
			chain:     []string{"--color-primary", "--color-base"},
			showLinks: true,
			expected:  "[--color-primary](#color-primary) → [--color-base](#color-base)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRefChain(tt.chain, tt.showLinks)
			if result != tt.expected {
				t.Errorf("formatRefChain() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestComputeRowsWithNewFields(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:               "color-primary",
			Value:              "#FF6B35",
			Type:               "color",
			Description:        "Primary color",
			Path:               []string{"color", "primary"},
			Deprecated:         true,
			DeprecationMessage: "Use brand-primary instead",
		},
	}

	rows := ComputeRows(tokens, false)

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	if !row.Deprecated {
		t.Error("expected Deprecated to be true")
	}
	if row.DeprecationMessage != "Use brand-primary instead" {
		t.Errorf("expected deprecation message, got %q", row.DeprecationMessage)
	}
	if len(row.Path) != 2 || row.Path[0] != "color" || row.Path[1] != "primary" {
		t.Errorf("expected path [color, primary], got %v", row.Path)
	}
}

func TestMarkdownWithOptionsGolden(t *testing.T) {
	expected := testutil.LoadFixtureFile(t, "fixtures/markdown/hierarchy/expected.md")

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tokens := []*token.Token{
		{Name: "color-brand-primary", Value: "#FF6B35", Type: "color", Description: "Main brand color", Path: []string{"color", "brand", "primary"}},
		{Name: "color-brand-secondary", Value: "#FF6B35", Type: "color", Path: []string{"color", "brand", "secondary"}, ResolutionChain: []string{"color-brand-primary"}, ResolvedValue: "#FF6B35"},
		{Name: "color-semantic-danger", Value: "#DC3545", Type: "color", Description: "Error and danger states", Path: []string{"color", "semantic", "danger"}},
		{Name: "color-semantic-error", Value: "#FF0000", Type: "color", Path: []string{"color", "semantic", "error"}, Deprecated: true, DeprecationMessage: "Use danger instead"},
		{Name: "spacing-large", Value: "16px", Type: "dimension", Path: []string{"spacing", "large"}},
		{Name: "spacing-medium", Value: "8px", Type: "dimension", Path: []string{"spacing", "medium"}},
		{Name: "spacing-small", Value: "4px", Type: "dimension", Path: []string{"spacing", "small"}},
	}

	rows := ComputeRows(tokens, false)

	groupMeta := map[string]GroupMeta{
		"color":          {Description: "Brand and semantic colors", Type: "color"},
		"color.brand":    {Description: "Primary brand palette"},
		"color.semantic": {Description: "Semantic color tokens"},
		"spacing":        {Description: "Spacing scale", Type: "dimension"},
	}

	opts := MarkdownOptions{
		GroupMeta: groupMeta,
	}

	_ = MarkdownWithOptions(rows, opts)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	actual := buf.String()
	testutil.UpdateGoldenFile(t, "fixtures/markdown/hierarchy/expected.md", []byte(actual))

	if actual != string(expected) {
		t.Errorf("markdown output mismatch.\n\nExpected:\n%s\n\nActual:\n%s", expected, actual)
	}
}
