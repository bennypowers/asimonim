/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package android provides Android XML resource formatting for design tokens.
//
// Android color resources only accept hex values (#RRGGBB or #AARRGGBB).
// Non-sRGB colors are downsampled to sRGB with a warning, using
// csscolorparser for perceptual spaces (oklch, oklab, lab, lch, hsl, hwb)
// and go-colorful for wide-gamut RGB and XYZ color space conversions.
// Out-of-gamut sRGB values are clamped to [0,1] after conversion.
package android

import (
	"encoding/json"
	"fmt"
	"strings"

	colorful "github.com/lucasb-eyer/go-colorful"
	"github.com/mazznoer/csscolorparser"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/internal/logger"
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
func toAndroidValue(tok *token.Token) string {
	value := formatter.ResolvedValue(tok)

	switch tok.Type {
	case token.TypeColor:
		if m, ok := value.(map[string]any); ok {
			return structuredColorToAndroid(m, tok.Name)
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

	switch v := value.(type) {
	case map[string]any:
		return formatter.MarshalFallback(v)
	case []any:
		if data, err := json.Marshal(v); err == nil {
			return string(data)
		}
	}

	return fmt.Sprintf("%v", value)
}

// structuredColorToAndroid converts a v2025.10 structured color to Android hex.
// All colors are converted to sRGB hex (#RRGGBB or #AARRGGBB).
// Non-sRGB color spaces are downsampled with a warning.
func structuredColorToAndroid(m map[string]any, tokenName string) string {
	// Structured color objects are a v2025.10 feature; draft colors are always strings.
	colorVal, err := common.ParseColorValue(m, schema.V2025_10)
	if err != nil {
		return formatter.MarshalFallback(m)
	}

	obj := colorVal.(*common.ObjectColorValue)

	// If it has a hex field, use it directly
	if obj.Hex != nil && *obj.Hex != "" {
		return *obj.Hex
	}

	// Extract numeric components ("none" → 0)
	components := make([]float64, len(obj.Components))
	for i, c := range obj.Components {
		if v, ok := c.(float64); ok {
			components[i] = v
		}
	}
	if len(components) < 3 {
		return formatter.MarshalFallback(m)
	}

	alpha := 1.0
	if obj.Alpha != nil {
		alpha = *obj.Alpha
	}

	// sRGB: direct conversion, no downsampling needed
	if obj.ColorSpace == "srgb" {
		return formatAndroidHex(components[0], components[1], components[2], alpha)
	}

	logger.Warn("downsampling %s from %s to sRGB for Android", tokenName, obj.ColorSpace)

	// Try csscolorparser first — handles oklch, oklab, hsl, hwb, lab, lch
	css := colorVal.ToCSS()
	if c, err := csscolorparser.Parse(css); err == nil {
		return formatAndroidHex(c.R, c.G, c.B, c.A)
	}

	// For color() function spaces, use targeted conversions
	r, g, b := colorSpaceToSRGB(obj.ColorSpace, components)
	return formatAndroidHex(r, g, b, alpha)
}

// formatAndroidHex formats sRGB [0,1] components as Android hex color.
func formatAndroidHex(r, g, b, a float64) string {
	ri := clamp(int(r*255+0.5), 0, 255)
	gi := clamp(int(g*255+0.5), 0, 255)
	bi := clamp(int(b*255+0.5), 0, 255)
	if a < common.AlphaThreshold {
		ai := clamp(int(a*255+0.5), 0, 255)
		return fmt.Sprintf("#%02X%02X%02X%02X", ai, ri, gi, bi)
	}
	return fmt.Sprintf("#%02X%02X%02X", ri, gi, bi)
}

// colorSpaceToSRGB converts components from wide-gamut color spaces to sRGB.
// Uses csscolorparser for srgb-linear and go-colorful for all other spaces.
func colorSpaceToSRGB(space string, c []float64) (r, g, b float64) {
	switch space {
	case "srgb-linear":
		col := csscolorparser.FromLinearRGB(c[0], c[1], c[2], 1.0)
		return col.R, col.G, col.B

	case "xyz-d65":
		col := colorful.Xyz(c[0], c[1], c[2])
		return clampF(col.R), clampF(col.G), clampF(col.B)

	case "xyz-d50":
		col := colorful.XyzD50(c[0], c[1], c[2])
		return clampF(col.R), clampF(col.G), clampF(col.B)

	case "display-p3":
		col := colorful.DisplayP3(c[0], c[1], c[2])
		return clampF(col.R), clampF(col.G), clampF(col.B)

	case "a98-rgb":
		col := colorful.A98Rgb(c[0], c[1], c[2])
		return clampF(col.R), clampF(col.G), clampF(col.B)

	case "prophoto-rgb":
		col := colorful.ProPhotoRgb(c[0], c[1], c[2])
		return clampF(col.R), clampF(col.G), clampF(col.B)

	case "rec2020":
		col := colorful.Rec2020(c[0], c[1], c[2])
		return clampF(col.R), clampF(col.G), clampF(col.B)

	default:
		// Unknown space; clamp raw values as best effort
		return clampF(c[0]), clampF(c[1]), clampF(c[2])
	}
}

func clamp(v, lo, hi int) int     { return max(lo, min(hi, v)) }
func clampF(v float64) float64    { return max(0, min(1, v)) }

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
