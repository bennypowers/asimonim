/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package search

import (
	"regexp"
	"testing"

	"bennypowers.dev/asimonim/token"
)

func TestMatchString(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		query    string
		pattern  *regexp.Regexp
		expected bool
	}{
		{"simple match", "color-primary", "primary", nil, true},
		{"case insensitive", "Color-Primary", "primary", nil, true},
		{"no match", "color-primary", "spacing", nil, false},
		{"partial match", "color-primary-dark", "primary", nil, true},
		{"empty query", "color-primary", "", nil, true},
		{"empty string", "", "query", nil, false},
		{"regex match", "color-primary", "", regexp.MustCompile(`^color-`), true},
		{"regex no match", "spacing-small", "", regexp.MustCompile(`^color-`), false},
		{"regex pattern", "token-123", "", regexp.MustCompile(`\d+`), true},
		{"regex case sensitive", "Color", "", regexp.MustCompile(`color`), false},
		{"regex case insensitive", "Color", "", regexp.MustCompile(`(?i)color`), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchString(tt.s, tt.query, tt.pattern)
			if got != tt.expected {
				t.Errorf("matchString(%q, %q, pattern) = %v, want %v", tt.s, tt.query, got, tt.expected)
			}
		})
	}
}

func TestFilterTokens(t *testing.T) {
	tokens := []*token.Token{
		{Name: "color-primary", Type: "color", Path: []string{"color", "primary"}, Deprecated: false},
		{Name: "color-secondary", Type: "color", Path: []string{"color", "secondary"}, Deprecated: true},
		{Name: "spacing-small", Type: "dimension", Path: []string{"spacing", "small"}, Deprecated: false},
		{Name: "spacing-large", Type: "dimension", Path: []string{"spacing", "large"}, Deprecated: false},
		{Name: "font-body", Type: "fontFamily", Path: []string{"font", "body"}, Deprecated: true},
	}

	t.Run("no filters", func(t *testing.T) {
		result := filterTokens(tokens, "", "", false, false)
		if len(result) != 5 {
			t.Errorf("expected 5 tokens, got %d", len(result))
		}
	})

	t.Run("filter by type", func(t *testing.T) {
		result := filterTokens(tokens, "color", "", false, false)
		if len(result) != 2 {
			t.Errorf("expected 2 color tokens, got %d", len(result))
		}
	})

	t.Run("filter by group", func(t *testing.T) {
		result := filterTokens(tokens, "", "color", false, false)
		if len(result) != 2 {
			t.Errorf("expected 2 tokens in color group, got %d", len(result))
		}
	})

	t.Run("deprecated only", func(t *testing.T) {
		result := filterTokens(tokens, "", "", true, false)
		if len(result) != 2 {
			t.Errorf("expected 2 deprecated tokens, got %d", len(result))
		}
	})

	t.Run("hide deprecated", func(t *testing.T) {
		result := filterTokens(tokens, "", "", false, true)
		if len(result) != 3 {
			t.Errorf("expected 3 non-deprecated tokens, got %d", len(result))
		}
	})
}
