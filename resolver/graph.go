/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package resolver provides token reference resolution.
package resolver

import (
	"fmt"
	"strings"

	"bennypowers.dev/asimonim/parser/common"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/token"
)

// DependencyGraph represents a directed graph of token dependencies.
type DependencyGraph struct {
	dependencies map[string][]string
	dependents   map[string][]string
	nodes        map[string]bool
}

// BuildDependencyGraph builds a dependency graph from a list of tokens.
func BuildDependencyGraph(tokens []*token.Token) *DependencyGraph {
	graph := &DependencyGraph{
		dependencies: make(map[string][]string),
		dependents:   make(map[string][]string),
		nodes:        make(map[string]bool),
	}

	for _, tok := range tokens {
		graph.nodes[tok.Name] = true
	}

	for _, tok := range tokens {
		deps := extractDependencies(tok)
		if len(deps) > 0 {
			graph.dependencies[tok.Name] = deps
			for _, dep := range deps {
				graph.dependents[dep] = append(graph.dependents[dep], tok.Name)
			}
		}
	}

	return graph
}

// extractDependencies extracts token names that this token depends on.
func extractDependencies(tok *token.Token) []string {
	deps := []string{}

	// Check for curly brace references in Value
	if strings.Contains(tok.Value, "{") {
		refs := extractCurlyBraceRefs(tok.Value)
		for _, ref := range refs {
			tokenName := strings.ReplaceAll(ref, ".", "-")
			deps = append(deps, tokenName)
		}
	}

	// Check for JSON Pointer references ($ref field)
	if tok.SchemaVersion != schema.Draft && strings.HasPrefix(tok.Value, "#/") {
		path := strings.TrimPrefix(tok.Value, "#/")
		tokenName := strings.ReplaceAll(path, "/", "-")
		deps = append(deps, tokenName)
	}

	return deps
}

// extractCurlyBraceRefs extracts token paths from curly brace references.
func extractCurlyBraceRefs(value string) []string {
	refs := []string{}
	matches := common.CurlyBraceRefPattern.FindAllStringSubmatch(value, -1)
	for _, match := range matches {
		if len(match) > 1 {
			refs = append(refs, match[1])
		}
	}
	return refs
}

// Dependencies returns the list of tokens that the given token depends on.
func (g *DependencyGraph) Dependencies(tokenName string) []string {
	if deps, ok := g.dependencies[tokenName]; ok {
		return deps
	}
	return []string{}
}

// Dependents returns the list of tokens that depend on the given token.
func (g *DependencyGraph) Dependents(tokenName string) []string {
	if deps, ok := g.dependents[tokenName]; ok {
		return deps
	}
	return []string{}
}

// HasCycle returns true if the graph contains a circular dependency.
func (g *DependencyGraph) HasCycle() bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for node := range g.nodes {
		if g.hasCycleDFS(node, visited, recStack) {
			return true
		}
	}
	return false
}

func (g *DependencyGraph) hasCycleDFS(node string, visited, recStack map[string]bool) bool {
	if recStack[node] {
		return true
	}
	if visited[node] {
		return false
	}

	visited[node] = true
	recStack[node] = true

	for _, dep := range g.dependencies[node] {
		if g.hasCycleDFS(dep, visited, recStack) {
			return true
		}
	}

	recStack[node] = false
	return false
}

// FindCycle returns the cycle path if one exists, or nil if no cycle.
func (g *DependencyGraph) FindCycle() []string {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := []string{}

	for node := range g.nodes {
		if cycle := g.findCycleDFS(node, visited, recStack, path); cycle != nil {
			return cycle
		}
	}
	return nil
}

func (g *DependencyGraph) findCycleDFS(node string, visited, recStack map[string]bool, path []string) []string {
	if recStack[node] {
		cycleStart := -1
		for i, n := range path {
			if n == node {
				cycleStart = i
				break
			}
		}
		if cycleStart == -1 {
			panic(fmt.Sprintf("cycle detection invariant violated: node %q in recStack but not in path %v", node, path))
		}
		return append(path[cycleStart:], node)
	}
	if visited[node] {
		return nil
	}

	visited[node] = true
	recStack[node] = true
	path = append(path, node)

	for _, dep := range g.dependencies[node] {
		if cycle := g.findCycleDFS(dep, visited, recStack, path); cycle != nil {
			return cycle
		}
	}

	recStack[node] = false
	return nil
}

// TopologicalSort returns tokens in dependency order (dependencies first).
// Returns error if graph contains a cycle.
func (g *DependencyGraph) TopologicalSort() ([]string, error) {
	if cycle := g.FindCycle(); cycle != nil {
		return nil, fmt.Errorf("%w: %v", schema.ErrCircularReference, cycle)
	}

	visited := make(map[string]bool)
	result := []string{}

	for node := range g.nodes {
		if !visited[node] {
			g.topologicalSortDFS(node, visited, &result)
		}
	}

	return result, nil
}

func (g *DependencyGraph) topologicalSortDFS(node string, visited map[string]bool, stack *[]string) {
	visited[node] = true

	for _, dep := range g.dependencies[node] {
		if !visited[dep] {
			g.topologicalSortDFS(dep, visited, stack)
		}
	}

	*stack = append(*stack, node)
}
