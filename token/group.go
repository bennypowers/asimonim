/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package token

// Group represents a group of tokens (can be nested).
type Group struct {
	// Name is the group's identifier.
	Name string `json:"-"`

	// Description is optional documentation for the group.
	Description string `json:"$description,omitempty"`

	// Type is the inherited $type for tokens in this group.
	Type string `json:"$type,omitempty"`

	// Extends is a JSON pointer to another group (2025.10 only).
	Extends string `json:"$extends,omitempty"`

	// Tokens contains the tokens in this group.
	Tokens map[string]*Token `json:"-"`

	// Groups contains nested groups.
	Groups map[string]*Group `json:"-"`

	// Line is the 0-based line number where this group is defined.
	Line uint32 `json:"-"`

	// Character is the 0-based character offset where this group is defined.
	Character uint32 `json:"-"`
}

// NewGroup creates a new empty token group.
func NewGroup(name string) *Group {
	return &Group{
		Name:   name,
		Tokens: make(map[string]*Token),
		Groups: make(map[string]*Group),
	}
}

// AllTokens returns all tokens in this group and nested groups.
func (g *Group) AllTokens() []*Token {
	var tokens []*Token
	for _, t := range g.Tokens {
		tokens = append(tokens, t)
	}
	for _, nested := range g.Groups {
		tokens = append(tokens, nested.AllTokens()...)
	}
	return tokens
}
