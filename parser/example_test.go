/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

package parser_test

import (
	"fmt"
	"log"

	"bennypowers.dev/asimonim/parser"
)

// This example parses a DTCG document. SchemaVersion is left at its zero
// value so the parser detects the 2025.10 schema from the $schema field.
func ExampleJSONParser_Parse() {
	data := []byte(`{
		"$schema": "https://www.designtokens.org/schemas/2025.10.json",
		"spacing": {
			"small": {
				"$type": "number",
				"$value": 4
			}
		}
	}`)

	p := parser.NewJSONParser()
	tokens, err := p.Parse(data, parser.Options{SkipPositions: true})
	if err != nil {
		log.Fatal(err)
	}

	for _, tok := range tokens {
		fmt.Printf("%s = %s (%s)\n", tok.Name, tok.Value, tok.SchemaVersion)
	}
	// Output: spacing-small = 4 (v2025.10)
}
