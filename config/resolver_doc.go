/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	asimfs "bennypowers.dev/asimonim/fs"
	"bennypowers.dev/asimonim/specifier"
)

// resolverDocument represents the structure of a DTCG resolver document.
type resolverDocument struct {
	Version         string                 `json:"version"`
	Sets            map[string]setDef      `json:"sets"`
	Modifiers       map[string]modifierDef `json:"modifiers"`
	ResolutionOrder json.RawMessage        `json:"resolutionOrder"`
}

// setDef represents a named set in a resolver document.
type setDef struct {
	Sources []sourceRef `json:"sources"`
}

// modifierDef represents a named modifier in a resolver document.
type modifierDef struct {
	Contexts map[string][]sourceRef `json:"contexts"`
}

// sourceRef represents a source reference in a resolver document.
type sourceRef struct {
	Ref string `json:"$ref"`
}

// ResolveResolverSources reads resolver documents and returns their source file paths
// as ResolvedFile entries. Each resolver document is parsed to extract $ref entries
// from its resolution order, and those paths are resolved relative to the resolver
// document's directory.
func (c *Config) ResolveResolverSources(resolver specifier.Resolver, filesystem asimfs.FileSystem, rootDir string) ([]*specifier.ResolvedFile, error) {
	// First resolve the resolver document paths themselves
	resolverFiles, err := c.ResolveResolvers(resolver, filesystem, rootDir)
	if err != nil {
		return nil, err
	}

	var result []*specifier.ResolvedFile
	seen := make(map[string]bool)

	for _, rf := range resolverFiles {
		sourcePaths, err := extractResolverSourcePaths(filesystem, rf.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to extract sources from resolver %s: %w", rf.Specifier, err)
		}

		for _, srcPath := range sourcePaths {
			if seen[srcPath] {
				continue
			}
			seen[srcPath] = true

			result = append(result, &specifier.ResolvedFile{
				Specifier: srcPath,
				Path:      srcPath,
				Kind:      specifier.KindLocal,
			})
		}
	}

	return result, nil
}

// extractResolverSourcePaths reads a resolver document and extracts source file paths.
func extractResolverSourcePaths(filesystem asimfs.FileSystem, resolverPath string) ([]string, error) {
	data, err := filesystem.ReadFile(resolverPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read resolver document: %w", err)
	}

	var doc resolverDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse resolver document: %w", err)
	}

	var entries []json.RawMessage
	if err := json.Unmarshal(doc.ResolutionOrder, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse resolutionOrder: %w", err)
	}

	resolverDir := filepath.Dir(resolverPath)
	var paths []string
	seen := make(map[string]bool)

	for i, entry := range entries {
		entryPaths, err := resolveEntry(entry, doc.Sets, doc.Modifiers)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve entry %d: %w", i, err)
		}
		for _, p := range entryPaths {
			absPath := resolveRefPath(p, resolverDir)
			if !seen[absPath] {
				seen[absPath] = true
				paths = append(paths, absPath)
			}
		}
	}

	return paths, nil
}

// resolveEntry extracts source file paths from a resolution order entry.
// Per the DTCG 2025.10 resolver spec, an entry can be:
//   - A reference to a named set: {"$ref": "#/sets/base"}
//   - A reference to a named modifier: {"$ref": "#/modifiers/mode"}
//   - An inline set with "sources": [{"$ref": "./file.json"}, ...]
//   - An inline modifier with "contexts": {"light": [...], "dark": [...]}
func resolveEntry(entry json.RawMessage, sets map[string]setDef, modifiers map[string]modifierDef) ([]string, error) {
	var ref sourceRef
	if err := json.Unmarshal(entry, &ref); err == nil && ref.Ref != "" {
		if rawName, ok := strings.CutPrefix(ref.Ref, "#/sets/"); ok {
			setName := unescapeJSONPointer(rawName)
			set, exists := sets[setName]
			if !exists {
				return nil, fmt.Errorf("referenced set %q not found", setName)
			}
			return fileRefsFromSources(set.Sources), nil
		}
		if rawName, ok := strings.CutPrefix(ref.Ref, "#/modifiers/"); ok {
			modName := unescapeJSONPointer(rawName)
			mod, exists := modifiers[modName]
			if !exists {
				return nil, fmt.Errorf("referenced modifier %q not found", modName)
			}
			return fileRefsFromContexts(mod.Contexts), nil
		}
	}

	var inlineSet struct {
		Sources []sourceRef `json:"sources"`
	}
	if err := json.Unmarshal(entry, &inlineSet); err == nil && len(inlineSet.Sources) > 0 {
		return fileRefsFromSources(inlineSet.Sources), nil
	}

	var inlineModifier struct {
		Contexts map[string][]sourceRef `json:"contexts"`
	}
	if err := json.Unmarshal(entry, &inlineModifier); err == nil && len(inlineModifier.Contexts) > 0 {
		return fileRefsFromContexts(inlineModifier.Contexts), nil
	}

	return nil, fmt.Errorf("unrecognized resolution order entry: %s", string(entry))
}

// fileRefsFromContexts extracts file paths from all contexts of a modifier,
// collecting $ref entries from each context's source array.
func fileRefsFromContexts(contexts map[string][]sourceRef) []string {
	var paths []string
	for _, sources := range contexts {
		paths = append(paths, fileRefsFromSources(sources)...)
	}
	return paths
}

// fileRefsFromSources extracts file paths from source $ref entries,
// filtering out JSON pointer references.
func fileRefsFromSources(sources []sourceRef) []string {
	var paths []string
	for _, src := range sources {
		if src.Ref == "" || strings.HasPrefix(src.Ref, "#") {
			continue
		}
		// Strip any fragment identifier (e.g., "palette.json#/brand" → "palette.json")
		path, _, _ := strings.Cut(src.Ref, "#")
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

// unescapeJSONPointer decodes a JSON Pointer token per RFC 6901:
// percent-decoding first, then replacing ~1 with / and ~0 with ~.
func unescapeJSONPointer(s string) string {
	if unescaped, err := url.PathUnescape(s); err == nil {
		s = unescaped
	}
	s = strings.ReplaceAll(s, "~1", "/")
	s = strings.ReplaceAll(s, "~0", "~")
	return s
}

// resolveRefPath resolves a $ref path relative to the resolver document's directory.
func resolveRefPath(refPath, resolverDir string) string {
	if filepath.IsAbs(refPath) {
		return filepath.Clean(refPath)
	}
	return filepath.Clean(filepath.Join(resolverDir, refPath))
}
