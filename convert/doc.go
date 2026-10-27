/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

// Package convert serializes DTCG tokens into output formats.
//
// Serialize produces a DTCG map structure. FormatTokens produces CSS, SCSS,
// JavaScript, TypeScript, Swift, Android XML, flat JSON, or editor snippets;
// ParseFormat and ValidFormats describe the supported names.
//
// # Options
//
// InputSchema and OutputSchema describe the schemas on each side. Their zero
// value is schema.Unknown: InputSchema then defaults to schema.Draft and
// OutputSchema matches InputSchema. Set them explicitly when converting
// between draft and 2025.10 representations.
//
// Prefix is added to generated variable names, so a prefix of "rh" yields
// names such as --rh-color-primary. It does not modify token paths.
//
// # Style Dictionary
//
// Style Dictionary JSON is not DTCG. Converters accept DTCG tokens only.
// Normalize Style Dictionary input into DTCG before parsing and converting.
package convert
