/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package list

import (
	"testing"

	"bennypowers.dev/asimonim/token"
)

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
		for _, tok := range result {
			if tok.Type != "color" {
				t.Errorf("expected type color, got %s", tok.Type)
			}
		}
	})

	t.Run("filter by group", func(t *testing.T) {
		result := filterTokens(tokens, "", "spacing", false, false)
		if len(result) != 2 {
			t.Errorf("expected 2 spacing tokens, got %d", len(result))
		}
		for _, tok := range result {
			if tok.Path[0] != "spacing" {
				t.Errorf("expected path starting with spacing, got %v", tok.Path)
			}
		}
	})

	t.Run("filter deprecated only", func(t *testing.T) {
		result := filterTokens(tokens, "", "", true, false)
		if len(result) != 2 {
			t.Errorf("expected 2 deprecated tokens, got %d", len(result))
		}
		for _, tok := range result {
			if !tok.Deprecated {
				t.Errorf("expected deprecated token, got non-deprecated %s", tok.Name)
			}
		}
	})

	t.Run("hide deprecated", func(t *testing.T) {
		result := filterTokens(tokens, "", "", false, true)
		if len(result) != 3 {
			t.Errorf("expected 3 non-deprecated tokens, got %d", len(result))
		}
		for _, tok := range result {
			if tok.Deprecated {
				t.Errorf("expected non-deprecated token, got deprecated %s", tok.Name)
			}
		}
	})

	t.Run("combined filters", func(t *testing.T) {
		result := filterTokens(tokens, "color", "", false, true)
		if len(result) != 1 {
			t.Errorf("expected 1 non-deprecated color token, got %d", len(result))
		}
		if result[0].Name != "color-primary" {
			t.Errorf("expected color-primary, got %s", result[0].Name)
		}
	})

	t.Run("type and group filter", func(t *testing.T) {
		result := filterTokens(tokens, "dimension", "spacing", false, false)
		if len(result) != 2 {
			t.Errorf("expected 2 dimension tokens in spacing group, got %d", len(result))
		}
	})

	t.Run("no matches", func(t *testing.T) {
		result := filterTokens(tokens, "shadow", "", false, false)
		if len(result) != 0 {
			t.Errorf("expected 0 tokens, got %d", len(result))
		}
	})
}
