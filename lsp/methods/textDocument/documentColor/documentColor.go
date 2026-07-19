package documentcolor

import (
	"bennypowers.dev/asimonim/lsp/internal/log"
	"fmt"
	"strings"

	"bennypowers.dev/asimonim/lsp/internal/parser"
	"bennypowers.dev/asimonim/lsp/types"
	"github.com/mazznoer/csscolorparser"
	"go.lsp.dev/protocol"
)

// DocumentColor handles the textDocument/documentColor request
func DocumentColor(req *types.RequestContext, params *protocol.DocumentColorParams) ([]protocol.ColorInformation, error) {
	uri := string(params.TextDocument.URI)

	log.Info("DocumentColor requested: %s", uri)

	// Get document
	doc := req.Server.Document(uri)
	if doc == nil {
		return nil, nil
	}

	// Only process CSS-supported files
	if !parser.IsCSSSupportedLanguage(doc.LanguageID()) {
		return nil, nil
	}

	// Parse CSS to find var() calls
	result, err := parser.ParseCSSFromDocument(doc.Content(), doc.LanguageID())
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSS: %w", err)
	}
	if result == nil {
		return nil, nil
	}

	var colors []protocol.ColorInformation
	var parseErrors []error

	// Find all var() calls that reference color tokens
	for _, varCall := range result.VarCalls {
		// Look up the token
		token := req.Server.Token(varCall.TokenName)
		if token == nil {
			continue
		}

		// Only process color tokens
		if token.Type != "color" {
			continue
		}

		// Parse the color value
		color, err := parseColor(token.Value)
		if err != nil {
			log.Info("Failed to parse color %s: %v", token.Value, err)
			parseErrors = append(parseErrors, fmt.Errorf("failed to parse color token %s (value: %s): %w", varCall.TokenName, token.Value, err))
			continue
		}

		colors = append(colors, protocol.ColorInformation{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      varCall.Range.Start.Line,
					Character: varCall.Range.Start.Character,
				},
				End: protocol.Position{
					Line:      varCall.Range.End.Line,
					Character: varCall.Range.End.Character,
				},
			},
			Color: *color,
		})
	}

	// Also check variable declarations
	for _, variable := range result.Variables {
		// Look up the token
		token := req.Server.Token(variable.Name)
		if token == nil {
			continue
		}

		// Only process color tokens
		if token.Type != "color" {
			continue
		}

		// Parse the color value
		color, err := parseColor(token.Value)
		if err != nil {
			log.Info("Failed to parse color %s: %v", token.Value, err)
			parseErrors = append(parseErrors, fmt.Errorf("failed to parse color token %s (value: %s): %w", variable.Name, token.Value, err))
			continue
		}

		colors = append(colors, protocol.ColorInformation{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      variable.Range.Start.Line,
					Character: variable.Range.Start.Character,
				},
				End: protocol.Position{
					Line:      variable.Range.End.Line,
					Character: variable.Range.End.Character,
				},
			},
			Color: *color,
		})
	}

	log.Info("Found %d colors", len(colors))

	// Add parse errors as warnings
	// Don't fail the operation - we can still return partial results
	// Middleware will log these warnings after successful completion
	for _, err := range parseErrors {
		req.AddWarning(err)
	}

	return colors, nil
}

// ColorPresentation handles the textDocument/colorPresentation request
// Returns token names that have the same color value as the requested color
func ColorPresentation(req *types.RequestContext, params *protocol.ColorPresentationParams) ([]protocol.ColorPresentation, error) {
	uri := string(params.TextDocument.URI)
	color := params.Color

	log.Info("ColorPresentation requested: %s", uri)

	// Convert protocol.Color to csscolorparser.Color for comparison
	requestedColor := csscolorparser.Color{
		R: float64(color.Red),
		G: float64(color.Green),
		B: float64(color.Blue),
		A: float64(color.Alpha),
	}
	requestedHex := requestedColor.HexString() // Includes alpha if < 1.0

	var presentations []protocol.ColorPresentation
	var parseErrors []error

	// Find all tokens with matching color values
	for _, token := range req.Server.TokenManager().GetAll() {
		// Only process color tokens
		if token.Type != "color" {
			continue
		}

		// Parse the token's color value
		tokenColor, err := parseColor(token.Value)
		if err != nil {
			log.Info("Failed to parse color token %s (value: %s): %v", token.Name, token.Value, err)
			parseErrors = append(parseErrors, fmt.Errorf("failed to parse color token %s (value: %s): %w", token.Name, token.Value, err))
			continue
		}

		// Convert to csscolorparser.Color for comparison
		c := csscolorparser.Color{
			R: float64(tokenColor.Red),
			G: float64(tokenColor.Green),
			B: float64(tokenColor.Blue),
			A: float64(tokenColor.Alpha),
		}

		// Compare hex strings (includes alpha channel)
		if c.HexString() == requestedHex {
			presentations = append(presentations, protocol.ColorPresentation{
				Label: token.Name,
			})
		}
	}

	log.Info("Found %d matching color tokens", len(presentations))

	// Add parse errors as warnings
	// Don't fail the operation - we can still return partial results
	// Middleware will log these warnings after successful completion
	for _, err := range parseErrors {
		req.AddWarning(err)
	}

	return presentations, nil
}

const maxColorParseDepth = 10

// parseColor parses a color string (hex, rgb, rgba, hsl, hsla, etc.) and returns a protocol.Color
func parseColor(value string) (*protocol.Color, error) {
	return parseColorDepth(value, 0)
}

func parseColorDepth(value string, depth int) (*protocol.Color, error) {
	if depth > maxColorParseDepth {
		return nil, fmt.Errorf("exceeded max nesting depth parsing color: %s", value)
	}

	value = strings.TrimSpace(value)

	if strings.HasPrefix(value, "light-dark(") {
		return parseLightDarkColor(value, depth)
	}

	if strings.HasPrefix(value, "var(") {
		if fallback := extractVarFallback(value); fallback != "" {
			return parseColorDepth(fallback, depth+1)
		}
		return nil, fmt.Errorf("unsupported color format: %s", value)
	}

	return parseCSSColor(value)
}

// parseLightDarkColor extracts and parses the light-mode color from light-dark().
// For values like light-dark(var(--x, #fff), var(--y, #000)), extracts the
// first argument's fallback color.
func parseLightDarkColor(value string, depth int) (*protocol.Color, error) {
	inner := strings.TrimPrefix(value, "light-dark(")
	inner = strings.TrimSuffix(inner, ")")

	lightArg := extractFirstBalancedArg(inner)
	if lightArg == "" {
		return nil, fmt.Errorf("unsupported light-dark format: %s", value)
	}

	lightArg = strings.TrimSpace(lightArg)

	return parseColorDepth(lightArg, depth+1)
}

// extractFirstBalancedArg extracts the first comma-separated argument,
// respecting nested parentheses.
func extractFirstBalancedArg(s string) string {
	depth := 0
	for i, ch := range s {
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				return s[:i]
			}
		}
	}
	return s
}

// extractVarFallback extracts the fallback value from a var() expression.
// For var(--name, fallback), returns "fallback".
func extractVarFallback(varExpr string) string {
	inner := strings.TrimPrefix(varExpr, "var(")
	inner = strings.TrimSuffix(inner, ")")

	// Find the comma after the custom property name, respecting nesting
	depth := 0
	for i, ch := range inner {
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				return strings.TrimSpace(inner[i+1:])
			}
		}
	}
	return ""
}

func parseCSSColor(value string) (*protocol.Color, error) {
	parsed, err := csscolorparser.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("unsupported color format: %s", value)
	}

	return &protocol.Color{
		Red:   parsed.R,
		Green: parsed.G,
		Blue:  parsed.B,
		Alpha: parsed.A,
	}, nil
}
