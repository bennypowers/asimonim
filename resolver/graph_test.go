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

func TestDependencyGraph_Dependencies(t *testing.T) {
	tokens := []*token.Token{
		{Name: "a", Value: "1"},
		{Name: "b", Value: "{a}"},
		{Name: "c", Value: "{a} and {b}"},
	}

	graph := resolver.BuildDependencyGraph(tokens)

	// "a" has no dependencies
	aDeps := graph.Dependencies("a")
	if len(aDeps) != 0 {
		t.Errorf("expected 0 dependencies for 'a', got %d", len(aDeps))
	}

	// "b" depends on "a"
	bDeps := graph.Dependencies("b")
	if len(bDeps) != 1 || bDeps[0] != "a" {
		t.Errorf("expected [a] for 'b' dependencies, got %v", bDeps)
	}

	// "c" depends on "a" and "b"
	cDeps := graph.Dependencies("c")
	if len(cDeps) != 2 {
		t.Errorf("expected 2 dependencies for 'c', got %d", len(cDeps))
	}

	// non-existent node
	xDeps := graph.Dependencies("nonexistent")
	if len(xDeps) != 0 {
		t.Errorf("expected 0 dependencies for nonexistent, got %d", len(xDeps))
	}
}

func TestDependencyGraph_Dependents(t *testing.T) {
	tokens := []*token.Token{
		{Name: "a", Value: "1"},
		{Name: "b", Value: "{a}"},
		{Name: "c", Value: "{a}"},
	}

	graph := resolver.BuildDependencyGraph(tokens)

	// "a" is depended on by "b" and "c"
	aDependents := graph.Dependents("a")
	if len(aDependents) != 2 {
		t.Errorf("expected 2 dependents for 'a', got %d", len(aDependents))
	}

	// "b" has no dependents
	bDependents := graph.Dependents("b")
	if len(bDependents) != 0 {
		t.Errorf("expected 0 dependents for 'b', got %d", len(bDependents))
	}

	// non-existent node
	xDependents := graph.Dependents("nonexistent")
	if len(xDependents) != 0 {
		t.Errorf("expected 0 dependents for nonexistent, got %d", len(xDependents))
	}
}

func TestDependencyGraph_TopologicalSort(t *testing.T) {
	tokens := []*token.Token{
		{Name: "a", Value: "1"},
		{Name: "b", Value: "{a}"},
		{Name: "c", Value: "{b}"},
	}

	graph := resolver.BuildDependencyGraph(tokens)

	sorted, err := graph.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sorted) != 3 {
		t.Fatalf("expected 3 nodes in sort, got %d", len(sorted))
	}

	// "a" must appear before "b", and "b" before "c"
	indexOf := make(map[string]int)
	for i, name := range sorted {
		indexOf[name] = i
	}

	if indexOf["a"] > indexOf["b"] {
		t.Errorf("'a' should appear before 'b' in topological sort")
	}
	if indexOf["b"] > indexOf["c"] {
		t.Errorf("'b' should appear before 'c' in topological sort")
	}
}

func TestDependencyGraph_TopologicalSort_Cycle(t *testing.T) {
	tokens := []*token.Token{
		{Name: "a", Value: "{b}"},
		{Name: "b", Value: "{a}"},
	}

	graph := resolver.BuildDependencyGraph(tokens)

	_, err := graph.TopologicalSort()
	if err == nil {
		t.Error("expected error for cyclic graph")
	}
}

func TestDependencyGraph_JSONPointerRef(t *testing.T) {
	tokens := []*token.Token{
		{Name: "color-primary", Value: "#FF6B35", SchemaVersion: schema.V2025_10},
		{Name: "color-secondary", Value: "#/color/primary", SchemaVersion: schema.V2025_10},
	}

	graph := resolver.BuildDependencyGraph(tokens)

	deps := graph.Dependencies("color-secondary")
	if len(deps) != 1 || deps[0] != "color-primary" {
		t.Errorf("expected [color-primary], got %v", deps)
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

func TestResolveAliases_V2025_10_CurlyRefs(t *testing.T) {
	// V2025_10 supports both $ref (JSON Pointer) and curly-brace syntax
	// This tests curly-brace refs in V2025_10 schema
	structuredColor := map[string]any{
		"colorSpace": "srgb",
		"components": []float64{1, 0.42, 0.21},
		"alpha":      1,
	}

	tokens := []*token.Token{
		{Name: "color-brand-primary", Value: "", RawValue: structuredColor},
		{Name: "color-brand-secondary", Value: "{color.brand.primary}"},
		{Name: "color-semantic-action", Value: "{color.brand.secondary}"},
	}

	err := resolver.ResolveAliases(tokens, schema.V2025_10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Primary should resolve to structured color
	if tokens[0].ResolvedValue == nil {
		t.Error("expected primary to have resolved value")
	}

	// Secondary should resolve to structured color via curly-brace ref
	resolved, ok := tokens[1].ResolvedValue.(map[string]any)
	if !ok {
		t.Fatalf("expected secondary to resolve to map, got %T", tokens[1].ResolvedValue)
	}
	if resolved["colorSpace"] != "srgb" {
		t.Errorf("expected colorSpace srgb, got %v", resolved["colorSpace"])
	}

	// Action should resolve through chain: action -> secondary -> primary
	actionResolved, ok := tokens[2].ResolvedValue.(map[string]any)
	if !ok {
		t.Fatalf("expected action to resolve to map, got %T", tokens[2].ResolvedValue)
	}
	if actionResolved["colorSpace"] != "srgb" {
		t.Errorf("expected colorSpace srgb, got %v", actionResolved["colorSpace"])
	}

	// Check resolution chains
	if len(tokens[1].ResolutionChain) != 1 || tokens[1].ResolutionChain[0] != "color-brand-primary" {
		t.Errorf("expected secondary chain [color-brand-primary], got %v", tokens[1].ResolutionChain)
	}
	if len(tokens[2].ResolutionChain) != 2 {
		t.Errorf("expected action chain length 2, got %d", len(tokens[2].ResolutionChain))
	}
}
