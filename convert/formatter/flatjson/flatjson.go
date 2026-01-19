/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package flatjson provides flat key-value JSON formatting for design tokens.
package flatjson

import (
	"encoding/json"
	"strings"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/token"
)

// Formatter outputs flat key-value JSON.
type Formatter struct{}

// New creates a new flat JSON formatter.
func New() *Formatter {
	return &Formatter{}
}

// Format converts tokens to flat key-value JSON.
func (f *Formatter) Format(tokens []*token.Token, opts formatter.Options) ([]byte, error) {
	delimiter := opts.Delimiter
	if delimiter == "" {
		delimiter = "-"
	}

	result := make(map[string]any)
	for _, tok := range tokens {
		key := formatter.ApplyPrefix(strings.Join(tok.Path, delimiter), opts.Prefix, delimiter)
		result[key] = formatter.ResolvedValue(tok)
	}

	return json.MarshalIndent(result, "", "  ")
}
