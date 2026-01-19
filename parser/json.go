/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package parser

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
	"sort"
	"strings"

	"bennypowers.dev/asimonim/fs"
	"bennypowers.dev/asimonim/parser/common"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/token"
	"github.com/tidwall/jsonc"
	"gopkg.in/yaml.v3"
)

// JSONParser parses DTCG-compliant JSON token files.
type JSONParser struct{}

// NewJSONParser creates a new JSON token parser.
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// Parse parses JSON or YAML token data and returns tokens.
func (p *JSONParser) Parse(data []byte, opts Options) ([]*token.Token, error) {
	var raw map[string]any
	var positionData []byte

	// Detect format: JSON typically starts with '{' or whitespace then '{'
	// YAML uses indentation-based structure
	if isLikelyJSON(data) {
		// JSON path: strip comments and parse
		cleanJSON := jsonc.ToJSON(data)
		if err := json.Unmarshal(cleanJSON, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
		positionData = cleanJSON
	} else {
		// YAML path: parse directly with yaml.v3
		var yamlRaw any
		if err := yaml.Unmarshal(data, &yamlRaw); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
		// Normalize map types (YAML numeric keys create map[any]any)
		normalized := normalizeMap(yamlRaw)
		var ok bool
		raw, ok = normalized.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("YAML root must be an object")
		}
		positionData = data
	}

	// Extract tokens using the single extraction path
	result := []*token.Token{}
	p.extractTokens(raw, []string{}, "", "", opts, &result)

	// Optional second pass: add position tracking
	if !opts.SkipPositions {
		if err := p.addPositions(positionData, result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// isLikelyJSON checks if data appears to be JSON rather than YAML.
// JSON typically starts with '{' (optionally preceded by whitespace/BOM).
func isLikelyJSON(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\t', '\n', '\r':
			continue
		case 0xEF, 0xBB, 0xBF: // UTF-8 BOM
			continue
		case '{':
			return true
		default:
			return false
		}
	}
	return false
}

// normalizeMap recursively converts map[interface{}]interface{} to map[string]any.
// YAML with numeric keys (like "10:") creates map[interface{}]interface{},
// which must be normalized for our string-keyed processing.
func normalizeMap(v any) any {
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			x[k] = normalizeMap(val)
		}
		return x
	case map[any]any:
		result := make(map[string]any, len(x))
		for k, val := range x {
			result[fmt.Sprintf("%v", k)] = normalizeMap(val)
		}
		return result
	case []any:
		for i, val := range x {
			x[i] = normalizeMap(val)
		}
		return x
	default:
		return v
	}
}

// extractTokens recursively extracts tokens from a parsed map.
// inheritedType is passed down from parent groups for $type inheritance.
func (p *JSONParser) extractTokens(data map[string]any, jsonPath []string, path, inheritedType string, opts Options, result *[]*token.Token) {
	// Check if this group has a $type that should be inherited by children
	currentType := inheritedType
	if groupType, ok := data["$type"].(string); ok {
		currentType = groupType
	}

	// Collect keys for sorting
	keys := make([]string, 0, len(data))
	for k := range data {
		if strings.HasPrefix(k, "$") {
			continue
		}
		keys = append(keys, k)
	}

	// Sort for deterministic order unless SkipSort is set
	if !opts.SkipSort {
		sort.Strings(keys)
	}

	for _, key := range keys {
		v := data[key]

		// Skip non-map values
		valueMap, ok := v.(map[string]any)
		if !ok {
			continue
		}

		// Check for token indicators
		dollarValue, hasValue := valueMap["$value"]
		dollarRef, hasRef := valueMap["$ref"]
		hasRef = hasRef && opts.SchemaVersion != schema.Draft

		// Check for root token / group markers
		isRootToken := common.IsRootToken(key, opts.SchemaVersion, opts.GroupMarkers)
		isTransparentMarker := p.isTransparent(key, valueMap, opts.GroupMarkers)
		isMarker := slices.Contains(opts.GroupMarkers, key) && opts.SchemaVersion == schema.Draft

		// Build paths - transparent markers don't affect either path
		// Value markers affect jsonPath (for references) but not name path
		currentPath, newPath := buildPaths(jsonPath, path, key, isTransparentMarker || isRootToken, isMarker)

		// Extract token if has $value or $ref
		if hasValue || hasRef {
			t := p.createToken(key, path, valueMap, currentPath, opts, isRootToken || isMarker, dollarValue, dollarRef, currentType)
			*result = append(*result, t)
		}

		// Determine if we should recurse
		shouldRecurse := false
		if !hasValue && !hasRef {
			shouldRecurse = true
		} else if isMarker || isRootToken {
			shouldRecurse = true
		}

		if shouldRecurse {
			// Get child type before filtering (for inheritance in nested groups)
			childType := currentType
			if typeStr, ok := valueMap["$type"].(string); ok {
				childType = typeStr
			}
			childMap := p.filterChildMap(valueMap)
			if len(childMap) > 0 {
				p.extractTokens(childMap, currentPath, newPath, childType, opts, result)
			}
		}
	}
}

// isTransparent checks if a key is a transparent group marker.
func (p *JSONParser) isTransparent(key string, valueMap map[string]any, groupMarkers []string) bool {
	if !slices.Contains(groupMarkers, key) {
		return false
	}
	_, hasValue := valueMap["$value"]
	return !hasValue
}

// filterChildMap filters out DTCG metadata keys from a map.
func (p *JSONParser) filterChildMap(valueMap map[string]any) map[string]any {
	result := make(map[string]any, len(valueMap))
	maps.Copy(result, valueMap)
	delete(result, "$type")
	delete(result, "$value")
	delete(result, "$description")
	delete(result, "$extensions")
	delete(result, "$deprecated")
	delete(result, "$schema")
	return result
}

// createToken creates a Token from map data.
// inheritedType is the $type from parent groups for inheritance.
func (p *JSONParser) createToken(key, path string, valueMap map[string]any, jsonPath []string, opts Options, isRootToken bool, dollarValue, dollarRef any, inheritedType string) *token.Token {
	// Build token name
	name := path
	if name == "" {
		name = key
	} else if !isRootToken {
		name = path + "-" + key
	}

	// Build reference format
	reference := "{" + strings.Join(jsonPath, ".") + "}"

	// Extract value
	value := ""
	var rawValue any

	if dollarValue != nil {
		if strVal, ok := dollarValue.(string); ok {
			value = strVal
			rawValue = value
		} else {
			rawValue = dollarValue
		}
	} else if dollarRef != nil && opts.SchemaVersion != schema.Draft {
		if strVal, ok := dollarRef.(string); ok {
			value = strVal
			rawValue = value
		}
	}

	t := &token.Token{
		Name:          name,
		Value:         value,
		Prefix:        opts.Prefix,
		Path:          jsonPath,
		Reference:     reference,
		Line:          0, // Filled in by addPositions if needed
		Character:     0,
		SchemaVersion: opts.SchemaVersion,
		RawValue:      rawValue,
		IsResolved:    false,
	}

	// Extract metadata - token's own $type takes precedence over inherited
	if typeStr, ok := valueMap["$type"].(string); ok {
		t.Type = typeStr
	} else if inheritedType != "" {
		t.Type = inheritedType
	}
	if descStr, ok := valueMap["$description"].(string); ok {
		t.Description = descStr
	}
	if deprecated, ok := valueMap["$deprecated"]; ok {
		if depBool, ok := deprecated.(bool); ok {
			t.Deprecated = depBool
		} else if depStr, ok := deprecated.(string); ok {
			t.Deprecated = true
			t.DeprecationMessage = depStr
		}
	}
	if extensions, ok := valueMap["$extensions"].(map[string]any); ok {
		t.Extensions = extensions
	}

	return t
}

// buildPaths builds the JSON path and string path.
// Returns the new jsonPath slice and string path.
// The returned slice shares capacity with the input for recursion efficiency,
// but is clipped to prevent mutation of parent paths.
func buildPaths(jsonPath []string, path, key string, transparent, nameTransparent bool) ([]string, string) {
	if transparent {
		return jsonPath, path
	}
	// Append key to jsonPath - this may reuse capacity from parent
	currentPath := append(jsonPath, key)
	// Clip to prevent child modifications from affecting this level
	currentPath = slices.Clip(currentPath)

	// For markers with values, add to jsonPath but not to name
	if nameTransparent {
		return currentPath, path
	}

	newPath := key
	if path != "" {
		newPath = path + "-" + key
	}
	return currentPath, newPath
}

// addPositions adds line/character positions to tokens by parsing with yaml.v3.
// This is a second pass that only runs when position tracking is enabled.
func (p *JSONParser) addPositions(data []byte, tokens []*token.Token) error {
	// Build a map from token path (as dot-separated string) to token pointer
	tokenByPath := make(map[string]*token.Token, len(tokens))
	for _, t := range tokens {
		pathKey := strings.Join(t.Path, ".")
		tokenByPath[pathKey] = t
	}

	// Parse with yaml.v3 to get AST with position data
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("failed to parse JSON for positions: %w", err)
	}

	// Walk the AST and update token positions
	if len(root.Content) > 0 {
		p.walkForPositions(root.Content[0], []string{}, tokenByPath)
	}

	return nil
}

// walkForPositions walks the yaml AST to find token positions.
func (p *JSONParser) walkForPositions(node *yaml.Node, jsonPath []string, tokenByPath map[string]*token.Token) {
	if node.Kind != yaml.MappingNode {
		return
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]
		key := keyNode.Value

		// Skip $ keys
		if strings.HasPrefix(key, "$") {
			continue
		}

		// Skip non-mapping values
		if valueNode.Kind != yaml.MappingNode {
			continue
		}

		// Build current path
		currentPath := append(jsonPath, key)
		currentPath = slices.Clip(currentPath)
		pathKey := strings.Join(currentPath, ".")

		// Check if this is a token we need to update
		if t, ok := tokenByPath[pathKey]; ok {
			// Extract position (yaml.v3 is 1-based, we use 0-based)
			if keyNode.Line > 0 {
				lineVal := keyNode.Line - 1
				if lineVal >= 0 && lineVal <= math.MaxUint32 {
					t.Line = uint32(lineVal)
				}
			}
			if keyNode.Column > 0 {
				colVal := keyNode.Column - 1
				if colVal >= 0 && colVal <= math.MaxUint32 {
					t.Character = uint32(colVal)
				}
			}
		}

		// Recurse into children
		p.walkForPositions(valueNode, currentPath, tokenByPath)
	}
}

// ParseFile parses a JSON token file and returns tokens.
func (p *JSONParser) ParseFile(filesystem fs.FileSystem, path string, opts Options) ([]*token.Token, error) {
	data, err := filesystem.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	tokens, err := p.Parse(data, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", path, err)
	}

	// Set FilePath on all tokens
	for _, t := range tokens {
		t.FilePath = path
	}

	return tokens, nil
}
