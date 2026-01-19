/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package token

import (
	"regexp"
	"strings"
)

// Reference represents a reference to another token.
type Reference struct {
	// Raw is the original reference string.
	Raw string

	// TokenPath is the resolved token path being referenced.
	TokenPath string

	// Kind indicates the reference format.
	Kind ReferenceKind

	// Line is the 0-based line number where this reference appears.
	Line uint32

	// Character is the 0-based character offset where this reference appears.
	Character uint32
}

// ReferenceKind indicates the type of reference.
type ReferenceKind int

const (
	// RefCurlyBrace is a curly brace reference: {color.primary}
	RefCurlyBrace ReferenceKind = iota

	// RefJSONPointer is a JSON pointer reference: {"$ref": "#/color/primary"}
	RefJSONPointer
)

var (
	// curlyBracePattern matches {token.path} references.
	curlyBracePattern = regexp.MustCompile(`\{([^{}]+)\}`)

	// jsonPointerPattern matches JSON pointer format: #/path/to/token
	jsonPointerPattern = regexp.MustCompile(`^#/(.+)$`)
)

// ParseCurlyBraceRef extracts the token path from a curly brace reference.
// Returns the path and true if valid, empty string and false otherwise.
func ParseCurlyBraceRef(value string) (string, bool) {
	matches := curlyBracePattern.FindStringSubmatch(value)
	if len(matches) != 2 {
		return "", false
	}
	return matches[1], true
}

// ParseJSONPointerRef extracts the token path from a JSON pointer reference.
// Returns the path and true if valid, empty string and false otherwise.
func ParseJSONPointerRef(ref string) (string, bool) {
	matches := jsonPointerPattern.FindStringSubmatch(ref)
	if len(matches) != 2 {
		return "", false
	}
	// Convert /path/to/token to path.to.token
	parts := strings.Split(matches[1], "/")
	// Decode RFC 6901 escape sequences in each segment
	// Order matters: ~1 must be replaced before ~0
	for i, part := range parts {
		part = strings.ReplaceAll(part, "~1", "/")
		part = strings.ReplaceAll(part, "~0", "~")
		parts[i] = part
	}
	return strings.Join(parts, "."), true
}

// IsCurlyBraceRef returns true if the value contains a curly brace reference.
func IsCurlyBraceRef(value string) bool {
	return curlyBracePattern.MatchString(value)
}

// IsJSONPointerRef returns true if the value is a JSON pointer reference.
func IsJSONPointerRef(ref string) bool {
	return jsonPointerPattern.MatchString(ref)
}

// ExtractAllRefs extracts all curly brace references from a string.
func ExtractAllRefs(value string) []string {
	matches := curlyBracePattern.FindAllStringSubmatch(value, -1)
	refs := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) >= 2 {
			refs = append(refs, m[1])
		}
	}
	return refs
}
