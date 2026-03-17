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
// and go-colorful for XYZ-based conversions. Wide-gamut RGB spaces
// (display-p3, a98-rgb, prophoto-rgb, rec2020) are converted through
// XYZ D65 using standard matrices from the CSS Color Level 4 specification.
// Out-of-gamut sRGB values are clamped to [0,1] after conversion.
package android

import (
	"fmt"
	"math"
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

	obj, ok := colorVal.(*common.ObjectColorValue)
	if !ok {
		return colorVal.ToCSS()
	}

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

	if obj.ColorSpace != "srgb" {
		logger.Warn("downsampling %s from %s to sRGB for Android", tokenName, obj.ColorSpace)
	}

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
// Uses csscolorparser.FromLinearRGB for srgb-linear, go-colorful for XYZ,
// and standard CSS Color 4 matrices for display-p3, a98-rgb, prophoto-rgb,
// and rec2020.
func colorSpaceToSRGB(space string, c []float64) (r, g, b float64) {
	switch space {
	case "srgb-linear":
		col := csscolorparser.FromLinearRGB(c[0], c[1], c[2], 1.0)
		return col.R, col.G, col.B

	case "xyz-d65":
		col := colorful.Xyz(c[0], c[1], c[2])
		return clampF(col.R), clampF(col.G), clampF(col.B)

	case "xyz-d50":
		// go-colorful uses D65 by default; adapt from D50 using Bradford matrix
		xd65, yd65, zd65 := d50ToD65(c[0], c[1], c[2])
		col := colorful.Xyz(xd65, yd65, zd65)
		return clampF(col.R), clampF(col.G), clampF(col.B)

	case "display-p3":
		// Display P3 uses sRGB transfer function; linearize then matrix to XYZ D65
		x, y, z := linearToXYZ(srgbToLinear(c[0]), srgbToLinear(c[1]), srgbToLinear(c[2]), matDisplayP3ToXYZ)
		col := colorful.Xyz(x, y, z)
		return clampF(col.R), clampF(col.G), clampF(col.B)

	case "a98-rgb":
		x, y, z := linearToXYZ(a98ToLinear(c[0]), a98ToLinear(c[1]), a98ToLinear(c[2]), matA98RGBToXYZ)
		col := colorful.Xyz(x, y, z)
		return clampF(col.R), clampF(col.G), clampF(col.B)

	case "prophoto-rgb":
		// ProPhoto uses D50 illuminant
		x, y, z := linearToXYZ(prophotoToLinear(c[0]), prophotoToLinear(c[1]), prophotoToLinear(c[2]), matProPhotoToXYZD50)
		xd65, yd65, zd65 := d50ToD65(x, y, z)
		col := colorful.Xyz(xd65, yd65, zd65)
		return clampF(col.R), clampF(col.G), clampF(col.B)

	case "rec2020":
		x, y, z := linearToXYZ(rec2020ToLinear(c[0]), rec2020ToLinear(c[1]), rec2020ToLinear(c[2]), matRec2020ToXYZ)
		col := colorful.Xyz(x, y, z)
		return clampF(col.R), clampF(col.G), clampF(col.B)

	default:
		// Unknown space; clamp raw values as best effort
		return clampF(c[0]), clampF(c[1]), clampF(c[2])
	}
}

// --- Transfer functions ---
// Standard OETF/EOTF from CSS Color Level 4 §12.

func srgbToLinear(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func a98ToLinear(c float64) float64 {
	sign := 1.0
	if c < 0 {
		sign = -1.0
		c = -c
	}
	return sign * math.Pow(c, 563.0/256.0)
}

func prophotoToLinear(c float64) float64 {
	if c <= 16.0/512.0 {
		return c / 16.0
	}
	return math.Pow(c, 1.8)
}

func rec2020ToLinear(c float64) float64 {
	const alpha = 1.09929682680944
	const beta = 0.018053968510807
	if c < beta*4.5 {
		return c / 4.5
	}
	return math.Pow((c+alpha-1)/alpha, 1.0/0.45)
}

// --- Matrix conversions ---
// Primaries→XYZ matrices from CSS Color Level 4 §10.
// https://www.w3.org/TR/css-color-4/#color-conversion-code

type mat3 [3][3]float64

var matDisplayP3ToXYZ = mat3{
	{0.4865709486482162, 0.26566769316909306, 0.1982172852343625},
	{0.2289745640697488, 0.6917385218365064, 0.079286914093745},
	{0.0, 0.04511338185890264, 1.043944368900976},
}

var matA98RGBToXYZ = mat3{
	{0.5766690429101305, 0.1855582379065463, 0.1882286462349947},
	{0.29734497525053605, 0.6273635662554661, 0.07529145849399788},
	{0.02703136138641234, 0.07068885253582723, 0.9913375368376388},
}

var matProPhotoToXYZD50 = mat3{
	{0.7977604896723027, 0.13518583717574031, 0.0313493495815248},
	{0.2880711282292934, 0.7118432178101014, 0.00008565396060525902},
	{0.0, 0.0, 0.8251046025104602},
}

var matRec2020ToXYZ = mat3{
	{0.6369580483012914, 0.14461690358620832, 0.1688809751641721},
	{0.2627002120112671, 0.6779980715188708, 0.05930171646986196},
	{0.0, 0.028072693049087428, 1.0609850577107909},
}

// Bradford chromatic adaptation D50→D65.
var matD50ToD65 = mat3{
	{0.9555766, -0.0230393, 0.0631636},
	{-0.0282895, 1.0099416, 0.0210077},
	{0.0122982, -0.0204830, 1.3299098},
}

func linearToXYZ(r, g, b float64, m mat3) (x, y, z float64) {
	return m[0][0]*r + m[0][1]*g + m[0][2]*b,
		m[1][0]*r + m[1][1]*g + m[1][2]*b,
		m[2][0]*r + m[2][1]*g + m[2][2]*b
}

func d50ToD65(x, y, z float64) (float64, float64, float64) {
	m := matD50ToD65
	return m[0][0]*x + m[0][1]*y + m[0][2]*z,
		m[1][0]*x + m[1][1]*y + m[1][2]*z,
		m[2][0]*x + m[2][1]*y + m[2][2]*z
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
