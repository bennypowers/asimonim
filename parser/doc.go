/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

// Package parser parses DTCG design token files in JSON or YAML.
//
// JSONParser extracts tokens from DTCG-formatted input. The parser accepts
// JSON with comments and trailing commas via jsonc, as well as YAML. Both
// the editor's draft and the stable 2025.10 schema are supported; see the
// schema package for detection and version handling.
//
// # Schema detection
//
// Options.SchemaVersion has the zero value schema.Unknown. When it is left
// at the zero value, Parse inspects the document for a $schema field,
// unambiguous 2025.10 features such as $ref, $extends, resolutionOrder, and
// structured color values, and otherwise falls back to schema.Draft. Set
// SchemaVersion explicitly to override detection, for example when parsing
// fragment files without a $schema declaration.
//
// # Options
//
// Prefix sets Token.Prefix so consumers can derive CSS custom property
// names such as --rh-color-primary. It does not rename tokens or change
// Token.Name; see token.Token.CSSVariableName and token.NewMap.
//
// SkipSort disables the default sorting of keys within each group. The
// default yields deterministic traversal order, which is usually what you
// want for generated files and golden tests. The resulting token list is
// not globally alphabetical: keys are sorted within each group as the
// parser walks the tree. Set SkipSort when order does not matter and
// parsing speed is critical.
//
// SkipPositions disables the second pass that records Line and Character on
// each token. Line is a 0-based line number and Character is a 0-based UTF-16
// offset, matching LSP positions. Position data is needed for editor features
// such as go-to-definition. Set SkipPositions for batch tools that do not
// need it.
//
// # Style Dictionary
//
// Style Dictionary JSON is not DTCG. Its keys omit the "$" prefix ("value"
// and "type" instead of "$value" and "$type"), and composite values such as
// shadows and typography use shapes that differ from DTCG. References use
// the same {group.token} syntax in both formats, but that does not make the
// surrounding document DTCG. Normalize Style Dictionary input into DTCG
// before passing it to this package. The parser does not translate Style
// Dictionary files.
//
// # Example
//
//	p := parser.NewJSONParser()
//	tokens, err := p.Parse(data, parser.Options{SkipPositions: true})
//	if err != nil {
//		return err
//	}
//	for _, tok := range tokens {
//		fmt.Println(tok.Name, tok.Value)
//	}
package parser
