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

func TestColumnWidths(t *testing.T) {
	rows := []Row{
		{Name: "--color-primary", Type: "color", Value: "#FF6B35"},
		{Name: "--spacing-small", Type: "dimension", Value: "4px"},
		{Name: "--a-very-long-name", Type: "a-long-type", Value: "a moderately long value"},
	}

	nameW, typeW, valW := ColumnWidths(rows)

	// --a-very-long-name is 18 chars
	if nameW != 18 {
		t.Errorf("nameW = %d, want 18", nameW)
	}
	// "a-long-type" is 11 chars
	if typeW != 11 {
		t.Errorf("typeW = %d, want 11", typeW)
	}
	// "a moderately long value" is 23 chars
	if valW != 23 {
		t.Errorf("valW = %d, want 23", valW)
	}
}

func TestColumnWidths_Empty(t *testing.T) {
	nameW, typeW, valW := ColumnWidths(nil)
	// minimums for headers
	if nameW != 4 {
		t.Errorf("nameW = %d, want 4", nameW)
	}
	if typeW != 4 {
		t.Errorf("typeW = %d, want 4", typeW)
	}
	if valW != 5 {
		t.Errorf("valW = %d, want 5", valW)
	}
}

func TestColorSwatch(t *testing.T) {
	// Valid color produces ANSI escape sequence
	swatch := ColorSwatch("#FF0000")
	if swatch == "" {
		t.Error("expected non-empty swatch for valid color")
	}
	if !strings.Contains(swatch, "\x1b[48;2;") {
		t.Error("expected 24-bit ANSI color escape sequence")
	}

	// Invalid color returns empty string
	swatch = ColorSwatch("not-a-color")
	if swatch != "" {
		t.Errorf("expected empty swatch for invalid color, got %q", swatch)
	}
}

func TestNameToCSSVar(t *testing.T) {
	tests := []struct {
		name, prefix, want string
	}{
		{"color-primary", "", "--color-primary"},
		{"color-primary", "rh", "--rh-color-primary"},
		{"a", "x", "--x-a"},
	}

	for _, tt := range tests {
		got := NameToCSSVar(tt.name, tt.prefix)
		if got != tt.want {
			t.Errorf("NameToCSSVar(%q, %q) = %q, want %q", tt.name, tt.prefix, got, tt.want)
		}
	}
}

func TestConvertReferences(t *testing.T) {
	tests := []struct {
		name, input, prefix, want string
	}{
		{"no refs", "plain text", "", "plain text"},
		{"with ref no prefix", "{color.primary}", "", "--color-primary"},
		{"with ref and prefix", "{color.primary}", "rh", "--rh-color-primary"},
		{"embedded ref", "var({color.primary})", "", "var(--color-primary)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertReferences(tt.input, tt.prefix)
			if got != tt.want {
				t.Errorf("convertReferences(%q, %q) = %q, want %q", tt.input, tt.prefix, got, tt.want)
			}
		})
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	w.Close()
	var buf bytes.Buffer
	if _, readErr := buf.ReadFrom(r); readErr != nil {
		t.Fatalf("failed to read captured output: %v", readErr)
	}
	return buf.String()
}

func TestTable(t *testing.T) {
	rows := []Row{
		{Name: "--color-primary", Type: "color", Value: "#FF6B35", IsColor: true},
		{Name: "--spacing-small", Type: "dimension", Value: "4px"},
	}

	output := captureStdout(t, func() {
		_ = Table(rows)
	})

	if !strings.Contains(output, "--color-primary") {
		t.Error("table output should contain --color-primary")
	}
	if !strings.Contains(output, "4px") {
		t.Error("table output should contain 4px")
	}
}

func TestTable_Empty(t *testing.T) {
	err := Table(nil)
	if err != nil {
		t.Errorf("Table(nil) returned error: %v", err)
	}
}

func TestTable_WithRefChain(t *testing.T) {
	rows := []Row{
		{Name: "--color-secondary", Type: "color", Value: "#FF6B35", RefChain: []string{"--color-primary"}},
	}

	output := captureStdout(t, func() {
		_ = Table(rows)
	})

	if !strings.Contains(output, "→") {
		t.Error("table output should contain arrow for ref chain")
	}
}

func TestCSS(t *testing.T) {
	rows := []Row{
		{Name: "--color-primary", Value: "#FF6B35"},
		{Name: "--spacing-small", Value: "4px"},
	}

	output := captureStdout(t, func() {
		_ = CSS(rows)
	})

	if !strings.Contains(output, ":root {") {
		t.Error("CSS output should contain :root selector")
	}
	if !strings.Contains(output, "--color-primary: #FF6B35;") {
		t.Error("CSS output should contain color property")
	}
	if !strings.Contains(output, "--spacing-small: 4px;") {
		t.Error("CSS output should contain spacing property")
	}
}

func TestCSS_SkipsMapValues(t *testing.T) {
	rows := []Row{
		{Name: "--structured", Value: `{"colorSpace": "srgb"}`},
		{Name: "--simple", Value: "#FF6B35"},
	}

	output := captureStdout(t, func() {
		_ = CSS(rows)
	})

	// Map-like values starting with { and containing : should be skipped
	if strings.Contains(output, "--structured") {
		t.Error("CSS should skip map-like values")
	}
	if !strings.Contains(output, "--simple") {
		t.Error("CSS should include simple values")
	}
}

func TestNames(t *testing.T) {
	rows := []Row{
		{Name: "--color-primary"},
		{Name: "--spacing-small"},
	}

	output := captureStdout(t, func() {
		_ = Names(rows)
	})

	if !strings.Contains(output, "--color-primary\n") {
		t.Error("Names output should contain --color-primary")
	}
	if !strings.Contains(output, "--spacing-small\n") {
		t.Error("Names output should contain --spacing-small")
	}
}

func TestMarkdown(t *testing.T) {
	rows := []Row{
		{Name: "--color-primary", Type: "color", Value: "#FF6B35"},
		{Name: "--spacing-small", Type: "dimension", Value: "4px"},
	}

	output := captureStdout(t, func() {
		_ = Markdown(rows)
	})

	if !strings.Contains(output, "## color") {
		t.Error("Markdown should contain color heading")
	}
	if !strings.Contains(output, "## dimension") {
		t.Error("Markdown should contain dimension heading")
	}
	if !strings.Contains(output, "--color-primary") {
		t.Error("Markdown should contain token name")
	}
}

func TestMarkdown_Empty(t *testing.T) {
	err := Markdown(nil)
	if err != nil {
		t.Errorf("Markdown(nil) returned error: %v", err)
	}
}

func TestMarkdown_UntypedHeading(t *testing.T) {
	rows := []Row{
		{Name: "--mystery", Type: "-", Value: "42"},
	}

	output := captureStdout(t, func() {
		_ = Markdown(rows)
	})

	if !strings.Contains(output, "## untyped") {
		t.Error("Markdown should use 'untyped' heading for '-' type")
	}
}

func TestMarkdown_WithRefChain(t *testing.T) {
	rows := []Row{
		{Name: "--color-primary", Type: "color", Value: "#FF6B35"},
		{Name: "--color-secondary", Type: "color", Value: "#FF6B35", RefChain: []string{"--color-primary"}},
	}

	output := captureStdout(t, func() {
		_ = Markdown(rows)
	})

	if !strings.Contains(output, "Reference") {
		t.Error("Markdown should contain Reference column when refs are present")
	}
}

func TestMarkdownWithOptions_Empty(t *testing.T) {
	err := MarkdownWithOptions(nil, MarkdownOptions{})
	if err != nil {
		t.Errorf("MarkdownWithOptions(nil) returned error: %v", err)
	}
}

func TestMarkdownWithOptions_WithTOC(t *testing.T) {
	tokens := []*token.Token{
		{Name: "color-brand-primary", Value: "#FF6B35", Type: "color", Path: []string{"color", "brand", "primary"}},
		{Name: "spacing-small", Value: "4px", Type: "dimension", Path: []string{"spacing", "small"}},
	}

	rows := ComputeRows(tokens, false)
	output := captureStdout(t, func() {
		_ = MarkdownWithOptions(rows, MarkdownOptions{
			IncludeTOC: true,
			TOCDepth:   2,
		})
	})

	if !strings.Contains(output, "## Table Of Contents") {
		t.Error("output should contain TOC header")
	}
}

func TestBuildHierarchy_EmptyPath(t *testing.T) {
	rows := []Row{
		{Name: "--orphan", Path: nil},
	}

	root := BuildHierarchy(rows)
	if len(root.Tokens) != 1 {
		t.Errorf("expected 1 root-level token, got %d", len(root.Tokens))
	}
}

func TestMarkdownWithOptionsGolden(t *testing.T) {
	expected := testutil.LoadFixtureFile(t, "fixtures/markdown/hierarchy/expected.md")

	// Capture stdout
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
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
