/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package js

import (
	"bytes"
	"embed"
	"encoding/json"
	"strings"
	"text/template"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/parser/common"
	"bennypowers.dev/asimonim/token"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

var templates *template.Template

func init() {
	templates = template.Must(template.ParseFS(templateFS, "templates/*.tmpl"))
}

// templateData holds data for template execution.
type templateData struct {
	Imports        string
	TypesPath      string
	TokenNames     []string
	Entries        []entryData
	ClassName      string
	ValueTypeUnion string
	Prefix         string
	Delimiter      string
	UseJSDoc       bool
	UseCJS         bool
}

// entryData holds data for a single token entry.
type entryData struct {
	CSSVar    string
	Value     string
	ValueType string
}

// formatMap generates TokenMap class output.
func (f *Formatter) formatMap(tokens []*token.Token, opts formatter.Options) ([]byte, error) {
	sorted := formatter.SortTokens(tokens)

	switch f.opts.MapMode {
	case MapModeTypes:
		return f.executeTemplate("types.ts.tmpl", nil)

	case MapModeModule:
		return f.formatSplitModule(sorted, opts)

	default:
		return f.formatFull(sorted, opts)
	}
}

// formatFull generates the complete output with types, class, and tokens.
func (f *Formatter) formatFull(tokens []*token.Token, opts formatter.Options) ([]byte, error) {
	data := templateData{
		TokenNames: buildTokenNames(tokens, opts),
		Entries:    buildEntries(tokens, opts),
		Prefix:     escapeTS(opts.Prefix),
		Delimiter:  escapeTS(defaultDelimiter(opts.Delimiter)),
		UseJSDoc:   f.opts.Types == TypesJSDoc,
		UseCJS:     f.opts.Module == ModuleCJS,
	}

	return f.executeTemplate("full.ts.tmpl", data)
}

// formatSplitModule generates a split module that imports from shared types.
func (f *Formatter) formatSplitModule(tokens []*token.Token, opts formatter.Options) ([]byte, error) {
	typesPath := f.opts.TypesPath
	if typesPath == "" {
		// Use extension matching the output type (.ts for TypeScript, .js for JSDoc)
		if f.opts.Types == TypesJSDoc {
			typesPath = "./types.js"
		} else {
			typesPath = "./types.ts"
		}
	}

	// Collect types that need to be imported
	usedTypes := collectUsedTypes(tokens)

	// Build import list: always include TokenMap and DesignToken
	imports := []string{"TokenMap", "DesignToken"}
	for _, t := range []string{"Color", "Dimension"} {
		if usedTypes[t] {
			imports = append(imports, t)
		}
	}

	// Generate class name from ClassName option
	className := f.opts.ClassName
	if className == "" {
		className = "TokensTokenMap"
	}

	// Collect all value types for the return type union
	valueTypes := collectValueTypes(tokens)

	data := templateData{
		Imports:        strings.Join(imports, ", "),
		TypesPath:      escapeTS(typesPath),
		TokenNames:     buildTokenNames(tokens, opts),
		Entries:        buildEntries(tokens, opts),
		ClassName:      className,
		ValueTypeUnion: buildValueTypeUnion(valueTypes),
		Prefix:         escapeTS(opts.Prefix),
		Delimiter:      escapeTS(defaultDelimiter(opts.Delimiter)),
		UseJSDoc:       f.opts.Types == TypesJSDoc,
		UseCJS:         f.opts.Module == ModuleCJS,
	}

	return f.executeTemplate("module.ts.tmpl", data)
}

// executeTemplate executes a template by name and returns the result.
func (f *Formatter) executeTemplate(name string, data any) ([]byte, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// buildTokenNames builds the list of token names (CSS var and dot-path) for the union type.
// Uses a set to deduplicate in case any names collide.
func buildTokenNames(tokens []*token.Token, opts formatter.Options) []string {
	seen := make(map[string]bool)
	var names []string
	for _, tok := range tokens {
		cssVar := escapeTS(buildCSSVarName(tok, opts))
		dotPath := escapeTS(buildDotPath(tok))
		if !seen[cssVar] {
			seen[cssVar] = true
			names = append(names, cssVar)
		}
		if !seen[dotPath] {
			seen[dotPath] = true
			names = append(names, dotPath)
		}
	}
	return names
}

// buildEntries builds the entry data for tokens.
func buildEntries(tokens []*token.Token, opts formatter.Options) []entryData {
	entries := make([]entryData, 0, len(tokens))
	for _, tok := range tokens {
		entries = append(entries, entryData{
			CSSVar:    escapeTS(buildCSSVarName(tok, opts)),
			Value:     formatValue(tok),
			ValueType: inferValueType(tok),
		})
	}
	return entries
}

// defaultDelimiter returns the delimiter, defaulting to "-".
func defaultDelimiter(d string) string {
	if d == "" {
		return "-"
	}
	return d
}

// buildCSSVarName constructs a CSS variable name like --rh-color-blue.
func buildCSSVarName(tok *token.Token, opts formatter.Options) string {
	name := strings.Join(tok.Path, "-")
	if opts.Prefix != "" {
		name = opts.Prefix + "-" + name
	}
	return "--" + name
}

// buildDotPath constructs a dot-separated path like color.blue (no prefix).
func buildDotPath(tok *token.Token) string {
	return strings.Join(tok.Path, ".")
}

// escapeTS escapes a string for use in a TypeScript double-quoted string literal.
// It escapes backslashes, double quotes, and control characters.
func escapeTS(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			sb.WriteString(`\\`)
		case '"':
			sb.WriteString(`\"`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		case '\t':
			sb.WriteString(`\t`)
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// collectUsedTypes returns which type interfaces are used by the tokens.
func collectUsedTypes(tokens []*token.Token) map[string]bool {
	used := make(map[string]bool)
	for _, tok := range tokens {
		valueType := inferValueType(tok)
		if strings.Contains(valueType, "Color") {
			used["Color"] = true
		}
		if strings.Contains(valueType, "Dimension") {
			used["Dimension"] = true
		}
	}
	return used
}

// collectValueTypes returns the unique value types for tokens.
func collectValueTypes(tokens []*token.Token) []string {
	seen := make(map[string]bool)
	var types []string
	for _, tok := range tokens {
		valueType := inferValueType(tok)
		if !seen[valueType] {
			seen[valueType] = true
			types = append(types, valueType)
		}
	}
	return types
}

// buildValueTypeUnion builds a union of DesignToken<T> for each value type.
func buildValueTypeUnion(valueTypes []string) string {
	if len(valueTypes) == 0 {
		return "DesignToken<unknown>"
	}
	var parts []string
	for _, t := range valueTypes {
		parts = append(parts, "DesignToken<"+t+">")
	}
	return strings.Join(parts, " | ")
}

// formatValue formats a token value for TypeScript output.
func formatValue(tok *token.Token) string {
	value := formatter.ResolvedValue(tok)

	result := map[string]any{
		"$value": formatTypedValue(tok, value),
	}

	if tok.Type != "" {
		result["$type"] = tok.Type
	}
	if tok.Description != "" {
		result["$description"] = tok.Description
	}

	data, err := json.MarshalIndent(result, "    ", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

// formatTypedValue formats a value based on token type.
func formatTypedValue(tok *token.Token, value any) any {
	switch tok.Type {
	case token.TypeColor:
		// Try to parse as structured color
		if colorVal, err := common.ParseColorValue(value, tok.SchemaVersion); err == nil {
			if objColor, ok := colorVal.(*common.ObjectColorValue); ok {
				result := map[string]any{
					"colorSpace": objColor.ColorSpace,
					"components": objColor.Components,
				}
				if objColor.Alpha != nil {
					result["alpha"] = *objColor.Alpha
				}
				if objColor.Hex != nil {
					result["hex"] = *objColor.Hex
				}
				return result
			}
		}
		// For string colors, return as-is
		return value

	case token.TypeDimension:
		// Check if it's already a structured dimension
		if m, ok := value.(map[string]any); ok {
			if _, hasValue := m["value"]; hasValue {
				if _, hasUnit := m["unit"]; hasUnit {
					return m
				}
			}
		}
		return value

	default:
		return value
	}
}

// inferValueType infers the TypeScript type for a token value.
func inferValueType(tok *token.Token) string {
	switch tok.Type {
	case token.TypeColor:
		// Check if it's a structured color
		value := formatter.ResolvedValue(tok)
		if _, err := common.ParseColorValue(value, tok.SchemaVersion); err == nil {
			return "Color"
		}
		if _, ok := value.(map[string]any); ok {
			return "Color"
		}
		return "string"

	case token.TypeDimension:
		value := formatter.ResolvedValue(tok)
		if m, ok := value.(map[string]any); ok {
			if _, hasValue := m["value"]; hasValue {
				if _, hasUnit := m["unit"]; hasUnit {
					return "Dimension"
				}
			}
		}
		return "string"

	case token.TypeNumber, token.TypeFontWeight:
		return "number"

	case token.TypeCubicBezier:
		return "[number, number, number, number]"

	case token.TypeFontFamily:
		value := formatter.ResolvedValue(tok)
		if _, ok := value.([]any); ok {
			return "string[]"
		}
		return "string"

	case token.TypeDuration:
		value := formatter.ResolvedValue(tok)
		if _, ok := value.(map[string]any); ok {
			return "{ value: number; unit: string }"
		}
		return "string"

	case token.TypeShadow:
		return "{ offsetX: Dimension | string; offsetY: Dimension | string; blur: Dimension | string; spread?: Dimension | string; color: Color | string }"

	case token.TypeBorder:
		return "{ width: Dimension | string; style: string; color: Color | string }"

	case token.TypeTypography:
		return "{ fontFamily?: string | string[]; fontSize?: Dimension | string; fontWeight?: number | string; lineHeight?: number | string; letterSpacing?: Dimension | string }"

	case token.TypeTransition:
		return "{ duration: { value: number; unit: string } | string; timingFunction: [number, number, number, number] | string; delay?: { value: number; unit: string } | string }"

	case token.TypeGradient:
		return "{ type: string; stops: { color: Color | string; position: number }[] }"

	case token.TypeStrokeStyle:
		value := formatter.ResolvedValue(tok)
		if _, ok := value.(map[string]any); ok {
			return "{ dashArray: Dimension[]; lineCap?: string }"
		}
		return "string"

	default:
		value := formatter.ResolvedValue(tok)
		switch value.(type) {
		case string:
			return "string"
		case float64:
			return "number"
		case int:
			return "number"
		case bool:
			return "boolean"
		case []any:
			return "unknown[]"
		case map[string]any:
			return "Record<string, unknown>"
		default:
			return "unknown"
		}
	}
}
