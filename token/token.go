/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package token provides DTCG design token types.
package token

import (
	"encoding/json"
	"fmt"
	"strings"

	"bennypowers.dev/asimonim/parser/common"
	"bennypowers.dev/asimonim/schema"
)

// DTCG token type constants.
// See: https://design-tokens.github.io/community-group/format/#types
const (
	TypeColor       = "color"
	TypeDimension   = "dimension"
	TypeFontFamily  = "fontFamily"
	TypeFontWeight  = "fontWeight"
	TypeDuration    = "duration"
	TypeCubicBezier = "cubicBezier"
	TypeNumber      = "number"
	TypeString      = "string"
	TypeStrokeStyle = "strokeStyle"
	TypeBorder      = "border"
	TypeTransition  = "transition"
	TypeShadow      = "shadow"
	TypeGradient    = "gradient"
	TypeTypography  = "typography"
)

// Token represents a design token following the DTCG specification.
// See: https://design-tokens.github.io/community-group/format/
type Token struct {
	// Name is the token's identifier (e.g., "color-primary").
	Name string `json:"name"`

	// Value is the resolved value of the token.
	Value string `json:"$value"`

	// Type specifies the type of token (color, dimension, etc.).
	Type string `json:"$type,omitempty"`

	// Description is optional documentation for the token.
	Description string `json:"$description,omitempty"`

	// Extensions allows for custom metadata.
	Extensions map[string]any `json:"$extensions,omitempty"`

	// Deprecated indicates if this token should no longer be used.
	Deprecated bool `json:"$deprecated,omitempty"`

	// DeprecationMessage provides context for deprecated tokens.
	DeprecationMessage string `json:"$deprecationMessage,omitempty"`

	// FilePath is the file this token was loaded from.
	FilePath string `json:"-"`

	// Prefix is the CSS variable prefix for this token.
	Prefix string `json:"-"`

	// Path is the JSON path to this token (e.g., ["color", "primary"]).
	Path []string `json:"-"`

	// DefinitionURI is the file URI where this token is defined.
	// This is typically set by LSP servers for go-to-definition support.
	DefinitionURI string `json:"-"`

	// Line is the 0-based line number where this token is defined.
	Line uint32 `json:"-"`

	// Character is the 0-based character offset where this token is defined.
	Character uint32 `json:"-"`

	// Reference is the original reference format (e.g., "{color.primary}").
	Reference string `json:"-"`

	// SchemaVersion is the detected schema version for this token.
	SchemaVersion schema.Version `json:"-"`

	// RawValue is the original $value before resolution.
	RawValue any `json:"-"`

	// ResolvedValue is the value after alias/extends resolution.
	ResolvedValue any `json:"-"`

	// IsResolved indicates if alias resolution has been performed.
	IsResolved bool `json:"-"`

	// ResolutionChain contains the token names in the resolution chain.
	// For example, if A references B which references C, A's chain is [B, C].
	// Empty if this token is not an alias.
	ResolutionChain []string `json:"-"`
}

// CSSVariableName returns the CSS custom property name for this token.
// e.g., "--color-primary" or "--my-prefix-color-primary"
// Returns an empty string if the token has no name.
func (t *Token) CSSVariableName() string {
	if t.Name == "" {
		return ""
	}
	name := strings.ReplaceAll(t.Name, ".", "-")
	if t.Prefix != "" {
		prefix := strings.ReplaceAll(t.Prefix, ".", "-")
		return "--" + prefix + "-" + name
	}
	return "--" + name
}

// DotPath returns the dot-separated path to this token.
func (t *Token) DotPath() string {
	return strings.Join(t.Path, ".")
}

// CSSSyntax returns the CSS syntax string for this token's type.
// For example, a "color" token returns "<color>".
// Returns "<custom-ident>" for unknown types.
func (t *Token) CSSSyntax() string {
	return TypeToCSSSyntax(t.Type)
}

// TypeToCSSSyntax maps a DTCG token type to its CSS syntax string.
// This is useful for generating CSS @property rules or custom property definitions.
// Returns "<custom-ident>" for unknown types as a safe fallback.
func TypeToCSSSyntax(tokenType string) string {
	switch tokenType {
	case TypeColor:
		return "<color>"
	case TypeDimension:
		return "<length>"
	case TypeNumber:
		return "<number>"
	case TypeString:
		return "<custom-ident>"
	case TypeFontFamily:
		return "<custom-ident>+"
	case TypeFontWeight:
		return "<number>"
	case TypeDuration:
		return "<time>"
	case TypeCubicBezier:
		return "<easing-function>"
	case TypeShadow:
		return "<shadow>"
	case TypeBorder:
		return "<line-width> || <line-style> || <color>"
	case TypeGradient:
		return "<image>"
	case TypeTypography:
		return "<custom-ident>" // Complex composite type
	case TypeStrokeStyle:
		return "<line-style>"
	case TypeTransition:
		return "<time> || <easing-function>"
	default:
		return "<custom-ident>" // Fallback for unknown types
	}
}

// DisplayValue returns a formatted string for display in hover/UI.
// It uses ResolvedValue if resolved, otherwise RawValue if set, else Value.
// The value is formatted based on the token's Type for human readability.
func (t *Token) DisplayValue() string {
	// Determine which value to use
	var val any
	if t.IsResolved && t.ResolvedValue != nil {
		val = t.ResolvedValue
	} else if t.RawValue != nil {
		val = t.RawValue
	} else {
		return t.Value
	}

	return t.formatValue(val)
}

// formatValue formats a value for human-readable display based on token type.
func (t *Token) formatValue(val any) string {
	if val == nil {
		return ""
	}

	// Handle string values directly
	if s, ok := val.(string); ok {
		return s
	}

	// Handle type-specific structured values
	switch t.Type {
	case TypeColor:
		if colorVal, err := common.ParseColorValue(val, t.SchemaVersion); err == nil {
			return colorVal.ToCSS()
		}
	case TypeDimension:
		if s := formatDimension(val); s != "" {
			return s
		}
	case TypeDuration:
		if s := formatDuration(val); s != "" {
			return s
		}
	case TypeCubicBezier:
		if s := formatCubicBezier(val); s != "" {
			return s
		}
	case TypeFontFamily:
		if s := formatFontFamily(val); s != "" {
			return s
		}
	case TypeShadow:
		if s := formatShadow(val); s != "" {
			return s
		}
	case TypeBorder:
		if s := formatBorder(val); s != "" {
			return s
		}
	case TypeTransition:
		if s := formatTransition(val); s != "" {
			return s
		}
	}

	// Handle maps and arrays with JSON serialization as fallback
	switch v := val.(type) {
	case map[string]any:
		if data, err := json.Marshal(v); err == nil {
			return string(data)
		}
		return fmt.Sprintf("%v", v)
	case []any:
		if data, err := json.Marshal(v); err == nil {
			return string(data)
		}
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// formatDimension formats a structured dimension value like {"value": 0.5, "unit": "rem"} to "0.5rem".
func formatDimension(val any) string {
	m, ok := val.(map[string]any)
	if !ok {
		return ""
	}
	v, hasValue := m["value"]
	u, hasUnit := m["unit"].(string)
	if !hasValue || !hasUnit {
		return ""
	}
	return fmt.Sprintf("%v%s", v, u)
}

// formatDuration formats a structured duration value like {"value": 100, "unit": "ms"} to "100ms".
func formatDuration(val any) string {
	m, ok := val.(map[string]any)
	if !ok {
		return ""
	}
	v, hasValue := m["value"]
	u, hasUnit := m["unit"].(string)
	if !hasValue || !hasUnit {
		return ""
	}
	return fmt.Sprintf("%v%s", v, u)
}

// formatCubicBezier formats an array [x1, y1, x2, y2] to "cubic-bezier(x1, y1, x2, y2)".
func formatCubicBezier(val any) string {
	arr, ok := val.([]any)
	if !ok || len(arr) != 4 {
		return ""
	}
	// Verify all elements are numeric
	for _, v := range arr {
		switch v.(type) {
		case int, int64, float64:
			continue
		default:
			return ""
		}
	}
	return fmt.Sprintf("cubic-bezier(%v, %v, %v, %v)", arr[0], arr[1], arr[2], arr[3])
}

// formatFontFamily formats a font family value. Handles both string and array formats.
func formatFontFamily(val any) string {
	switch v := val.(type) {
	case string:
		return v
	case []any:
		if len(v) == 0 {
			return ""
		}
		parts := make([]string, 0, len(v))
		for _, f := range v {
			if s, ok := f.(string); ok {
				// Quote font names that contain spaces
				if strings.Contains(s, " ") {
					parts = append(parts, fmt.Sprintf("%q", s))
				} else {
					parts = append(parts, s)
				}
			}
		}
		return strings.Join(parts, ", ")
	default:
		return ""
	}
}

// formatShadow formats a shadow value to CSS box-shadow format.
// Handles both single shadow objects and arrays of shadows.
func formatShadow(val any) string {
	switch v := val.(type) {
	case map[string]any:
		return formatSingleShadow(v)
	case []any:
		shadows := make([]string, 0, len(v))
		for _, s := range v {
			if m, ok := s.(map[string]any); ok {
				if shadow := formatSingleShadow(m); shadow != "" {
					shadows = append(shadows, shadow)
				}
			}
		}
		if len(shadows) == 0 {
			return ""
		}
		return strings.Join(shadows, ", ")
	default:
		return ""
	}
}

// formatSingleShadow formats a single shadow object to CSS.
func formatSingleShadow(m map[string]any) string {
	offsetX := formatDimensionField(m["offsetX"])
	offsetY := formatDimensionField(m["offsetY"])
	blur := formatDimensionField(m["blur"])
	spread := formatDimensionField(m["spread"])
	color := formatColorField(m["color"])

	if offsetX == "" || offsetY == "" || blur == "" || color == "" {
		return ""
	}

	if spread != "" && spread != "0px" && spread != "0rem" {
		return fmt.Sprintf("%s %s %s %s %s", offsetX, offsetY, blur, spread, color)
	}
	return fmt.Sprintf("%s %s %s %s", offsetX, offsetY, blur, color)
}

// formatBorder formats a border value to CSS border shorthand.
func formatBorder(val any) string {
	m, ok := val.(map[string]any)
	if !ok {
		return ""
	}

	width := formatDimensionField(m["width"])
	style := formatStyleField(m["style"])
	color := formatColorField(m["color"])

	if width == "" || style == "" || color == "" {
		return ""
	}

	return fmt.Sprintf("%s %s %s", width, style, color)
}

// formatTransition formats a transition value to CSS transition format.
func formatTransition(val any) string {
	m, ok := val.(map[string]any)
	if !ok {
		return ""
	}

	duration := formatDurationField(m["duration"])
	timing := formatTimingField(m["timingFunction"])

	if duration == "" || timing == "" {
		return ""
	}

	delay := formatDurationField(m["delay"])
	if delay != "" && delay != "0ms" && delay != "0s" {
		return fmt.Sprintf("%s %s %s", duration, timing, delay)
	}
	return fmt.Sprintf("%s %s", duration, timing)
}

// Helper functions for formatting composite type fields

func formatDimensionField(val any) string {
	switch v := val.(type) {
	case string:
		return v
	case map[string]any:
		return formatDimension(v)
	default:
		return ""
	}
}

func formatDurationField(val any) string {
	switch v := val.(type) {
	case string:
		return v
	case map[string]any:
		return formatDuration(v)
	default:
		return ""
	}
}

func formatColorField(val any) string {
	switch v := val.(type) {
	case string:
		return v
	case map[string]any:
		// Try to parse as structured color
		if colorVal, err := common.ParseColorValue(v, schema.V2025_10); err == nil {
			return colorVal.ToCSS()
		}
		return ""
	default:
		return ""
	}
}

func formatStyleField(val any) string {
	switch v := val.(type) {
	case string:
		return v
	case map[string]any:
		// strokeStyle object format - just return the dashArray description
		if dashArray, ok := v["dashArray"]; ok {
			return fmt.Sprintf("dash:%v", dashArray)
		}
		return ""
	default:
		return ""
	}
}

func formatTimingField(val any) string {
	switch v := val.(type) {
	case string:
		return v
	case []any:
		return formatCubicBezier(v)
	default:
		return ""
	}
}
