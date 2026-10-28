/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

package schema_test

import (
	"encoding/json"
	"strings"
	"testing"

	"bennypowers.dev/asimonim/schema"
)

func TestDetectVersion_HasFeatureInSlice(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected schema.Version
	}{
		{
			name: "$ref inside array element",
			content: `{
				"group": {
					"$value": [
						{"$ref": "#/other/token"}
					]
				}
			}`,
			expected: schema.V2025_10,
		},
		{
			name: "$extends inside nested array",
			content: `{
				"tokens": {
					"list": [
						[{"$extends": "#/base"}]
					]
				}
			}`,
			expected: schema.V2025_10,
		},
		{
			name: "array with no features",
			content: `{
				"group": {
					"$value": [1, 2, 3]
				}
			}`,
			expected: schema.Draft,
		},
		{
			name: "array with string elements only",
			content: `{
				"tags": ["color", "brand"]
			}`,
			expected: schema.Draft,
		},
		{
			name: "deeply nested array with $ref",
			content: `{
				"a": {
					"b": {
						"$value": [
							{"nested": [{"$ref": "#/x"}]}
						]
					}
				}
			}`,
			expected: schema.V2025_10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := schema.DetectVersion([]byte(tt.content), nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("DetectVersion() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDetectVersion_StructuredColors(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected schema.Version
	}{
		{
			name: "structured color with colorSpace",
			content: `{
				"brand": {
					"primary": {
						"$type": "color",
						"$value": {"colorSpace": "srgb", "channels": [1, 0, 0]}
					}
				}
			}`,
			expected: schema.V2025_10,
		},
		{
			name: "deeply nested structured color",
			content: `{
				"theme": {
					"dark": {
						"surface": {
							"$type": "color",
							"$value": {"colorSpace": "oklch", "channels": [0.5, 0.1, 200]}
						}
					}
				}
			}`,
			expected: schema.V2025_10,
		},
		{
			name: "structured color in array context",
			content: `{
				"palette": {
					"$value": [
						{"colorSpace": "srgb", "channels": [1, 0, 0]}
					]
				}
			}`,
			// The $value is an array, not a map with colorSpace, so no structured color detection
			expected: schema.Draft,
		},
		{
			name: "string color value (not structured)",
			content: `{
				"color": {
					"$type": "color",
					"$value": "#ff0000"
				}
			}`,
			expected: schema.Draft,
		},
		{
			name: "map $value without colorSpace",
			content: `{
				"spacing": {
					"$type": "dimension",
					"$value": {"value": 4, "unit": "px"}
				}
			}`,
			expected: schema.Draft,
		},
		{
			name: "colorSpace at wrong nesting level",
			content: `{
				"color": {
					"colorSpace": "srgb",
					"$value": "#ff0000"
				}
			}`,
			// colorSpace is a sibling of $value, not inside $value
			expected: schema.Draft,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := schema.DetectVersion([]byte(tt.content), nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("DetectVersion() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNormalizeJSON(t *testing.T) {
	t.Run("trailing comment and comma are normalized", func(t *testing.T) {
		input := []byte(`{"name":"p","asimonim":{"prefix":"rh"}, // note
}`)
		got := schema.NormalizeJSON(input)
		if !json.Valid(got) {
			t.Errorf("NormalizeJSON() = %q, want valid JSON", got)
		}
	})

	t.Run("leading comment is removed", func(t *testing.T) {
		input := []byte("// header\n{\"a\": 1}")
		got := schema.NormalizeJSON(input)
		if !json.Valid(got) || strings.Contains(string(got), "//") {
			t.Errorf("NormalizeJSON() = %q, want valid JSON without comments", got)
		}
	})
}

func TestDetectVersion(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		config   *schema.DetectionConfig
		expected schema.Version
		wantErr  bool
	}{
		{
			name:     "explicit draft schema",
			content:  `{"$schema": "https://www.designtokens.org/schemas/draft.json"}`,
			expected: schema.Draft,
		},
		{
			name:     "explicit v2025.10 schema",
			content:  `{"$schema": "https://www.designtokens.org/schemas/2025.10.json"}`,
			expected: schema.V2025_10,
		},
		{
			name:    "config default version",
			content: `{"color": {"$value": "#fff"}}`,
			config: &schema.DetectionConfig{
				DefaultVersion: schema.V2025_10,
			},
			expected: schema.V2025_10,
		},
		{
			name:     "duck type $ref",
			content:  `{"color": {"$value": {"$ref": "#/other/color"}}}`,
			expected: schema.V2025_10,
		},
		{
			name:     "duck type $extends",
			content:  `{"color": {"$extends": "#/base/color", "$value": "#fff"}}`,
			expected: schema.V2025_10,
		},
		{
			name:     "duck type resolutionOrder",
			content:  `{"resolutionOrder": ["a", "b"]}`,
			expected: schema.V2025_10,
		},
		{
			name: "duck type structured color",
			content: `{
				"color": {
					"$type": "color",
					"$value": {"colorSpace": "srgb", "channels": [1, 0, 0]}
				}
			}`,
			expected: schema.V2025_10,
		},
		{
			name:     "fallback to draft",
			content:  `{"color": {"$value": "#fff"}}`,
			expected: schema.Draft,
		},
		{
			name:     "nested $ref detection",
			content:  `{"group": {"nested": {"$value": {"$ref": "#/other"}}}}`,
			expected: schema.V2025_10,
		},
		{
			name:    "invalid JSON",
			content: `{invalid json`,
			wantErr: true,
		},
		{
			name:     "JSONC with line and block comments",
			content:  "// leading comment\n{\n  \"color\": {\"$value\": {\"$ref\": \"#/other\"}} /* trailing */\n}\n",
			expected: schema.V2025_10,
		},
		{
			name:     "JSONC with inline block comment after $schema",
			content:  `{"$schema": /* DTCG 2025.10 */ "https://www.designtokens.org/schemas/2025.10.json"}`,
			expected: schema.V2025_10,
		},
		{
			name:     "JSONC with UTF-8 BOM",
			content:  "\xEF\xBB\xBF// leading comment\n{\"color\": {\"$value\": {\"$ref\": \"#/other\"}}}",
			expected: schema.V2025_10,
		},
		{
			name:     "YAML flow mapping",
			content:  `{color: {"$value": "#fff"}}`,
			expected: schema.Draft,
		},
		{
			name:     "YAML format",
			content:  "color:\n  $value: '#fff'\n",
			expected: schema.Draft,
		},
		{
			name:     "YAML with $ref",
			content:  "color:\n  $value:\n    $ref: '#/other'\n",
			expected: schema.V2025_10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := schema.DetectVersion([]byte(tt.content), tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("DetectVersion() = %v, want %v", got, tt.expected)
			}
		})
	}
}
