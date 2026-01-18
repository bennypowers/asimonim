/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package resolver_test

import (
	"testing"

	"bennypowers.dev/asimonim/resolver"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/token"
)

func TestDependencyGraph_NoCycle(t *testing.T) {
	tokens := []*token.Token{
		{Name: "a", Value: "1"},
		{Name: "b", Value: "{a}"},
		{Name: "c", Value: "{b}"},
	}

	graph := resolver.BuildDependencyGraph(tokens)

	if graph.HasCycle() {
		t.Error("expected no cycle")
	}
}

func TestDependencyGraph_Cycle(t *testing.T) {
	tokens := []*token.Token{
		{Name: "a", Value: "{c}"},
		{Name: "b", Value: "{a}"},
		{Name: "c", Value: "{b}"},
	}

	graph := resolver.BuildDependencyGraph(tokens)

	if !graph.HasCycle() {
		t.Error("expected cycle")
	}

	cycle := graph.FindCycle()
	if cycle == nil {
		t.Error("expected to find cycle path")
	}
}

func TestResolveAliases(t *testing.T) {
	tokens := []*token.Token{
		{Name: "base", Value: "#FF6B35"},
		{Name: "primary", Value: "{base}"},
	}

	err := resolver.ResolveAliases(tokens, schema.Draft)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tokens[0].ResolvedValue != "#FF6B35" {
		t.Errorf("expected base to resolve to #FF6B35, got %v", tokens[0].ResolvedValue)
	}

	if tokens[1].ResolvedValue != "#FF6B35" {
		t.Errorf("expected primary to resolve to #FF6B35, got %v", tokens[1].ResolvedValue)
	}
}
