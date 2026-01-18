/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package common

import (
	"strings"

	"bennypowers.dev/asimonim/schema"
)

// ReferenceType indicates the type of reference.
type ReferenceType int

const (
	// CurlyBraceReference is a {token.path} style reference (both schemas).
	CurlyBraceReference ReferenceType = iota

	// JSONPointerReference is a $ref field (2025.10 only).
	JSONPointerReference
)

// Reference represents a reference to another token.
type Reference struct {
	Type   ReferenceType
	Path   string
	Line   int
	Column int
}

// ExtractReferences extracts references from a string value.
func ExtractReferences(content string, version schema.Version) ([]Reference, error) {
	var refs []Reference

	// Extract curly brace references (supported in both schemas)
	matches := CurlyBraceRefPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			refs = append(refs, Reference{
				Type: CurlyBraceReference,
				Path: match[1],
			})
		}
	}

	return refs, nil
}

// ExtractReferencesFromValue extracts references from any value type.
// Handles both string interpolation and $ref fields.
func ExtractReferencesFromValue(value any, version schema.Version) ([]Reference, error) {
	switch v := value.(type) {
	case string:
		return ExtractReferences(v, version)

	case map[string]any:
		if refPath, ok := v["$ref"].(string); ok {
			if version == schema.Draft {
				return nil, schema.ErrInvalidReference
			}
			path := strings.TrimPrefix(refPath, "#/")
			return []Reference{
				{
					Type: JSONPointerReference,
					Path: path,
				},
			}, nil
		}
		return nil, nil

	default:
		return nil, nil
	}
}

// ConvertJSONPointerToTokenPath converts a JSON Pointer path to a token path.
// Examples:
//
//	"#/color/brand/primary" -> "color.brand.primary"
//	"color/brand/primary" -> "color.brand.primary"
func ConvertJSONPointerToTokenPath(jsonPointer string) string {
	jsonPointer = strings.TrimPrefix(jsonPointer, "#/")
	return strings.ReplaceAll(jsonPointer, "/", ".")
}

// ConvertTokenPathToJSONPointer converts a token path to a JSON Pointer.
// Example: "color.brand.primary" -> "#/color/brand/primary"
func ConvertTokenPathToJSONPointer(tokenPath string) string {
	return "#/" + strings.ReplaceAll(tokenPath, ".", "/")
}
