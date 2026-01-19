/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package render provides shared rendering functions for CLI output.
package render

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/mazznoer/csscolorparser"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"bennypowers.dev/asimonim/token"
)

// Row holds computed display values for a single token.
type Row struct {
	Name               string   // CSS variable name with prefix
	Type               string   // Token type or "-"
	Value              string   // Display value (resolved if applicable)
	Description        string   // Token description
	RefChain           []string // Resolution chain as CSS variable names
	IsColor            bool     // Whether this is a color token with parseable value
	Deprecated         bool     // Whether this token is deprecated
	DeprecationMessage string   // Optional message explaining deprecation
	Path               []string // Token path in the hierarchy (e.g., ["color", "brand", "primary"])
}

// GroupMeta holds metadata extracted from group definitions.
type GroupMeta struct {
	Description string
	Type        string
}

// HierarchyNode represents a node in the token hierarchy tree.
type HierarchyNode struct {
	Name     string
	Path     []string
	Meta     *GroupMeta
	Tokens   []Row
	Children map[string]*HierarchyNode
}

// MarkdownOptions configures markdown output.
type MarkdownOptions struct {
	GroupMeta  map[string]GroupMeta // key: dot-separated path
	IncludeTOC bool
	TOCDepth   int
	ShowLinks  bool
}

// ComputeRows transforms tokens into display rows with all values computed.
func ComputeRows(tokens []*token.Token, resolved bool) []Row {
	rows := make([]Row, 0, len(tokens))
	for _, tok := range tokens {
		row := Row{
			Name:               tok.CSSVariableName(),
			Type:               tok.Type,
			Value:              FormatValue(tok.Value, tok.Prefix),
			Description:        tok.Description,
			Deprecated:         tok.Deprecated,
			DeprecationMessage: tok.DeprecationMessage,
			Path:               tok.Path,
		}
		if row.Type == "" {
			row.Type = "-"
		}

		// Handle alias resolution
		if len(tok.ResolutionChain) > 0 && tok.ResolvedValue != nil {
			row.Value = FormatValue(tok.ResolvedValue, tok.Prefix)
			// Convert chain to CSS variable names
			row.RefChain = make([]string, len(tok.ResolutionChain))
			for i, name := range tok.ResolutionChain {
				row.RefChain[i] = NameToCSSVar(name, tok.Prefix)
			}
		} else if resolved && tok.ResolvedValue != nil {
			row.Value = FormatValue(tok.ResolvedValue, tok.Prefix)
		} else if row.Value == "" && tok.RawValue != nil {
			// Composite types (cubicBezier, shadow, etc.) have empty Value but populated RawValue
			row.Value = FormatValue(tok.RawValue, tok.Prefix)
		}

		// Check if this is a parseable color
		if tok.Type == "color" && !strings.HasPrefix(row.Value, "{") && !strings.HasPrefix(row.Value, "--") {
			if _, err := csscolorparser.Parse(row.Value); err == nil {
				row.IsColor = true
			}
		}

		rows = append(rows, row)
	}
	return rows
}

// FormatValue converts any value to a display string.
// References like {foo.bar} are converted to CSS variable names.
func FormatValue(v any, prefix string) string {
	switch val := v.(type) {
	case string:
		return formatStringValue(val, prefix)
	case []any:
		// cubicBezier: [x1, y1, x2, y2] -> cubic-bezier(x1, y1, x2, y2)
		if len(val) == 4 && isNumericArray(val) {
			return fmt.Sprintf("cubic-bezier(%v, %v, %v, %v)", val[0], val[1], val[2], val[3])
		}
		// Array of references or values - format each element
		parts := make([]string, len(val))
		for i, v := range val {
			if s, ok := v.(string); ok {
				parts[i] = formatStringValue(s, prefix)
			} else {
				parts[i] = fmt.Sprintf("%v", v)
			}
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		return formatCompositeValue(val, prefix)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// formatStringValue formats a string value, converting references to CSS variable names.
func formatStringValue(s, prefix string) string {
	if !strings.Contains(s, "{") {
		return s
	}
	// Replace each {ref.path} with --prefix-ref-path
	result := refPattern.ReplaceAllStringFunc(s, func(match string) string {
		// Extract the path from {path}
		path := strings.TrimSuffix(strings.TrimPrefix(match, "{"), "}")
		name := strings.ReplaceAll(path, ".", "-")
		return NameToCSSVar(name, prefix)
	})
	return result
}

var refPattern = regexp.MustCompile(`\{[^}]+\}`)

// isNumericArray checks if all elements in the array are numeric (for cubicBezier detection).
func isNumericArray(arr []any) bool {
	for _, v := range arr {
		switch v.(type) {
		case int, int64, float64:
			continue
		default:
			return false
		}
	}
	return true
}

// formatCompositeValue formats a composite token value (shadow, border, etc.) as CSS.
func formatCompositeValue(m map[string]any, prefix string) string {
	// Helper to format a value from the map, converting references
	fv := func(key string) string {
		v := m[key]
		if s, ok := v.(string); ok {
			return formatStringValue(s, prefix)
		}
		return fmt.Sprintf("%v", v)
	}

	// shadow: offsetX offsetY blur spread color
	if hasKeys(m, "offsetX", "offsetY", "blur", "color") {
		spread := ""
		if s, ok := m["spread"].(string); ok && s != "" && s != "0px" {
			spread = " " + formatStringValue(s, prefix)
		}
		return fmt.Sprintf("%s %s %s%s %s", fv("offsetX"), fv("offsetY"), fv("blur"), spread, fv("color"))
	}
	// border: width style color
	if hasKeys(m, "width", "style", "color") {
		return fmt.Sprintf("%s %s %s", fv("width"), fv("style"), fv("color"))
	}
	// strokeStyle: dashArray lineCap
	if hasKeys(m, "dashArray", "lineCap") {
		return fmt.Sprintf("dash:%s cap:%s", fv("dashArray"), fv("lineCap"))
	}
	// transition: duration delay timingFunction
	if hasKeys(m, "duration", "timingFunction") {
		delay := ""
		if d, ok := m["delay"].(string); ok && d != "" && d != "0s" && d != "0ms" {
			delay = " " + formatStringValue(d, prefix)
		}
		return fmt.Sprintf("%s%s %s", fv("duration"), delay, fv("timingFunction"))
	}
	// gradient: type, stops
	if hasKeys(m, "type", "stops") {
		return fmt.Sprintf("%s-gradient(...)", fv("type"))
	}
	// typography: fontFamily fontSize fontWeight lineHeight
	if hasKeys(m, "fontFamily") {
		parts := []string{}
		if _, ok := m["fontWeight"]; ok {
			parts = append(parts, fv("fontWeight"))
		}
		if _, ok := m["fontSize"]; ok {
			parts = append(parts, fv("fontSize"))
		}
		if _, ok := m["lineHeight"]; ok {
			parts = append(parts, fmt.Sprintf("/ %s", fv("lineHeight")))
		}
		parts = append(parts, fv("fontFamily"))
		return strings.Join(parts, " ")
	}
	// fallback: key: value pairs
	parts := make([]string, 0, len(m))
	for k := range m {
		parts = append(parts, fmt.Sprintf("%s: %s", k, fv(k)))
	}
	sort.Strings(parts)
	return strings.Join(parts, "; ")
}

// hasKeys returns true if the map contains all specified keys.
func hasKeys(m map[string]any, keys ...string) bool {
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			return false
		}
	}
	return true
}

// NameToCSSVar converts a token name to a CSS variable name.
// e.g., "color-primary" with prefix "rh" → "--rh-color-primary"
func NameToCSSVar(name, prefix string) string {
	if prefix != "" {
		return "--" + prefix + "-" + name
	}
	return "--" + name
}

// ColumnWidths calculates the max width needed for each column.
func ColumnWidths(rows []Row) (name, typ, val int) {
	name, typ, val = 4, 4, 5 // minimums for headers
	for _, r := range rows {
		if len(r.Name) > name {
			name = len(r.Name)
		}
		if len(r.Type) > typ {
			typ = len(r.Type)
		}
		if len(r.Value) > val {
			val = len(r.Value)
		}
	}
	return
}

// ColorSwatch returns a 24-bit ANSI color block for the given color value.
func ColorSwatch(value string) string {
	c, err := csscolorparser.Parse(value)
	if err != nil {
		return ""
	}
	r, g, b, _ := c.RGBA255()
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm  \x1b[0m ", r, g, b)
}

// Table renders rows as a table to stdout.
func Table(rows []Row) error {
	if len(rows) == 0 {
		return nil
	}
	nameW, typeW, _ := ColumnWidths(rows)
	for _, r := range rows {
		swatch := ""
		if r.IsColor {
			swatch = ColorSwatch(r.Value)
		}
		refChain := ""
		if len(r.RefChain) > 0 {
			refChain = " → " + strings.Join(r.RefChain, " → ")
		}
		fmt.Printf("%-*s  %-*s  %s%s%s\n", nameW, r.Name, typeW, r.Type, swatch, r.Value, refChain)
	}
	return nil
}

// Markdown renders rows as markdown tables grouped by type.
func Markdown(rows []Row) error {
	if len(rows) == 0 {
		return nil
	}

	// Group rows by type, preserving order of first occurrence
	typeOrder := make([]string, 0)
	byType := make(map[string][]Row)
	for _, r := range rows {
		if _, exists := byType[r.Type]; !exists {
			typeOrder = append(typeOrder, r.Type)
		}
		byType[r.Type] = append(byType[r.Type], r)
	}

	first := true
	for _, typ := range typeOrder {
		group := byType[typ]
		if !first {
			fmt.Println()
		}
		first = false

		// Heading
		heading := typ
		if heading == "-" {
			heading = "untyped"
		}
		fmt.Printf("## %s\n\n", heading)

		// Calculate column widths for this group
		nameW, valW, refW := 4, 5, 0
		hasRefs := false
		for _, r := range group {
			if len(r.Name) > nameW {
				nameW = len(r.Name)
			}
			if len(r.Value) > valW {
				valW = len(r.Value)
			}
			if len(r.RefChain) > 0 {
				hasRefs = true
				refStr := strings.Join(r.RefChain, " → ")
				if len(refStr) > refW {
					refW = len(refStr)
				}
			}
		}
		if refW < 9 {
			refW = 9 // "Reference"
		}

		// Render table
		if hasRefs {
			fmt.Printf("| %-*s | %-*s | %-*s |\n", nameW, "Name", valW, "Value", refW, "Reference")
			fmt.Printf("|-%s-|-%s-|-%s-|\n", strings.Repeat("-", nameW), strings.Repeat("-", valW), strings.Repeat("-", refW))
			for _, r := range group {
				refStr := strings.Join(r.RefChain, " → ")
				fmt.Printf("| %-*s | %-*s | %-*s |\n", nameW, r.Name, valW, r.Value, refW, refStr)
			}
		} else {
			fmt.Printf("| %-*s | %-*s |\n", nameW, "Name", valW, "Value")
			fmt.Printf("|-%s-|-%s-|\n", strings.Repeat("-", nameW), strings.Repeat("-", valW))
			for _, r := range group {
				fmt.Printf("| %-*s | %-*s |\n", nameW, r.Name, valW, r.Value)
			}
		}
	}
	return nil
}

// CSS renders rows as CSS custom properties.
func CSS(rows []Row) error {
	fmt.Println(":root {")
	for _, r := range rows {
		if strings.HasPrefix(r.Value, "{") && strings.Contains(r.Value, ":") {
			continue
		}
		fmt.Printf("  %s: %s;\n", r.Name, r.Value)
	}
	fmt.Println("}")
	return nil
}

// Names renders just the token names, one per line.
func Names(rows []Row) error {
	for _, r := range rows {
		fmt.Println(r.Name)
	}
	return nil
}

// slugify converts a name to a URL-safe anchor ID.
// e.g., "Color Brand" -> "color-brand"
func slugify(name string) string {
	var result strings.Builder
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' || r == '.' {
			result.WriteRune('-')
		}
	}
	// Remove consecutive dashes
	s := result.String()
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

// toTitleCase converts a string to Title Case.
func toTitleCase(s string) string {
	caser := cases.Title(language.English)
	return caser.String(s)
}

// BuildHierarchy builds a tree from rows based on their Path.
func BuildHierarchy(rows []Row) *HierarchyNode {
	root := &HierarchyNode{
		Name:     "",
		Path:     nil,
		Children: make(map[string]*HierarchyNode),
	}

	for _, row := range rows {
		if len(row.Path) == 0 {
			root.Tokens = append(root.Tokens, row)
			continue
		}

		// Navigate/create path to parent node
		current := root
		for i := 0; i < len(row.Path)-1; i++ {
			name := row.Path[i]
			if current.Children[name] == nil {
				current.Children[name] = &HierarchyNode{
					Name:     name,
					Path:     row.Path[:i+1],
					Children: make(map[string]*HierarchyNode),
				}
			}
			current = current.Children[name]
		}

		// Add token to parent
		current.Tokens = append(current.Tokens, row)
	}

	return root
}

// ExtractGroupMeta parses JSON to extract group $description and $type values.
// Returns a map keyed by dot-separated path (e.g., "color.brand").
func ExtractGroupMeta(data []byte) (map[string]GroupMeta, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	result := make(map[string]GroupMeta)
	extractGroupMetaRecursive(raw, nil, result)
	return result, nil
}

func extractGroupMetaRecursive(obj map[string]any, path []string, result map[string]GroupMeta) {
	meta := GroupMeta{}
	hasMetadata := false

	if desc, ok := obj["$description"].(string); ok {
		meta.Description = desc
		hasMetadata = true
	}
	if typ, ok := obj["$type"].(string); ok {
		meta.Type = typ
		hasMetadata = true
	}

	if hasMetadata && len(path) > 0 {
		result[strings.Join(path, ".")] = meta
	}

	for key, value := range obj {
		if strings.HasPrefix(key, "$") {
			continue
		}
		if child, ok := value.(map[string]any); ok {
			// Create a new slice to avoid aliasing the backing array
			childPath := make([]string, len(path)+1)
			copy(childPath, path)
			childPath[len(path)] = key
			extractGroupMetaRecursive(child, childPath, result)
		}
	}
}

// GenerateTOC generates a markdown table of contents from the hierarchy.
func GenerateTOC(root *HierarchyNode, maxDepth int) string {
	var sb strings.Builder
	sb.WriteString("## Table Of Contents\n\n")
	generateTOCRecursive(root, 0, maxDepth, &sb)
	return sb.String()
}

func generateTOCRecursive(node *HierarchyNode, depth int, maxDepth int, sb *strings.Builder) {
	if depth >= maxDepth {
		return
	}

	// Get sorted child names
	names := make([]string, 0, len(node.Children))
	for name := range node.Children {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		child := node.Children[name]
		indent := strings.Repeat("  ", depth)
		slug := slugify(strings.Join(child.Path, "-"))
		title := toTitleCase(name)
		fmt.Fprintf(sb, "%s- [%s](#%s)\n", indent, title, slug)
		generateTOCRecursive(child, depth+1, maxDepth, sb)
	}
}

// MarkdownWithOptions renders rows as markdown with hierarchy grouping and options.
func MarkdownWithOptions(rows []Row, opts MarkdownOptions) error {
	if len(rows) == 0 {
		return nil
	}

	hierarchy := BuildHierarchy(rows)

	// Inject group metadata if provided
	if opts.GroupMeta != nil {
		injectGroupMeta(hierarchy, opts.GroupMeta)
	}

	// Generate TOC if requested
	if opts.IncludeTOC {
		tocDepth := opts.TOCDepth
		if tocDepth <= 0 {
			tocDepth = 3
		}
		fmt.Print(GenerateTOC(hierarchy, tocDepth))
		fmt.Println()
	}

	// Render hierarchy
	renderHierarchyNode(hierarchy, 1, opts)
	return nil
}

func injectGroupMeta(node *HierarchyNode, meta map[string]GroupMeta) {
	if len(node.Path) > 0 {
		key := strings.Join(node.Path, ".")
		if m, ok := meta[key]; ok {
			node.Meta = &m
		}
	}
	for _, child := range node.Children {
		injectGroupMeta(child, meta)
	}
}

func renderHierarchyNode(node *HierarchyNode, depth int, opts MarkdownOptions) {
	// Get sorted child names for consistent output
	names := make([]string, 0, len(node.Children))
	for name := range node.Children {
		names = append(names, name)
	}
	sort.Strings(names)

	// Render children first (sections)
	for _, name := range names {
		child := node.Children[name]

		// Heading level: ## for depth 1, ### for depth 2, etc. (max h6)
		level := min(depth+1, 6)
		heading := strings.Repeat("#", level)
		title := toTitleCase(name)
		slug := slugify(strings.Join(child.Path, "-"))

		fmt.Printf("%s %s {#%s}\n\n", heading, title, slug)

		// Render group description if available
		if child.Meta != nil && child.Meta.Description != "" {
			fmt.Println(child.Meta.Description)
			fmt.Println()
		}

		// Render tokens at this level
		if len(child.Tokens) > 0 {
			renderTokenTable(child.Tokens, opts)
			fmt.Println()
		}

		// Recurse into children
		renderHierarchyNode(child, depth+1, opts)
	}

	// Render root-level tokens (no path)
	if node.Path == nil && len(node.Tokens) > 0 {
		renderTokenTable(node.Tokens, opts)
		fmt.Println()
	}
}

func renderTokenTable(tokens []Row, opts MarkdownOptions) {
	if len(tokens) == 0 {
		return
	}

	// Calculate column widths
	nameW, valW, descW, refW := 4, 5, 11, 9 // minimums for headers
	hasRefs := false
	hasDesc := false
	hasDeprecated := false

	for _, r := range tokens {
		displayName := formatTokenName(r, opts.ShowLinks)
		if len(displayName) > nameW {
			nameW = len(displayName)
		}
		if len(r.Value) > valW {
			valW = len(r.Value)
		}
		if r.Description != "" || r.DeprecationMessage != "" {
			hasDesc = true
			desc := formatDescription(r)
			if len(desc) > descW {
				descW = len(desc)
			}
		}
		if len(r.RefChain) > 0 {
			hasRefs = true
			refStr := formatRefChain(r.RefChain, opts.ShowLinks)
			if len(refStr) > refW {
				refW = len(refStr)
			}
		}
		if r.Deprecated {
			hasDeprecated = true
		}
	}

	// Ensure minimum widths for headers
	if descW < 11 {
		descW = 11 // "Description"
	}
	if refW < 9 {
		refW = 9 // "Reference"
	}

	_ = hasDeprecated // deprecation is shown inline in name and description

	// Render table header
	if hasRefs && hasDesc {
		fmt.Printf("| %-*s | %-*s | %-*s | %-*s |\n", nameW, "Name", valW, "Value", descW, "Description", refW, "Reference")
		fmt.Printf("|-%s-|-%s-|-%s-|-%s-|\n",
			strings.Repeat("-", nameW), strings.Repeat("-", valW),
			strings.Repeat("-", descW), strings.Repeat("-", refW))
	} else if hasRefs {
		fmt.Printf("| %-*s | %-*s | %-*s |\n", nameW, "Name", valW, "Value", refW, "Reference")
		fmt.Printf("|-%s-|-%s-|-%s-|\n",
			strings.Repeat("-", nameW), strings.Repeat("-", valW), strings.Repeat("-", refW))
	} else if hasDesc {
		fmt.Printf("| %-*s | %-*s | %-*s |\n", nameW, "Name", valW, "Value", descW, "Description")
		fmt.Printf("|-%s-|-%s-|-%s-|\n",
			strings.Repeat("-", nameW), strings.Repeat("-", valW), strings.Repeat("-", descW))
	} else {
		fmt.Printf("| %-*s | %-*s |\n", nameW, "Name", valW, "Value")
		fmt.Printf("|-%s-|-%s-|\n", strings.Repeat("-", nameW), strings.Repeat("-", valW))
	}

	// Render rows
	for _, r := range tokens {
		displayName := formatTokenName(r, opts.ShowLinks)
		desc := formatDescription(r)
		refStr := formatRefChain(r.RefChain, opts.ShowLinks)

		if hasRefs && hasDesc {
			fmt.Printf("| %-*s | %-*s | %-*s | %-*s |\n", nameW, displayName, valW, r.Value, descW, desc, refW, refStr)
		} else if hasRefs {
			fmt.Printf("| %-*s | %-*s | %-*s |\n", nameW, displayName, valW, r.Value, refW, refStr)
		} else if hasDesc {
			fmt.Printf("| %-*s | %-*s | %-*s |\n", nameW, displayName, valW, r.Value, descW, desc)
		} else {
			fmt.Printf("| %-*s | %-*s |\n", nameW, displayName, valW, r.Value)
		}
	}
}

func formatTokenName(r Row, showLinks bool) string {
	name := r.Name
	if showLinks {
		slug := slugify(r.Name)
		name = fmt.Sprintf("[%s](#%s)", r.Name, slug)
	}
	if r.Deprecated {
		name = "~~" + name + "~~"
	}
	return name
}

func formatDescription(r Row) string {
	desc := r.Description
	if r.Deprecated && r.DeprecationMessage != "" {
		if desc != "" {
			desc += " "
		}
		desc += "*Deprecated: " + r.DeprecationMessage + "*"
	} else if r.Deprecated && desc == "" {
		desc = "*Deprecated*"
	}
	return desc
}

func formatRefChain(chain []string, showLinks bool) string {
	if len(chain) == 0 {
		return ""
	}
	if showLinks {
		parts := make([]string, len(chain))
		for i, ref := range chain {
			slug := slugify(ref)
			parts[i] = fmt.Sprintf("[%s](#%s)", ref, slug)
		}
		return strings.Join(parts, " → ")
	}
	return strings.Join(chain, " → ")
}
