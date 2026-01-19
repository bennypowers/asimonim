/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package render provides shared rendering functions for CLI output.
package render

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/mazznoer/csscolorparser"

	"bennypowers.dev/asimonim/token"
)

// Row holds computed display values for a single token.
type Row struct {
	Name        string   // CSS variable name with prefix
	Type        string   // Token type or "-"
	Value       string   // Display value (resolved if applicable)
	Description string   // Token description
	RefChain    []string // Resolution chain as CSS variable names
	IsColor     bool     // Whether this is a color token with parseable value
}

// ComputeRows transforms tokens into display rows with all values computed.
func ComputeRows(tokens []*token.Token, resolved bool) []Row {
	rows := make([]Row, 0, len(tokens))
	for _, tok := range tokens {
		row := Row{
			Name:        tok.CSSVariableName(),
			Type:        tok.Type,
			Value:       FormatValue(tok.Value, tok.Prefix),
			Description: tok.Description,
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
		return fmt.Sprintf("dash:%v cap:%s", m["dashArray"], fv("lineCap"))
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
