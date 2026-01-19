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
		resolved, ok := resolveCurlyBraceRef(tok.Value, tokenByName)
		if !ok {
			// Leave unresolved - will show raw reference
			return
		}
		tok.ResolvedValue = resolved
	} else if version != schema.Draft && strings.HasPrefix(tok.Value, "#/") {
		isAlias = true
		resolved, ok := resolveJSONPointerRef(tok.Value, tokenByName)
		if !ok {
			return
		}
		tok.ResolvedValue = resolved
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

func resolveCurlyBraceRef(value string, tokenByName map[string]*token.Token) (any, bool) {
	refs := extractCurlyBraceRefs(value)
	if len(refs) == 0 {
		return value, true
	}

	// Only support whole-token references for now
	if len(refs) > 1 || !strings.HasPrefix(value, "{") || !strings.HasSuffix(value, "}") {
		return value, true
	}

	ref := refs[0]
	tokenName := strings.ReplaceAll(ref, ".", "-")

	refToken := tokenByName[tokenName]
	if refToken == nil {
		// Reference not found - leave unresolved
		return nil, false
	}

	if !refToken.IsResolved {
		// Referenced token not yet resolved - leave unresolved
		return nil, false
	}

	return refToken.ResolvedValue, true
}

func resolveJSONPointerRef(value string, tokenByName map[string]*token.Token) (any, bool) {
	path := strings.TrimPrefix(value, "#/")
	tokenName := strings.ReplaceAll(path, "/", "-")

	refToken := tokenByName[tokenName]
	if refToken == nil {
		return nil, false
	}

	if !refToken.IsResolved {
		return nil, false
	}

	return refToken.ResolvedValue, true
}
