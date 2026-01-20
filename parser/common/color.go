/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package common

import (
	"fmt"
	"strings"

	"bennypowers.dev/asimonim/schema"
)

// AlphaThreshold is the value below which alpha is included in CSS output.
// Values >= 0.999 are treated as fully opaque to avoid unnecessary alpha channels.
const AlphaThreshold = 0.999

// ValidColorSpaces lists the 14 color spaces supported by DTCG 2025.10 spec.
var ValidColorSpaces = map[string]bool{
	"srgb":         true,
	"display-p3":   true,
	"a98-rgb":      true,
	"prophoto-rgb": true,
	"rec2020":      true,
	"xyz-d50":      true,
	"xyz-d65":      true,
	"lab":          true,
	"lch":          true,
	"oklab":        true,
	"oklch":        true,
	"srgb-linear":  true,
	"hsl":          true,
	"hwb":          true,
}

// ColorValue represents a color token value in any schema format.
type ColorValue interface {
	ToCSS() string
	Version() schema.Version
	IsValid() bool
}

// StringColorValue represents a draft schema color value (string format).
type StringColorValue struct {
	Value  string
	Schema schema.Version
}

// ToCSS returns the CSS representation of the color.
func (s *StringColorValue) ToCSS() string {
	return s.Value
}

// Version returns the schema version for this color.
func (s *StringColorValue) Version() schema.Version {
	return s.Schema
}

// IsValid returns true if the color value is valid.
func (s *StringColorValue) IsValid() bool {
	return s.Value != ""
}

// ObjectColorValue represents a 2025.10 schema color value (structured format).
type ObjectColorValue struct {
	ColorSpace string
	Components []any // Can be float64 or "none" keyword
	Alpha      *float64
	Hex        *string
	Schema     schema.Version
}

// ToCSS returns the CSS representation of the color.
func (o *ObjectColorValue) ToCSS() string {
	// If hex field is provided, use it
	if o.Hex != nil && *o.Hex != "" {
		return *o.Hex
	}

	// For sRGB without hex field, convert to hex format
	if o.ColorSpace == "srgb" && o.canConvertToHex() {
		return o.toHex()
	}

	// Build components string using strings.Builder
	var sb strings.Builder
	for i, comp := range o.Components {
		if i > 0 {
			sb.WriteString(" ")
		}
		switch v := comp.(type) {
		case float64:
			sb.WriteString(fmt.Sprintf("%.4g", v))
		case string:
			sb.WriteString(v) // "none" keyword
		default:
			sb.WriteString(fmt.Sprintf("%v", v))
		}
	}
	compStr := sb.String()

	hasAlpha := o.Alpha != nil && *o.Alpha < AlphaThreshold

	// Color spaces that have native CSS functions (more widely supported than color())
	switch o.ColorSpace {
	case "hsl", "hwb", "lab", "lch", "oklab", "oklch":
		if hasAlpha {
			return fmt.Sprintf("%s(%s / %.4g)", o.ColorSpace, compStr, *o.Alpha)
		}
		return fmt.Sprintf("%s(%s)", o.ColorSpace, compStr)
	default:
		// Generate CSS color() function with optional alpha
		if hasAlpha {
			return fmt.Sprintf("color(%s %s / %.4g)", o.ColorSpace, compStr, *o.Alpha)
		}
		return fmt.Sprintf("color(%s %s)", o.ColorSpace, compStr)
	}
}

// canConvertToHex returns true if this sRGB color can be converted to hex.
// Requires exactly 3 numeric components in 0-1 range and alpha >= threshold.
func (o *ObjectColorValue) canConvertToHex() bool {
	if len(o.Components) != 3 {
		return false
	}

	// Check for alpha that would require rgba format
	if o.Alpha != nil && *o.Alpha < AlphaThreshold {
		return false
	}

	// All components must be numeric (not "none")
	for _, comp := range o.Components {
		if _, ok := comp.(float64); !ok {
			return false
		}
	}

	return true
}

// toHex converts sRGB components to hex format (#RRGGBB).
func (o *ObjectColorValue) toHex() string {
	r := clamp(int(o.Components[0].(float64)*255+0.5), 0, 255)
	g := clamp(int(o.Components[1].(float64)*255+0.5), 0, 255)
	b := clamp(int(o.Components[2].(float64)*255+0.5), 0, 255)
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

// clamp restricts a value to the given range.
func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// Version returns the schema version for this color.
func (o *ObjectColorValue) Version() schema.Version {
	return o.Schema
}

// IsValid returns true if the color value is valid.
func (o *ObjectColorValue) IsValid() bool {
	return o.ColorSpace != "" && len(o.Components) > 0
}

// ParseColorValue parses a color value according to the schema version.
func ParseColorValue(value any, version schema.Version) (ColorValue, error) {
	switch version {
	case schema.Draft:
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("draft schema expects string color value, got %T", value)
		}
		return &StringColorValue{
			Value:  str,
			Schema: schema.Draft,
		}, nil

	case schema.V2025_10:
		obj, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("v2025_10 schema expects structured color object, got %T", value)
		}

		colorSpace, ok := obj["colorSpace"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid colorSpace field")
		}

		componentsRaw, ok := obj["components"]
		if !ok {
			return nil, fmt.Errorf("missing components field")
		}

		components, ok := componentsRaw.([]any)
		if !ok {
			return nil, fmt.Errorf("components must be an array")
		}

		// Validate component values
		for i, comp := range components {
			switch v := comp.(type) {
			case float64:
				// Valid
			case string:
				if v != "none" {
					return nil, fmt.Errorf("component[%d]: invalid string %q; only \"none\" allowed", i, v)
				}
			default:
				return nil, fmt.Errorf("component[%d]: invalid type %T", i, comp)
			}
		}

		var alpha *float64
		if alphaRaw, exists := obj["alpha"]; exists {
			if alphaVal, ok := alphaRaw.(float64); ok {
				alpha = &alphaVal
			}
		}

		var hex *string
		if hexRaw, exists := obj["hex"]; exists {
			if hexVal, ok := hexRaw.(string); ok {
				hex = &hexVal
			}
		}

		return &ObjectColorValue{
			ColorSpace: colorSpace,
			Components: components,
			Alpha:      alpha,
			Hex:        hex,
			Schema:     schema.V2025_10,
		}, nil

	default:
		return nil, fmt.Errorf("unknown schema version: %v", version)
	}
}
