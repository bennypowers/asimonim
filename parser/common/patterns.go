/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package common provides shared utilities for token parsing.
package common

import "regexp"

// Shared regex patterns for token references across parsers.
// These patterns support both JSON (quoted field names) and YAML (unquoted field names).

// CurlyBraceRefPattern matches curly brace token references: {token.reference.path}
var CurlyBraceRefPattern = regexp.MustCompile(`\{([^}]+)\}`)

// JSONPointerRefPattern matches JSON Pointer references in both JSON and YAML:
// JSON: "$ref": "#/path/to/token"
// YAML: $ref: "#/path/to/token" or $ref: '#/path/to/token'
var JSONPointerRefPattern = regexp.MustCompile(`"?\$ref"?\s*:\s*["']?(#[^"'\s]+)["']?`)

// RootKeywordPattern matches $root keyword in token definitions (JSON and YAML):
// JSON: "$root": { ... }
// YAML: $root: { ... }
var RootKeywordPattern = regexp.MustCompile(`"?\$root"?\s*:`)

// SchemaFieldPattern matches the $schema field with its value in JSON and YAML.
// Anchored to line start to match only top-level $schema declarations.
// JSON: "$schema": "https://..."
// YAML: $schema: "https://..." or $schema: 'https://...'
var SchemaFieldPattern = regexp.MustCompile(`(?m)^\s*"?\$schema"?\s*:\s*["']([^"']+)["']`)

// ValueFieldPattern matches $value fields in token definitions.
var ValueFieldPattern = regexp.MustCompile(`"?\$value"?\s*:`)

// TypeFieldPattern matches $type fields in token/group definitions.
var TypeFieldPattern = regexp.MustCompile(`"?\$type"?\s*:`)

// ExtendsFieldPattern matches $extends fields (2025.10 only).
var ExtendsFieldPattern = regexp.MustCompile(`"?\$extends"?\s*:\s*["']?(#[^"'\s]+)["']?`)
