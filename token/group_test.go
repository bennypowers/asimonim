/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package token_test

import (
	"sort"
	"testing"

	"bennypowers.dev/asimonim/token"
)

func TestNewGroup(t *testing.T) {
	g := token.NewGroup("colors")
	if g.Name != "colors" {
		t.Errorf("NewGroup name = %q, want %q", g.Name, "colors")
	}
	if g.Tokens == nil {
		t.Error("NewGroup Tokens map is nil, want initialized")
	}
	if g.Groups == nil {
		t.Error("NewGroup Groups map is nil, want initialized")
	}
	if len(g.Tokens) != 0 {
		t.Errorf("NewGroup Tokens len = %d, want 0", len(g.Tokens))
	}
	if len(g.Groups) != 0 {
		t.Errorf("NewGroup Groups len = %d, want 0", len(g.Groups))
	}
}

func TestGroup_AllTokens_Empty(t *testing.T) {
	g := token.NewGroup("empty")
	tokens := g.AllTokens()
	if len(tokens) != 0 {
		t.Errorf("AllTokens() on empty group returned %d tokens, want 0", len(tokens))
	}
}

func TestGroup_AllTokens_FlatTokens(t *testing.T) {
	g := token.NewGroup("colors")
	g.Tokens["primary"] = &token.Token{Name: "primary"}
	g.Tokens["secondary"] = &token.Token{Name: "secondary"}

	tokens := g.AllTokens()
	if len(tokens) != 2 {
		t.Fatalf("AllTokens() returned %d tokens, want 2", len(tokens))
	}

	names := []string{tokens[0].Name, tokens[1].Name}
	sort.Strings(names)
	if names[0] != "primary" || names[1] != "secondary" {
		t.Errorf("AllTokens() names = %v, want [primary, secondary]", names)
	}
}

func TestGroup_AllTokens_Nested(t *testing.T) {
	root := token.NewGroup("root")
	root.Tokens["root-token"] = &token.Token{Name: "root-token"}

	child := token.NewGroup("brand")
	child.Tokens["primary"] = &token.Token{Name: "primary"}
	child.Tokens["secondary"] = &token.Token{Name: "secondary"}

	grandchild := token.NewGroup("accent")
	grandchild.Tokens["highlight"] = &token.Token{Name: "highlight"}

	child.Groups["accent"] = grandchild
	root.Groups["brand"] = child

	tokens := root.AllTokens()
	// root-token + primary + secondary + highlight = 4
	if len(tokens) != 4 {
		t.Fatalf("AllTokens() returned %d tokens, want 4", len(tokens))
	}

	names := make([]string, len(tokens))
	for i, tok := range tokens {
		names[i] = tok.Name
	}
	sort.Strings(names)
	expected := []string{"highlight", "primary", "root-token", "secondary"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("AllTokens() names[%d] = %q, want %q", i, name, expected[i])
		}
	}
}
