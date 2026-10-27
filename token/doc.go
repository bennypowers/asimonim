/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

// Package token provides DTCG design token types and lookup helpers.
//
// Token is the core type produced by the parser and consumed by the
// resolver, validator, and converters. A token carries its DTCG $value,
// type, metadata, JSON path, schema version, and the result of alias
// resolution.
//
// # Lookup
//
// Map provides prefix-aware lookup. NewMap indexes tokens by CSS custom
// property name. Get accepts a short name (color-primary), a dot path
// (color.primary), or a full CSS name (--rh-color-primary). When the map
// was created with a prefix, short and dot-path lookups automatically gain
// that prefix.
//
// Prefix is applied to a copy of each token that did not already have one.
// A parser-provided prefix is normalized on a copy too, so leading dashes
// ("--rh") still produce --rh-* names and map keys. A token whose prefix is
// explicitly different from the requested prefix keeps it; only its dash
// spelling is normalized. Building a Map does not mutate the input slice.
//
// # Example
//
//	m := token.NewMap(tokens, "rh")
//	tok, ok := m.Get("color.primary")
//	if ok {
//		fmt.Println(tok.CSSVariableName())
//	}
package token
