/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package dtcg provides DTCG-compliant JSON formatting for design tokens.
package dtcg

import (
	"encoding/json"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/token"
)

// Formatter outputs DTCG-compliant JSON.
type Formatter struct {
	// Serialize is the function used to convert tokens to DTCG map structure.
	// This allows the formatter to use the serialization logic from the convert package.
	Serialize func(tokens []*token.Token) map[string]any
}

// New creates a new DTCG formatter with the given serialization function.
func New(serialize func(tokens []*token.Token) map[string]any) *Formatter {
	return &Formatter{Serialize: serialize}
}

// Format converts tokens to DTCG-compliant JSON.
func (f *Formatter) Format(tokens []*token.Token, _ formatter.Options) ([]byte, error) {
	result := f.Serialize(tokens)
	return json.MarshalIndent(result, "", "  ")
}
