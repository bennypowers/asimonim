/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

// Package load provides a high-level API for loading design tokens.
//
// Load resolves a local path, npm: specifier, or jsr: specifier, reads and
// parses the file, resolves $extends relationships and aliases, and returns
// a token.Map.
//
// LoadAll loads several sources, returns a *Result that groups tokens by
// source, and resolves aliases across all sources. Unlike Load, LoadAll does
// not call ResolveGroupExtensions, so tokens inherited through $extends are
// not materialized.
//
// # Configuration
//
// Load reads an optional .config/design-tokens.{yaml,yml,json} file from
// Options.Root and merges it with Options; values set in Options take
// precedence. Network access is off by default; set Options.Fetcher to opt
// Load into a CDN fallback for package specifiers that cannot be resolved
// from the local filesystem. LoadAll uses the *config.Config supplied by the
// caller, does not read a config file, and does not use Options.Fetcher, so
// non-local package specifiers in LoadAll must resolve through the local
// filesystem.
//
// # Schema detection
//
// Options.SchemaVersion has the zero value schema.Unknown. Leave it at the
// zero value to auto-detect from the config file and then from the document
// contents. Set it to schema.Draft or schema.V2025_10 to force a version.
//
// # Prefix
//
// Options.Prefix is the CSS variable prefix applied to loaded tokens. For
// Load it is also the prefix used by the returned token.Map; for LoadAll it
// is stored on the tokens in the returned *Result. Load falls back to the
// Prefix in the config file when Options.Prefix is empty; LoadAll uses only
// the supplied config.
//
// # Example
//
//	m, err := load.Load(ctx, "tokens.json", load.Options{
//		Root: ".",
//		FS:   fs.NewOSFileSystem(),
//	})
//	if err != nil {
//		return err
//	}
//	if tok, ok := m.Get("color.primary"); ok {
//		fmt.Println(tok.CSSVariableName())
//	}
package load
