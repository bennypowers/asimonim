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

func TestVersion_String(t *testing.T) {
	tests := []struct {
		version  schema.Version
		expected string
	}{
		{schema.Unknown, "unknown"},
		{schema.Draft, "draft"},
		{schema.V2025_10, "v2025.10"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.version.String(); got != tt.expected {
				t.Errorf("Version.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestVersion_URL(t *testing.T) {
	tests := []struct {
		version  schema.Version
		expected string
	}{
		{schema.Unknown, ""},
		{schema.Draft, "https://www.designtokens.org/schemas/draft.json"},
		{schema.V2025_10, "https://www.designtokens.org/schemas/2025.10.json"},
	}

	for _, tt := range tests {
		t.Run(tt.version.String(), func(t *testing.T) {
			if got := tt.version.URL(); got != tt.expected {
				t.Errorf("Version.URL() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFromURL(t *testing.T) {
	tests := []struct {
		url      string
		expected schema.Version
		wantErr  bool
	}{
		{"https://www.designtokens.org/schemas/draft.json", schema.Draft, false},
		{"https://www.designtokens.org/schemas/2025.10.json", schema.V2025_10, false},
		{"https://example.com/unknown.json", schema.Unknown, true},
		{"", schema.Unknown, true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got, err := schema.FromURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("FromURL() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected schema.Version
		wantErr  bool
	}{
		{"", schema.Unknown, false},
		{"unknown", schema.Unknown, false},
		{"draft", schema.Draft, false},
		{"v2025.10", schema.V2025_10, false},
		{"v2025_10", schema.V2025_10, false},
		{"2025.10", schema.V2025_10, false},
		{"2025", schema.V2025_10, false},
		{"v2025", schema.V2025_10, false},
		{"invalid", schema.Unknown, true},
		{"v1.0", schema.Unknown, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := schema.FromString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromString(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("FromString(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
