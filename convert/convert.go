/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package convert provides DTCG token serialization and schema conversion.
package convert

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mazznoer/csscolorparser"

	"bennypowers.dev/asimonim/parser/common"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/token"
)

// Options configures token serialization behavior.
type Options struct {
	// InputSchema is the schema version of the input tokens.
	// If Unknown, defaults to Draft.
	InputSchema schema.Version

	// OutputSchema is the target schema version for output.
	// If Unknown, matches InputSchema.
	OutputSchema schema.Version

	// Flatten produces a shallow structure with delimiter-separated keys
	// instead of nested groups.
	Flatten bool

	// Delimiter is the separator for flattened keys (default "-").
	Delimiter string

	// Format specifies the output format (default FormatDTCG).
	Format Format

	// Prefix is added to output variable names.
	Prefix string
}

// DefaultOptions returns options with sensible defaults.
func DefaultOptions() Options {
	return Options{
		InputSchema:  schema.Draft,
		OutputSchema: schema.Unknown,
		Flatten:      false,
		Delimiter:    "-",
		Format:       FormatDTCG,
	}
}

// curlyBraceRefPattern matches {token.path} references.
var curlyBraceRefPattern = regexp.MustCompile(`\{([^}]+)\}`)

// Serialize converts parsed tokens to a DTCG map structure.
func Serialize(tokens []*token.Token, opts Options) map[string]any {
	// Apply defaults
	if opts.Delimiter == "" {
		opts.Delimiter = "-"
	}
	if opts.InputSchema == schema.Unknown {
		opts.InputSchema = schema.Draft
	}
	if opts.OutputSchema == schema.Unknown {
		opts.OutputSchema = opts.InputSchema
	}

	if opts.Flatten {
		return buildFlatStructure(tokens, opts.InputSchema, opts.OutputSchema, opts.Delimiter)
	}
	return buildNestedStructure(tokens, opts.InputSchema, opts.OutputSchema)
}

// SerializeTokens converts parsed tokens to a DTCG map structure.
// Deprecated: Use Serialize with Options instead.
func SerializeTokens(
	tokens []*token.Token,
	inputSchema, outputSchema schema.Version,
	flatten bool,
	delimiter string,
) map[string]any {
	return Serialize(tokens, Options{
		InputSchema:  inputSchema,
		OutputSchema: outputSchema,
		Flatten:      flatten,
		Delimiter:    delimiter,
	})
}

// buildFlatStructure creates a shallow map with delimiter-separated keys.
func buildFlatStructure(
	tokens []*token.Token,
	inputSchema, outputSchema schema.Version,
	delimiter string,
) map[string]any {
	result := make(map[string]any)

	// Add $schema for v2025_10 output
	if outputSchema == schema.V2025_10 {
		result["$schema"] = outputSchema.URL()
	}

	for _, tok := range tokens {
		// Use Path segments joined by delimiter for flattened keys
		key := strings.Join(tok.Path, delimiter)
		tokenMap := serializeToken(tok, inputSchema, outputSchema)
		result[key] = tokenMap
	}

	return result
}

// buildNestedStructure creates a nested map following the token paths.
func buildNestedStructure(
	tokens []*token.Token,
	inputSchema, outputSchema schema.Version,
) map[string]any {
	result := make(map[string]any)

	// Add $schema for v2025_10 output
	if outputSchema == schema.V2025_10 {
		result["$schema"] = outputSchema.URL()
	}

	for _, tok := range tokens {
		current := result
		path := tok.Path

		// Navigate/create nested structure up to parent
		for i := 0; i < len(path)-1; i++ {
			segment := path[i]
			if _, exists := current[segment]; !exists {
				current[segment] = make(map[string]any)
			}
			current = current[segment].(map[string]any)
		}

		// Set the token at the final key
		if len(path) > 0 {
			current[path[len(path)-1]] = serializeToken(tok, inputSchema, outputSchema)
		}
	}

	return result
}

// serializeToken converts a single token to its DTCG map representation.
func serializeToken(tok *token.Token, inputSchema, outputSchema schema.Version) map[string]any {
	result := make(map[string]any)

	// Handle value conversion
	value := convertValue(tok, inputSchema, outputSchema)
	if value != nil {
		result["$value"] = value
	}

	if tok.Type != "" {
		result["$type"] = tok.Type
	}

	if tok.Description != "" {
		result["$description"] = tok.Description
	}

	if len(tok.Extensions) > 0 {
		result["$extensions"] = tok.Extensions
	}

	if tok.Deprecated {
		result["$deprecated"] = true
		if tok.DeprecationMessage != "" {
			result["$deprecationMessage"] = tok.DeprecationMessage
		}
	}

	return result
}

// convertValue handles value conversion between schemas.
func convertValue(tok *token.Token, inputSchema, outputSchema schema.Version) any {
	rawValue := tok.RawValue
	if rawValue == nil {
		rawValue = tok.Value
	}

	// If same schema, pass through with minimal conversion
	if inputSchema == outputSchema {
		return convertReferences(rawValue, inputSchema, outputSchema)
	}

	// Handle schema conversion
	switch {
	case inputSchema == schema.Draft && outputSchema == schema.V2025_10:
		return convertDraftToV2025(tok, rawValue)
	case inputSchema == schema.V2025_10 && outputSchema == schema.Draft:
		return convertV2025ToDraft(rawValue)
	default:
		return convertReferences(rawValue, inputSchema, outputSchema)
	}
}

// convertDraftToV2025 converts Editor's Draft values to v2025_10 format.
func convertDraftToV2025(tok *token.Token, rawValue any) any {
	switch v := rawValue.(type) {
	case string:
		// Check if it's a reference
		if curlyBraceRefPattern.MatchString(v) {
			// Check if the entire value is a single reference
			if matched := curlyBraceRefPattern.FindStringSubmatch(v); matched != nil && matched[0] == v {
				// Full reference - convert to $ref
				return map[string]any{
					"$ref": common.ConvertTokenPathToJSONPointer(matched[1]),
				}
			}
			// Embedded reference - keep as-is (no standard for this)
			return v
		}

		// Check if it's a color and convert to structured format
		if tok.Type == "color" {
			return convertStringColorToStructured(v)
		}

		return v

	case map[string]any:
		return convertMapReferences(v, schema.Draft, schema.V2025_10)

	case []any:
		return convertArrayReferences(v, schema.Draft, schema.V2025_10)

	default:
		return v
	}
}

// convertV2025ToDraft converts v2025_10 values to Editor's Draft format.
func convertV2025ToDraft(rawValue any) any {
	switch v := rawValue.(type) {
	case string:
		// Check if it's a JSON pointer reference (starts with #/)
		if strings.HasPrefix(v, "#/") {
			tokenPath := common.ConvertJSONPointerToTokenPath(v)
			return "{" + tokenPath + "}"
		}
		return v

	case map[string]any:
		// Check if it's a $ref
		if ref, ok := v["$ref"].(string); ok {
			tokenPath := common.ConvertJSONPointerToTokenPath(ref)
			return "{" + tokenPath + "}"
		}

		// Check if it's a structured color value
		if _, hasColorSpace := v["colorSpace"].(string); hasColorSpace {
			return convertStructuredColorToString(v)
		}

		return convertMapReferences(v, schema.V2025_10, schema.Draft)

	case []any:
		return convertArrayReferences(v, schema.V2025_10, schema.Draft)

	default:
		return v
	}
}

// convertReferences converts references between schemas without changing value types.
func convertReferences(value any, inputSchema, outputSchema schema.Version) any {
	switch v := value.(type) {
	case string:
		return convertStringReferences(v, inputSchema, outputSchema)
	case map[string]any:
		return convertMapReferences(v, inputSchema, outputSchema)
	case []any:
		return convertArrayReferences(v, inputSchema, outputSchema)
	default:
		return v
	}
}

// convertStringReferences handles reference conversion in strings.
func convertStringReferences(s string, inputSchema, outputSchema schema.Version) string {
	if inputSchema == schema.Draft && outputSchema == schema.V2025_10 {
		// Convert {token.path} to {token/path} style - but keep string format
		// (Full reference conversion to $ref is handled in convertDraftToV2025)
		return s
	}
	return s
}

// convertMapReferences converts references within a map.
// Note: $ref conversion from V2025_10 to Draft is handled by convertV2025ToDraft.
func convertMapReferences(m map[string]any, inputSchema, outputSchema schema.Version) map[string]any {
	result := make(map[string]any)

	for k, v := range m {
		result[k] = convertReferences(v, inputSchema, outputSchema)
	}

	return result
}

// convertArrayReferences converts references within an array.
func convertArrayReferences(arr []any, inputSchema, outputSchema schema.Version) []any {
	result := make([]any, len(arr))
	for i, v := range arr {
		result[i] = convertReferences(v, inputSchema, outputSchema)
	}
	return result
}

// convertStringColorToStructured converts a string color to v2025_10 structured format.
func convertStringColorToStructured(colorStr string) any {
	c, err := csscolorparser.Parse(colorStr)
	if err != nil {
		// If parsing fails, return the original string
		return colorStr
	}

	// Use the Color struct fields directly (float64 0-1 range)
	result := map[string]any{
		"colorSpace": "srgb",
		"components": []any{c.R, c.G, c.B},
		"alpha":      c.A,
	}

	// Include hex for convenience
	if strings.HasPrefix(colorStr, "#") {
		result["hex"] = colorStr
	} else {
		result["hex"] = c.HexString()
	}

	return result
}

// convertStructuredColorToString converts a v2025_10 structured color to a string.
func convertStructuredColorToString(colorObj map[string]any) string {
	// If hex field is provided, use it
	if hex, ok := colorObj["hex"].(string); ok && hex != "" {
		return hex
	}

	colorSpace, _ := colorObj["colorSpace"].(string)
	componentsRaw, _ := colorObj["components"].([]any)
	alphaRaw := colorObj["alpha"]

	// Try to convert to CSS color() function
	if colorSpace != "" && len(componentsRaw) > 0 {
		var compStrs []string
		for _, comp := range componentsRaw {
			switch v := comp.(type) {
			case float64:
				compStrs = append(compStrs, fmt.Sprintf("%.4g", v))
			case string:
				compStrs = append(compStrs, v)
			}
		}

		// Handle alpha
		alpha := 1.0
		if a, ok := alphaRaw.(float64); ok {
			alpha = a
		}

		if alpha < 0.999 {
			return fmt.Sprintf("color(%s %s / %.4g)", colorSpace, strings.Join(compStrs, " "), alpha)
		}
		return fmt.Sprintf("color(%s %s)", colorSpace, strings.Join(compStrs, " "))
	}

	// Fallback - return empty if we can't convert
	return ""
}
