/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package resolver

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/token"
	"gopkg.in/yaml.v3"
)

// groupExtension represents a group that extends another group.
type groupExtension struct {
	// path is the JSON path to this group (e.g., ["theme"])
	path []string
	// extendsPath is the JSON path to the extended group (e.g., ["base"])
	extendsPath []string
}

// ResolveGroupExtensions resolves $extends relationships in DTCG 2025.10 files.
// It creates copies of inherited tokens with updated paths and names.
// Child tokens override inherited tokens with the same terminal name.
//
// This function should be called AFTER parsing, BEFORE alias resolution.
// For Draft schema, this is a no-op that returns the tokens unchanged.
func ResolveGroupExtensions(tokens []*token.Token, data []byte) ([]*token.Token, error) {
	if len(tokens) == 0 {
		return tokens, nil
	}

	// Check if any tokens are V2025_10 schema
	isV2025 := false
	for _, t := range tokens {
		if t.SchemaVersion == schema.V2025_10 {
			isV2025 = true
			break
		}
	}
	if !isV2025 {
		return tokens, nil
	}

	// Parse raw data to find $extends relationships
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse data for extends resolution: %w", err)
	}

	// Find all groups with $extends
	extensions := findExtensions(raw, nil)
	if len(extensions) == 0 {
		return tokens, nil
	}

	// Build extension dependency graph and check for cycles
	if cycle := findExtensionCycle(extensions); cycle != nil {
		return nil, fmt.Errorf("%w in $extends: %s", schema.ErrCircularReference, strings.Join(cycle, " -> "))
	}

	// Sort extensions in topological order (base groups first)
	sortedExtensions := topologicalSortExtensions(extensions)

	// Build token map by path prefix for quick lookup
	tokensByPathPrefix := make(map[string][]*token.Token)
	for _, t := range tokens {
		prefix := strings.Join(t.Path, "/")
		tokensByPathPrefix[prefix] = append(tokensByPathPrefix[prefix], t)
	}

	// Track which terminal names exist in each extending group (for override detection)
	terminalNamesByGroup := make(map[string]map[string]bool)
	for _, t := range tokens {
		if len(t.Path) == 0 {
			continue
		}
		groupPath := strings.Join(t.Path[:len(t.Path)-1], "/")
		if terminalNamesByGroup[groupPath] == nil {
			terminalNamesByGroup[groupPath] = make(map[string]bool)
		}
		terminalName := t.Path[len(t.Path)-1]
		terminalNamesByGroup[groupPath][terminalName] = true
	}

	// Process extensions in order
	result := slices.Clone(tokens)
	for _, ext := range sortedExtensions {
		inherited, err := resolveExtension(ext, result, terminalNamesByGroup)
		if err != nil {
			return nil, err
		}
		result = append(result, inherited...)

		// Update terminal names for the extending group with newly inherited tokens
		extGroupPath := strings.Join(ext.path, "/")
		if terminalNamesByGroup[extGroupPath] == nil {
			terminalNamesByGroup[extGroupPath] = make(map[string]bool)
		}
		for _, t := range inherited {
			if len(t.Path) > 0 {
				terminalName := t.Path[len(t.Path)-1]
				terminalNamesByGroup[extGroupPath][terminalName] = true
			}
		}
	}

	// Sort result for deterministic output
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}

// findExtensions recursively finds all groups with $extends.
func findExtensions(data map[string]any, currentPath []string) []groupExtension {
	var extensions []groupExtension

	for key, value := range data {
		if strings.HasPrefix(key, "$") {
			continue
		}

		valueMap, ok := value.(map[string]any)
		if !ok {
			continue
		}

		childPath := append(slices.Clone(currentPath), key)

		// Check if this group has $extends
		if extendsRef, ok := valueMap["$extends"].(string); ok {
			extendsPath := parseJSONPointer(extendsRef)
			if extendsPath != nil {
				extensions = append(extensions, groupExtension{
					path:        childPath,
					extendsPath: extendsPath,
				})
			}
		}

		// Recurse into children
		childExtensions := findExtensions(valueMap, childPath)
		extensions = append(extensions, childExtensions...)
	}

	return extensions
}

// parseJSONPointer parses a JSON Pointer reference (e.g., "#/base/colors") into path segments.
func parseJSONPointer(ref string) []string {
	if !strings.HasPrefix(ref, "#/") {
		return nil
	}
	path := strings.TrimPrefix(ref, "#/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

// findExtensionCycle detects circular $extends references.
// Returns the cycle path if found, nil otherwise.
func findExtensionCycle(extensions []groupExtension) []string {
	// Build adjacency map: extending group -> extended group
	extendsMap := make(map[string]string)
	for _, ext := range extensions {
		from := strings.Join(ext.path, "/")
		to := strings.Join(ext.extendsPath, "/")
		extendsMap[from] = to
	}

	// Check for cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var findCycleDFS func(node string, path []string) []string
	findCycleDFS = func(node string, path []string) []string {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		if next, ok := extendsMap[node]; ok {
			if recStack[next] {
				// Found cycle - return the cycle path
				cycleStart := slices.Index(path, next)
				if cycleStart >= 0 {
					return append(path[cycleStart:], next)
				}
				return append(path, next)
			}
			if !visited[next] {
				if cycle := findCycleDFS(next, path); cycle != nil {
					return cycle
				}
			}
		}

		recStack[node] = false
		return nil
	}

	for _, ext := range extensions {
		node := strings.Join(ext.path, "/")
		if !visited[node] {
			if cycle := findCycleDFS(node, nil); cycle != nil {
				return cycle
			}
		}
	}

	return nil
}

// topologicalSortExtensions sorts extensions so that base groups come first.
func topologicalSortExtensions(extensions []groupExtension) []groupExtension {
	// Build adjacency map
	extendsMap := make(map[string]string)
	extByPath := make(map[string]groupExtension)
	for _, ext := range extensions {
		from := strings.Join(ext.path, "/")
		to := strings.Join(ext.extendsPath, "/")
		extendsMap[from] = to
		extByPath[from] = ext
	}

	// Calculate depth (distance from root) for each extension
	depths := make(map[string]int)
	var getDepth func(path string) int
	getDepth = func(path string) int {
		if d, ok := depths[path]; ok {
			return d
		}
		if next, ok := extendsMap[path]; ok {
			depths[path] = getDepth(next) + 1
		} else {
			depths[path] = 0
		}
		return depths[path]
	}

	for _, ext := range extensions {
		path := strings.Join(ext.path, "/")
		getDepth(path)
	}

	// Sort by depth (base groups first)
	result := slices.Clone(extensions)
	sort.Slice(result, func(i, j int) bool {
		pathI := strings.Join(result[i].path, "/")
		pathJ := strings.Join(result[j].path, "/")
		return depths[pathI] < depths[pathJ]
	})

	return result
}

// resolveExtension creates inherited tokens for a single extension.
func resolveExtension(ext groupExtension, tokens []*token.Token, terminalNames map[string]map[string]bool) ([]*token.Token, error) {
	extGroupPath := strings.Join(ext.path, "/")
	basePrefix := strings.Join(ext.extendsPath, "-")
	newPrefix := strings.Join(ext.path, "-")

	// Get terminal names that exist in the extending group (for override detection)
	existingTerminals := terminalNames[extGroupPath]
	if existingTerminals == nil {
		existingTerminals = make(map[string]bool)
	}

	var inherited []*token.Token

	for _, t := range tokens {
		// Check if this token belongs to the extended group
		if !tokenBelongsToGroup(t, ext.extendsPath) {
			continue
		}

		// Get the relative path within the extended group
		relativePath := t.Path[len(ext.extendsPath):]
		if len(relativePath) == 0 {
			continue
		}

		// Check for override - if terminal name exists in extending group, skip
		terminalName := relativePath[0]
		if len(relativePath) == 1 && existingTerminals[terminalName] {
			continue
		}

		// Create a copy with updated path and name
		newPath := append(slices.Clone(ext.path), relativePath...)
		newName := strings.ReplaceAll(t.Name, basePrefix, newPrefix)

		inherited = append(inherited, &token.Token{
			Name:              newName,
			Value:             t.Value,
			Type:              t.Type,
			Description:       t.Description,
			Extensions:        t.Extensions,
			Deprecated:        t.Deprecated,
			DeprecationMessage: t.DeprecationMessage,
			FilePath:          t.FilePath,
			Prefix:            t.Prefix,
			Path:              newPath,
			Reference:         "{" + strings.Join(newPath, ".") + "}",
			SchemaVersion:     t.SchemaVersion,
			RawValue:          t.RawValue,
			// Inherited tokens start unresolved
			IsResolved: false,
		})
	}

	return inherited, nil
}

// tokenBelongsToGroup checks if a token's path starts with the given group path.
func tokenBelongsToGroup(t *token.Token, groupPath []string) bool {
	if len(t.Path) <= len(groupPath) {
		return false
	}
	for i, segment := range groupPath {
		if t.Path[i] != segment {
			return false
		}
	}
	return true
}
