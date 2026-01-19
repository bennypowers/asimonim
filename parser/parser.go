/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package parser provides DTCG token file parsing.
package parser

import (
	"bennypowers.dev/asimonim/fs"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/token"
)

// Options configures token parsing.
type Options struct {
	// Prefix is the CSS variable prefix.
	Prefix string

	// SchemaVersion overrides auto-detection.
	SchemaVersion schema.Version

	// GroupMarkers are token names that can be both tokens and groups (draft only).
	GroupMarkers []string

	// SkipSort disables alphabetical sorting of tokens for better performance.
	// When false (default), tokens are sorted for deterministic output order.
	SkipSort bool
}

// Parser parses design token files.
type Parser interface {
	// Parse parses token data and returns tokens.
	Parse(data []byte, opts Options) ([]*token.Token, error)

	// ParseFile parses a token file and returns tokens.
	ParseFile(filesystem fs.FileSystem, path string, opts Options) ([]*token.Token, error)
}
