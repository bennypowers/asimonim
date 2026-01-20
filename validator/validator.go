/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package validator provides schema consistency validation for DTCG token files.
package validator

import (
	"fmt"
	"strings"

	"bennypowers.dev/asimonim/schema"
	"gopkg.in/yaml.v3"
)

// ValidationError represents a schema consistency error.
type ValidationError struct {
	// FilePath is the path to the file containing the error.
	FilePath string
	// Path is the JSON path to the problematic element.
	Path string
	// Message describes what's wrong.
	Message string
	// Suggestion provides an actionable fix.
	Suggestion string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	var sb strings.Builder
	if e.FilePath != "" {
		sb.WriteString(e.FilePath)
		sb.WriteString(": ")
	}
	if e.Path != "" {
		sb.WriteString(e.Path)
		sb.WriteString(": ")
	}
	sb.WriteString(e.Message)
	if e.Suggestion != "" {
		sb.WriteString(" (")
		sb.WriteString(e.Suggestion)
		sb.WriteString(")")
	}
	return sb.String()
}

// ValidateConsistency checks that file content matches the expected schema version.
// Returns errors for:
// - Mixed schema features (e.g., $ref in draft file)
// - Color format mismatch (structured colors in draft, string colors in 2025.10)
// - Conflicting root token patterns ($root + group markers like "_" in same group)
// - 2025.10 file using draft group markers instead of $root
func ValidateConsistency(content []byte, version schema.Version) []ValidationError {
	return ValidateConsistencyWithPath(content, version, "")
}

// ValidateConsistencyWithPath validates content and includes file path in errors.
func ValidateConsistencyWithPath(content []byte, version schema.Version, filePath string) []ValidationError {
	var data map[string]any
	if err := yaml.Unmarshal(content, &data); err != nil {
		return []ValidationError{{
			FilePath: filePath,
			Message:  fmt.Sprintf("failed to parse content: %v", err),
		}}
	}

	var errors []ValidationError

	switch version {
	case schema.Draft:
		errors = validateDraft(data, filePath, nil)
	case schema.V2025_10:
		errors = validateV2025(data, filePath, nil)
	}

	return errors
}

// validateDraft checks for 2025.10 features that shouldn't appear in draft schema.
func validateDraft(data map[string]any, filePath string, path []string) []ValidationError {
	var errors []ValidationError

	for key, value := range data {
		currentPath := append(path[:len(path):len(path)], key)
		pathStr := strings.Join(currentPath, ".")

		// Check for $ref (2025.10 feature)
		if key == "$ref" {
			errors = append(errors, ValidationError{
				FilePath:   filePath,
				Path:       pathStr,
				Message:    "$ref is not valid in draft schema",
				Suggestion: "use curly-brace references like {token.path} or update to 2025.10 schema",
			})
			continue
		}

		// Check for $extends (2025.10 feature)
		if key == "$extends" {
			errors = append(errors, ValidationError{
				FilePath:   filePath,
				Path:       pathStr,
				Message:    "$extends is not valid in draft schema",
				Suggestion: "update $schema to 2025.10 to use group extensions",
			})
			continue
		}

		// Check for $root (2025.10 feature)
		if key == "$root" {
			errors = append(errors, ValidationError{
				FilePath:   filePath,
				Path:       pathStr,
				Message:    "$root is not valid in draft schema",
				Suggestion: "use group markers like \"_\" or update to 2025.10 schema",
			})
			continue
		}

		valueMap, ok := value.(map[string]any)
		if !ok {
			continue
		}

		// Check for structured color values in draft (only for color type tokens)
		if isColorToken(valueMap, path) {
			if rawValue, hasValue := valueMap["$value"]; hasValue {
				if colorMap, isMap := rawValue.(map[string]any); isMap {
					if _, hasColorSpace := colorMap["colorSpace"]; hasColorSpace {
						errors = append(errors, ValidationError{
							FilePath:   filePath,
							Path:       pathStr,
							Message:    "structured color values are not valid in draft schema",
							Suggestion: "use string color format like \"#RRGGBB\" or update $schema to 2025.10",
						})
					}
				}
			}
		}

		// Recurse into nested objects
		childErrors := validateDraft(valueMap, filePath, currentPath)
		errors = append(errors, childErrors...)
	}

	return errors
}

// validateV2025 checks for draft patterns that shouldn't appear in 2025.10 schema.
func validateV2025(data map[string]any, filePath string, path []string) []ValidationError {
	var errors []ValidationError

	// Track root token patterns in this group
	hasRootToken := false
	hasGroupMarker := false
	groupMarkerPath := ""

	for key, value := range data {
		currentPath := append(path[:len(path):len(path)], key)
		pathStr := strings.Join(currentPath, ".")

		// Skip schema field
		if key == "$schema" {
			continue
		}

		// Check for $root
		if key == "$root" {
			hasRootToken = true
		}

		// Check for group markers (draft pattern)
		if isGroupMarker(key) {
			hasGroupMarker = true
			groupMarkerPath = pathStr
		}

		valueMap, ok := value.(map[string]any)
		if !ok {
			continue
		}

		// Check for string color values in 2025.10 (only for color type tokens)
		if isColorToken(valueMap, path) {
			if rawValue, hasValue := valueMap["$value"]; hasValue {
				if colorStr, isString := rawValue.(string); isString {
					// String colors are not valid in 2025.10
					errors = append(errors, ValidationError{
						FilePath:   filePath,
						Path:       pathStr,
						Message:    fmt.Sprintf("string color value %q is not valid in 2025.10 schema", colorStr),
						Suggestion: "use structured color format with colorSpace and components",
					})
				}
			}
		}

		// Recurse into nested objects
		childErrors := validateV2025(valueMap, filePath, currentPath)
		errors = append(errors, childErrors...)
	}

	// Check for conflicting root patterns (both $root and group marker in same group)
	if hasRootToken && hasGroupMarker {
		errors = append(errors, ValidationError{
			FilePath:   filePath,
			Path:       strings.Join(path, "."),
			Message:    "conflicting root token patterns: both $root and group marker found",
			Suggestion: "use only $root in 2025.10 schema, remove group markers like \"_\"",
		})
	} else if hasGroupMarker && !hasRootToken {
		// Group marker without $root in 2025.10
		errors = append(errors, ValidationError{
			FilePath:   filePath,
			Path:       groupMarkerPath,
			Message:    "group marker tokens are deprecated in 2025.10 schema",
			Suggestion: "use $root instead of group markers like \"_\"",
		})
	}

	return errors
}

// isColorToken checks if a value map represents a color token.
func isColorToken(valueMap map[string]any, parentPath []string) bool {
	// Check for explicit $type: color
	if tokenType, ok := valueMap["$type"].(string); ok {
		return tokenType == "color"
	}

	// Check if parent group has $type: color (type inheritance)
	// This is a simplified check - in practice we'd need the full context
	for i := len(parentPath) - 1; i >= 0; i-- {
		if parentPath[i] == "color" || parentPath[i] == "colors" {
			return true
		}
	}

	return false
}

// isGroupMarker checks if a key is a draft-style group marker.
func isGroupMarker(key string) bool {
	// Common group markers in draft schema
	return key == "_" || key == "-" || key == "."
}
