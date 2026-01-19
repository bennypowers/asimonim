/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package schema_test

import (
	"testing"

	"bennypowers.dev/asimonim/schema"
)

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
