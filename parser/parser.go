/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

package parser

import (
	"bennypowers.dev/asimonim/fs"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/token"
)

// Options configures token parsing.
type Options struct {
	// Prefix is the CSS variable prefix applied to parsed tokens.
	// It populates Token.Prefix so callers can produce names such as
	// --rh-color-primary. It does not rename tokens or change Token.Name.
	Prefix string

	// SchemaVersion selects the DTCG schema version.
	// The zero value, schema.Unknown, enables auto-detection from the
	// document contents. Set schema.Draft or schema.V2025_10 to override
	// detection.
	SchemaVersion schema.Version

	// GroupMarkers are token names that can be both tokens and groups (draft only).
	GroupMarkers []string

	// SkipSort disables alphabetical sorting of tokens for better performance.
	// When false (default), tokens are sorted by name for deterministic
	// output order, which is desirable for generated files and golden tests.
	SkipSort bool

	// SkipPositions disables position tracking for better performance.
	// When true, Line and Character fields will be zero on all tokens.
	// Use this when LSP features such as go-to-definition are not needed.
	SkipPositions bool
}

// Parser parses design token files.
type Parser interface {
	// Parse parses token data and returns tokens.
	Parse(data []byte, opts Options) ([]*token.Token, error)

	// ParseFile parses a token file and returns tokens.
	ParseFile(filesystem fs.FileSystem, path string, opts Options) ([]*token.Token, error)
}
