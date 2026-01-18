/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package token provides DTCG design token types.
package token

import (
	"strings"

	"bennypowers.dev/asimonim/schema"
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
	Deprecated bool `json:"deprecated,omitempty"`

	// DeprecationMessage provides context for deprecated tokens.
	DeprecationMessage string `json:"deprecationMessage,omitempty"`

	// FilePath is the file this token was loaded from.
	FilePath string `json:"-"`

	// Prefix is the CSS variable prefix for this token.
	Prefix string `json:"-"`

	// Path is the JSON path to this token (e.g., ["color", "primary"]).
	Path []string `json:"-"`

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
}

// CSSVariableName returns the CSS custom property name for this token.
// e.g., "--color-primary" or "--my-prefix-color-primary"
func (t *Token) CSSVariableName() string {
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
