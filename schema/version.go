/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package schema provides DTCG schema version handling.
package schema

import "fmt"

// Version represents a design tokens schema version.
type Version int

const (
	// Unknown represents an undetected or unrecognized schema version.
	Unknown Version = iota

	// Draft represents the editor's draft schema.
	Draft

	// V2025_10 represents the stable 2025.10 schema.
	V2025_10
)

// String returns the string representation of the schema version.
func (v Version) String() string {
	switch v {
	case Draft:
		return "draft"
	case V2025_10:
		return "v2025.10"
	default:
		return "unknown"
	}
}

// URL returns the JSON Schema URL for this version.
func (v Version) URL() string {
	switch v {
	case Draft:
		return "https://www.designtokens.org/schemas/draft.json"
	case V2025_10:
		return "https://www.designtokens.org/schemas/2025.10.json"
	default:
		return ""
	}
}

// FromURL returns the schema version from a JSON Schema URL.
func FromURL(url string) (Version, error) {
	switch url {
	case "https://www.designtokens.org/schemas/draft.json":
		return Draft, nil
	case "https://www.designtokens.org/schemas/2025.10.json":
		return V2025_10, nil
	default:
		return Unknown, fmt.Errorf("unrecognized schema URL: %s", url)
	}
}

// FromString returns the schema version from a string representation.
func FromString(s string) (Version, error) {
	switch s {
	case "draft":
		return Draft, nil
	case "v2025.10", "v2025_10", "2025.10", "2025", "v2025":
		return V2025_10, nil
	default:
		return Unknown, fmt.Errorf("unrecognized schema version string: %s", s)
	}
}
