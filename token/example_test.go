/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

package token_test

import (
	"fmt"

	"bennypowers.dev/asimonim/token"
)

// This example looks up a token by dot path. The map applies the "rh"
// prefix, so the lookup accepts either the short name or the full CSS
// custom property name.
func ExampleMap_Get() {
	tokens := []*token.Token{
		{Name: "color-primary", Value: "#ff0000"},
	}

	m := token.NewMap(tokens, "rh")

	tok, ok := m.Get("color.primary")
	fmt.Println(ok, tok.CSSVariableName(), tok.Value)
	// Output: true --rh-color-primary #ff0000
}
