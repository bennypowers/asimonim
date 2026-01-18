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
		if err := resolveToken(tok, tokenByName, version); err != nil {
			return err
		}
	}

	return nil
}

func resolveToken(tok *token.Token, tokenByName map[string]*token.Token, version schema.Version) error {
	if tok.IsResolved {
		return nil
	}

	isAlias := false

	if strings.Contains(tok.Value, "{") {
		isAlias = true
		resolved, err := resolveCurlyBraceRef(tok.Value, tokenByName)
		if err != nil {
			return err
		}
		tok.ResolvedValue = resolved
	} else if version != schema.Draft && strings.HasPrefix(tok.Value, "#/") {
		isAlias = true
		resolved, err := resolveJSONPointerRef(tok.Value, tokenByName)
		if err != nil {
			return err
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
	return nil
}

func resolveCurlyBraceRef(value string, tokenByName map[string]*token.Token) (any, error) {
	refs := extractCurlyBraceRefs(value)
	if len(refs) == 0 {
		return value, nil
	}

	// Only support whole-token references for now
	if len(refs) > 1 || !strings.HasPrefix(value, "{") || !strings.HasSuffix(value, "}") {
		return value, nil
	}

	ref := refs[0]
	tokenName := strings.ReplaceAll(ref, ".", "-")

	refToken := tokenByName[tokenName]
	if refToken == nil {
		return nil, fmt.Errorf("%w: %s", schema.ErrUnresolvedReference, ref)
	}

	if !refToken.IsResolved {
		return nil, fmt.Errorf("referenced token not yet resolved: %s", ref)
	}

	return refToken.ResolvedValue, nil
}

func resolveJSONPointerRef(value string, tokenByName map[string]*token.Token) (any, error) {
	path := strings.TrimPrefix(value, "#/")
	tokenName := strings.ReplaceAll(path, "/", "-")

	refToken := tokenByName[tokenName]
	if refToken == nil {
		return nil, fmt.Errorf("%w: %s", schema.ErrUnresolvedReference, value)
	}

	if !refToken.IsResolved {
		return nil, fmt.Errorf("referenced token not yet resolved: %s", value)
	}

	return refToken.ResolvedValue, nil
}
