/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

package resolver_test

import (
	"fmt"
	"log"

	"bennypowers.dev/asimonim/resolver"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/token"
)

// This example resolves a curly brace reference. ResolveAliases fills in
// ResolvedValue and marks each token as resolved.
func ExampleResolveAliases() {
	tokens := []*token.Token{
		{Name: "color-base", Value: "#ff0000", SchemaVersion: schema.V2025_10},
		{Name: "color-primary", Value: "{color.base}", SchemaVersion: schema.V2025_10},
	}

	if err := resolver.ResolveAliases(tokens, schema.V2025_10); err != nil {
		log.Fatal(err)
	}

	for _, tok := range tokens {
		if tok.Name == "color-primary" {
			fmt.Printf("%s = %v\n", tok.Name, tok.ResolvedValue)
		}
	}
	// Output: color-primary = #ff0000
}
