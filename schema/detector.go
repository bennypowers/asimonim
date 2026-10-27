/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

package schema

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/tidwall/jsonc"
	"gopkg.in/yaml.v3"
)

// utf8BOM is the UTF-8 byte order mark.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// NormalizeContent strips a UTF-8 BOM and JSONC comments from content that
// begins as a JSON object or array. Content that is not valid JSON after
// comment removal, including YAML flow mappings, is returned unchanged.
// Callers that parse content with a YAML decoder should normalize first,
// because that decoder cannot parse JSONC comments.
func NormalizeContent(content []byte) []byte {
	content = bytes.TrimPrefix(content, utf8BOM)
	if json.Valid(content) {
		return content
	}
	// A document that YAML decodes to a map or sequence is a YAML document,
	// including flow mappings that start with '{' or '['. Do not rewrite it
	// as JSONC: in YAML, "//" is ordinary text, not a comment.
	var probe any
	if err := yaml.Unmarshal(content, &probe); err == nil {
		switch probe.(type) {
		case map[string]any, map[any]any, []any:
			return content
		}
	}
	if !looksLikeJSON(content) {
		return content
	}
	normalized := jsonc.ToJSON(content)
	if !json.Valid(normalized) {
		return content
	}
	return normalized
}

// looksLikeJSON reports whether content begins with a JSON object or array,
// skipping leading whitespace and JSONC comments.
func looksLikeJSON(content []byte) bool {
	for i := 0; i < len(content); {
		switch content[i] {
		case ' ', '\t', '\n', '\r':
			i++
		case '/':
			switch {
			case i+1 < len(content) && content[i+1] == '/':
				for i < len(content) && content[i] != '\n' {
					i++
				}
			case i+1 < len(content) && content[i+1] == '*':
				end := bytes.Index(content[i+2:], []byte("*/"))
				if end < 0 {
					return false
				}
				i += 2 + end + 2
			default:
				return false
			}
		case '{', '[':
			return true
		default:
			return false
		}
	}
	return false
}

// DetectionConfig provides configuration for schema version detection.
type DetectionConfig struct {
	// DefaultVersion is used when no other detection method succeeds.
	DefaultVersion Version
}

// DetectVersion detects the schema version from file content.
// Priority order:
// 1. $schema field in file root
// 2. Config default version
// 3. Duck typing (detect reserved fields/structured formats)
// 4. Default to draft (backward compatibility)
func DetectVersion(content []byte, config *DetectionConfig) (Version, error) {
	content = NormalizeContent(content)

	var data map[string]any
	if err := yaml.Unmarshal(content, &data); err != nil {
		return Unknown, fmt.Errorf("invalid YAML/JSON: %w", err)
	}

	// 1. Check for explicit $schema field
	if schemaURL, ok := data["$schema"].(string); ok {
		version, err := FromURL(schemaURL)
		if err == nil {
			return version, nil
		}
	}

	// 2. Check config default
	if config != nil && config.DefaultVersion != Unknown {
		return config.DefaultVersion, nil
	}

	// 3. Duck typing - check for unambiguous 2025.10 features
	if version := duckTypeSchema(data); version != Unknown {
		return version, nil
	}

	// 4. Default to draft for backward compatibility
	return Draft, nil
}

// duckTypeSchema attempts to detect schema version from content patterns.
func duckTypeSchema(data map[string]any) Version {
	if hasFeature(data, "$ref") {
		return V2025_10
	}
	if hasFeature(data, "$extends") {
		return V2025_10
	}
	if hasFeature(data, "resolutionOrder") {
		return V2025_10
	}
	if hasStructuredColorObjects(data) {
		return V2025_10
	}
	return Unknown
}

// hasFeature checks if a feature (field name) exists anywhere in the structure.
func hasFeature(data map[string]any, featureName string) bool {
	if _, exists := data[featureName]; exists {
		return true
	}
	for _, value := range data {
		switch v := value.(type) {
		case map[string]any:
			if hasFeature(v, featureName) {
				return true
			}
		case []any:
			if hasFeatureInSlice(v, featureName) {
				return true
			}
		}
	}
	return false
}

// hasFeatureInSlice recursively checks for a feature in slice elements.
func hasFeatureInSlice(arr []any, featureName string) bool {
	for _, elem := range arr {
		switch v := elem.(type) {
		case map[string]any:
			if hasFeature(v, featureName) {
				return true
			}
		case []any:
			if hasFeatureInSlice(v, featureName) {
				return true
			}
		}
	}
	return false
}

// hasStructuredColorObjects checks for 2025.10-style structured color values.
func hasStructuredColorObjects(data map[string]any) bool {
	return checkForStructuredColors(data)
}

func checkForStructuredColors(obj any) bool {
	switch v := obj.(type) {
	case map[string]any:
		// Check if this node has a structured $value with colorSpace
		if value, ok := v["$value"].(map[string]any); ok {
			if _, hasColorSpace := value["colorSpace"]; hasColorSpace {
				return true
			}
		}
		for _, child := range v {
			if checkForStructuredColors(child) {
				return true
			}
		}
	case []any:
		for _, elem := range v {
			if checkForStructuredColors(elem) {
				return true
			}
		}
	}
	return false
}
