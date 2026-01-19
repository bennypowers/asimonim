/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package resolver

import (
	"fmt"
	"strings"

	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/token"
)

// ResolveAliases resolves all alias references in the token list.
// Updates ResolvedValue and IsResolved fields on each token.
func ResolveAliases(tokens []*token.Token, version schema.Version) error {
	graph := BuildDependencyGraph(tokens)

	if graph.HasCycle() {
		cycle := graph.FindCycle()
		return fmt.Errorf("%w: %v", schema.ErrCircularReference, cycle)
	}

	sortedNames, err := graph.TopologicalSort()
	if err != nil {
		return err
	}

	tokenByName := make(map[string]*token.Token)
	for _, tok := range tokens {
		tokenByName[tok.Name] = tok
	}

	for _, name := range sortedNames {
		tok := tokenByName[name]
		if tok == nil {
			continue
		}
		resolveToken(tok, tokenByName, version)
	}

	return nil
}

func resolveToken(tok *token.Token, tokenByName map[string]*token.Token, version schema.Version) {
	if tok.IsResolved {
		return
	}

	isAlias := false

	if strings.Contains(tok.Value, "{") {
		isAlias = true
		result := resolveCurlyBraceRef(tok.Value, tokenByName)
		if !result.ok {
			// Resolution failed - use original value as fallback
			tok.ResolvedValue = tok.Value
			tok.ResolutionChain = nil
			tok.IsResolved = true
			return
		}
		tok.ResolvedValue = result.value
		tok.ResolutionChain = result.chain
	} else if version != schema.Draft && strings.HasPrefix(tok.Value, "#/") {
		isAlias = true
		result := resolveJSONPointerRef(tok.Value, tokenByName)
		if !result.ok {
			// Resolution failed - use original value as fallback
			tok.ResolvedValue = tok.Value
			tok.ResolutionChain = nil
			tok.IsResolved = true
			return
		}
		tok.ResolvedValue = result.value
		tok.ResolutionChain = result.chain
	}

	if !isAlias {
		if tok.RawValue != nil {
			tok.ResolvedValue = tok.RawValue
		} else {
			tok.ResolvedValue = tok.Value
		}
	}

	tok.IsResolved = true
}

// resolveResult holds the result of resolving a reference.
type resolveResult struct {
	value any
	chain []string
	ok    bool
}

func resolveCurlyBraceRef(value string, tokenByName map[string]*token.Token) resolveResult {
	refs := extractCurlyBraceRefs(value)
	if len(refs) == 0 {
		return resolveResult{value: value, ok: true}
	}

	// Per DTCG spec, curly brace syntax references complete token values only.
	// Partial references (e.g., "1px solid {color.red}") are not specified
	// and are returned unchanged.
	if len(refs) > 1 || !strings.HasPrefix(value, "{") || !strings.HasSuffix(value, "}") {
		return resolveResult{value: value, ok: true}
	}

	ref := refs[0]
	tokenName := strings.ReplaceAll(ref, ".", "-")

	refToken := tokenByName[tokenName]
	if refToken == nil {
		// Reference not found - leave unresolved
		return resolveResult{ok: false}
	}

	if !refToken.IsResolved {
		// Referenced token not yet resolved - leave unresolved
		return resolveResult{ok: false}
	}

	// Build the chain: this reference + any chain from the referenced token
	chain := []string{refToken.Name}
	chain = append(chain, refToken.ResolutionChain...)

	return resolveResult{value: refToken.ResolvedValue, chain: chain, ok: true}
}

func resolveJSONPointerRef(value string, tokenByName map[string]*token.Token) resolveResult {
	path := strings.TrimPrefix(value, "#/")
	tokenName := strings.ReplaceAll(path, "/", "-")

	refToken := tokenByName[tokenName]
	if refToken == nil {
		return resolveResult{ok: false}
	}

	if !refToken.IsResolved {
		return resolveResult{ok: false}
	}

	// Build the chain: this reference + any chain from the referenced token
	chain := []string{refToken.Name}
	chain = append(chain, refToken.ResolutionChain...)

	return resolveResult{value: refToken.ResolvedValue, chain: chain, ok: true}
}
