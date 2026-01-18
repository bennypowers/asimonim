/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package schema

import (
	"encoding/json"
	"fmt"
)

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
	var data map[string]any
	if err := json.Unmarshal(content, &data); err != nil {
		return Unknown, fmt.Errorf("invalid JSON: %w", err)
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
		if obj, ok := value.(map[string]any); ok {
			if hasFeature(obj, featureName) {
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
		if colorType, ok := v["$type"].(string); ok && colorType == "color" {
			if value, ok := v["$value"].(map[string]any); ok {
				if _, hasColorSpace := value["colorSpace"]; hasColorSpace {
					return true
				}
			}
		}
		for _, child := range v {
			if checkForStructuredColors(child) {
				return true
			}
		}
	}
	return false
}

// hasStringColorValues checks for draft-style string color values.
func hasStringColorValues(data map[string]any) bool {
	return checkForStringColors(data)
}

func checkForStringColors(obj any) bool {
	switch v := obj.(type) {
	case map[string]any:
		if colorType, ok := v["$type"].(string); ok && colorType == "color" {
			if value, ok := v["$value"].(string); ok && value != "" {
				return true
			}
		}
		for _, child := range v {
			if checkForStringColors(child) {
				return true
			}
		}
	}
	return false
}
