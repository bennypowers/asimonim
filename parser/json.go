/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package parser

import (
	"fmt"
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

// Parse parses JSON token data and returns tokens.
func (p *JSONParser) Parse(data []byte, opts Options) ([]*token.Token, error) {
	// Remove comments using jsonc
	cleanJSON := jsonc.ToJSON(data)

	// Parse JSON with yaml.v3 to get AST with position data
	var root yaml.Node
	if err := yaml.Unmarshal(cleanJSON, &root); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	result := []*token.Token{}
	if len(root.Content) > 0 {
		if err := p.extractTokens(root.Content[0], []string{}, "", opts, &result); err != nil {
			return nil, err
		}
	}

	return result, nil
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

// getNodeValue finds a child node by key in a mapping node.
func getNodeValue(node *yaml.Node, key string) *yaml.Node {
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// extractPosition extracts line and character from AST node (yaml.v3 is 1-based, we use 0-based).
func extractPosition(keyNode *yaml.Node) (line, character uint32, err error) {
	if keyNode.Line > 0 {
		lineVal := keyNode.Line - 1
		if lineVal < 0 || lineVal > math.MaxUint32 {
			return 0, 0, fmt.Errorf("position line %d exceeds uint32 limit", lineVal)
		}
		line = uint32(lineVal)
	}
	if keyNode.Column > 0 {
		colVal := keyNode.Column - 1
		if colVal < 0 || colVal > math.MaxUint32 {
			return 0, 0, fmt.Errorf("position column %d exceeds uint32 limit", colVal)
		}
		character = uint32(colVal)
	}
	return line, character, nil
}

// extractMetadata extracts DTCG metadata fields from value node.
func extractMetadata(valueNode *yaml.Node, t *token.Token) {
	if typeNode := getNodeValue(valueNode, "$type"); typeNode != nil {
		t.Type = typeNode.Value
	}
	if descNode := getNodeValue(valueNode, "$description"); descNode != nil {
		t.Description = descNode.Value
	}
	if deprecatedNode := getNodeValue(valueNode, "$deprecated"); deprecatedNode != nil {
		if deprecatedNode.Kind == yaml.ScalarNode {
			if deprecatedNode.Tag == "!!bool" {
				t.Deprecated = deprecatedNode.Value == "true"
			} else {
				t.Deprecated = true
				t.DeprecationMessage = deprecatedNode.Value
			}
		}
	}
	if extensionsNode := getNodeValue(valueNode, "$extensions"); extensionsNode != nil {
		var extensions map[string]any
		if err := extensionsNode.Decode(&extensions); err == nil {
			t.Extensions = extensions
		}
	}
}

// isGroupMarker checks if a key is in the group markers list.
func isGroupMarker(key string, groupMarkers []string) bool {
	return slices.Contains(groupMarkers, key)
}

// isTransparent determines if a group marker should be transparent (not added to path).
func isTransparent(key string, valueNode *yaml.Node, groupMarkers []string) bool {
	if !isGroupMarker(key, groupMarkers) {
		return false
	}
	return getNodeValue(valueNode, "$value") == nil
}

// buildPaths builds the JSON path and string path.
// Returns the new jsonPath slice and string path.
// The returned slice shares capacity with the input for recursion efficiency,
// but is clipped to prevent mutation of parent paths.
func buildPaths(jsonPath []string, path, key string, transparent bool) ([]string, string) {
	if transparent {
		return jsonPath, path
	}
	// Append key to path - this may reuse capacity from parent
	currentPath := append(jsonPath, key)
	// Clip to prevent child modifications from affecting this level
	currentPath = slices.Clip(currentPath)
	newPath := key
	if path != "" {
		newPath = path + "-" + key
	}
	return currentPath, newPath
}

// extractTokens recursively extracts tokens from AST.
func (p *JSONParser) extractTokens(node *yaml.Node, jsonPath []string, path string, opts Options, result *[]*token.Token) error {
	if node.Kind != yaml.MappingNode {
		return nil
	}

	// Collect key-value pairs
	type kvPair struct {
		keyNode   *yaml.Node
		valueNode *yaml.Node
	}
	numPairs := len(node.Content) / 2
	pairs := make([]kvPair, numPairs)
	for i := range numPairs {
		pairs[i] = kvPair{
			keyNode:   node.Content[i*2],
			valueNode: node.Content[i*2+1],
		}
	}

	// Sort for deterministic order unless SkipSort is set
	if !opts.SkipSort {
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].keyNode.Value < pairs[j].keyNode.Value
		})
	}

	for _, pair := range pairs {
		keyNode := pair.keyNode
		valueNode := pair.valueNode
		key := keyNode.Value

		// Skip $schema field
		if key == "$schema" {
			continue
		}

		// Skip non-mapping values
		if valueNode.Kind != yaml.MappingNode {
			continue
		}

		// Check for token indicators
		dollarValueNode := getNodeValue(valueNode, "$value")
		dollarRefNode := getNodeValue(valueNode, "$ref")
		hasValue := dollarValueNode != nil
		hasRef := dollarRefNode != nil && opts.SchemaVersion != schema.Draft

		// Check for root token
		isRootToken := common.IsRootToken(key, opts.SchemaVersion, opts.GroupMarkers)
		isTransparentMarker := isTransparent(key, valueNode, opts.GroupMarkers)
		isMarker := isGroupMarker(key, opts.GroupMarkers) && opts.SchemaVersion == schema.Draft

		// Build paths
		currentPath, newPath := buildPaths(jsonPath, path, key, isTransparentMarker || isRootToken)

		// Extract token if has $value or $ref
		if hasValue || hasRef {
			t, err := p.createToken(keyNode, path, valueNode, currentPath, opts, isRootToken)
			if err != nil {
				return err
			}
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
			childNode := p.filterChildNode(valueNode)
			if len(childNode.Content) > 0 {
				if err := p.extractTokens(childNode, currentPath, newPath, opts, result); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// filterChildNode creates a child node with metadata keys filtered out.
func (p *JSONParser) filterChildNode(valueNode *yaml.Node) *yaml.Node {
	// Pre-allocate with capacity for all content (we'll use less due to filtering)
	childNode := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: make([]*yaml.Node, 0, len(valueNode.Content)),
	}
	for i := 0; i < len(valueNode.Content); i += 2 {
		k := valueNode.Content[i].Value
		if k == "$type" || k == "$value" || k == "$description" || k == "$extensions" || k == "$deprecated" || k == "$schema" {
			continue
		}
		childNode.Content = append(childNode.Content, valueNode.Content[i], valueNode.Content[i+1])
	}
	return childNode
}

// createToken creates a Token from AST nodes.
func (p *JSONParser) createToken(keyNode *yaml.Node, path string, valueNode *yaml.Node, jsonPath []string, opts Options, isRootToken bool) (*token.Token, error) {
	key := keyNode.Value

	// Build token name
	name := path
	if name == "" {
		name = key
	} else if !isRootToken {
		name = path + "-" + key
	}

	// Build reference format
	reference := "{" + strings.Join(jsonPath, ".") + "}"

	// Extract position
	line, character, err := extractPosition(keyNode)
	if err != nil {
		return nil, err
	}

	// Extract value
	dollarValueNode := getNodeValue(valueNode, "$value")
	dollarRefNode := getNodeValue(valueNode, "$ref")

	value := ""
	var rawValue any

	if dollarValueNode != nil {
		if dollarValueNode.Kind == yaml.ScalarNode {
			value = dollarValueNode.Value
			rawValue = value
		} else {
			var structuredValue any
			if err := dollarValueNode.Decode(&structuredValue); err == nil {
				rawValue = structuredValue
			}
		}
	} else if dollarRefNode != nil && opts.SchemaVersion != schema.Draft {
		value = dollarRefNode.Value
		rawValue = value
	}

	t := &token.Token{
		Name:          name,
		Value:         value,
		Prefix:        opts.Prefix,
		Path:          jsonPath,
		Reference:     reference,
		Line:          line,
		Character:     character,
		SchemaVersion: opts.SchemaVersion,
		RawValue:      rawValue,
		IsResolved:    false,
	}

	extractMetadata(valueNode, t)

	return t, nil
}
