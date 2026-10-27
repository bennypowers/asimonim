/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

// Package resolver resolves design token references and group extensions.
//
// Resolution happens after parsing. For 2025.10 documents, call
// ResolveGroupExtensions first to materialize inherited tokens, then call
// ResolveAliases. Draft documents do not use $extends, so ResolveAliases
// alone is sufficient.
//
// ResolveAliases walks the dependency graph in topological order, detects
// cycles, and updates each token's ResolvedValue, ResolutionChain, and
// IsResolved fields. Curly brace references such as {color.primary} work in
// every schema version. JSON Pointer references such as #/color/primary are
// only treated as references outside the draft schema.
//
// The version argument is the fallback schema version. Tokens that carry
// their own SchemaVersion use that value, which supports workspaces that mix
// draft and 2025.10 files.
//
// # Example
//
//	tokens, err := parser.NewJSONParser().Parse(data, parser.Options{})
//	if err != nil {
//		return err
//	}
//	if err := resolver.ResolveAliases(tokens, schema.Draft); err != nil {
//		return err
//	}
//	if len(tokens) > 0 {
//		fmt.Println(tokens[0].ResolvedValue)
//	}
package resolver
