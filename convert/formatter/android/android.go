/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package android provides Android XML resource formatting for design tokens.
package android

import (
	"fmt"
	"strings"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/parser/common"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/token"
)

// Formatter outputs Android-style XML resources.
type Formatter struct{}

// New creates a new Android formatter.
func New() *Formatter {
	return &Formatter{}
}

// Format converts tokens to Android XML resource format.
func (f *Formatter) Format(tokens []*token.Token, opts formatter.Options) ([]byte, error) {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="utf-8"?>`)
	sb.WriteString("\n")

	// Add header if provided
	if opts.Header != "" {
		sb.WriteString(formatter.FormatHeader(opts.Header, formatter.XMLComments))
	}

	sb.WriteString("<resources>\n")

	sorted := formatter.SortTokens(tokens)

	for _, tok := range sorted {
		baseName := formatter.ToSnakeCase(strings.Join(tok.Path, "_"))
		name := formatter.ApplyPrefix(baseName, opts.Prefix, "_")
		value := toAndroidValue(tok)
		xmlType := xmlType(tok.Type)

		fmt.Fprintf(&sb, "    <%s name=\"%s\">%s</%s>\n",
			xmlType, formatter.EscapeXML(name), formatter.EscapeXML(value), xmlType)
	}

	sb.WriteString("</resources>\n")
	return []byte(sb.String()), nil
}

// toAndroidValue formats a token value for Android XML resources.
// Android color resources expect hex values (#AARRGGBB or #RRGGBB),
// not CSS color functions.
func toAndroidValue(tok *token.Token) string {
	value := formatter.ResolvedValue(tok)

	switch tok.Type {
	case token.TypeColor:
		if m, ok := value.(map[string]any); ok {
			return structuredColorToAndroid(m)
		}
	case token.TypeDimension:
		if m, ok := value.(map[string]any); ok {
			if v, hasValue := m["value"]; hasValue && v != nil {
				if u, hasUnit := m["unit"].(string); hasUnit {
					return fmt.Sprintf("%v%s", v, u)
				}
			}
			return formatter.MarshalFallback(m)
		}
	}

	return fmt.Sprintf("%v", value)
}

// structuredColorToAndroid converts a v2025.10 structured color to Android hex format.
func structuredColorToAndroid(m map[string]any) string {
	// Structured color objects are a v2025.10 feature; draft colors are always strings.
	colorVal, err := common.ParseColorValue(m, schema.V2025_10)
	if err != nil {
		return formatter.MarshalFallback(m)
	}

	// If it has a hex field, use it directly
	if obj, ok := colorVal.(*common.ObjectColorValue); ok {
		if obj.Hex != nil && *obj.Hex != "" {
			return *obj.Hex
		}
		// For sRGB, convert to hex
		if obj.ColorSpace == "srgb" && obj.Alpha != nil && *obj.Alpha >= common.AlphaThreshold {
			return colorVal.ToCSS() // ToCSS already returns hex for opaque sRGB
		}
		// For non-sRGB colors, Android doesn't support CSS color functions.
		// Fall back to hex approximation for sRGB, or emit CSS as best-effort.
	}

	return colorVal.ToCSS()
}

func xmlType(tokenType string) string {
	switch tokenType {
	case token.TypeColor:
		return "color"
	case token.TypeDimension:
		return "dimen"
	case token.TypeNumber:
		return "integer"
	case token.TypeString, token.TypeFontFamily:
		return "string"
	default:
		return "string"
	}
}
