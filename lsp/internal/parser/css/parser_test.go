package css_test

import (
	"encoding/json"
	"flag"
	"os"
	"testing"

	"bennypowers.dev/asimonim/lsp/internal/parser/css"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update golden files")

// TestParseSimpleCSSVariable tests parsing a simple CSS custom property declaration
func TestParseSimpleCSSVariable(t *testing.T) {
	cssCode := `:root {
  --color-primary: #0000ff;
}`

	parser := css.AcquireParser()
	defer css.ReleaseParser(parser)
	result, err := parser.Parse(cssCode)
	require.NoError(t, err, "Parsing should not error")
	require.NotNil(t, result, "Parse result should not be nil")

	// Should find one variable declaration
	variables := result.Variables
	require.Len(t, variables, 1, "Should find one variable")

	variable := variables[0]
	assert.Equal(t, "--color-primary", variable.Name)
	assert.Equal(t, "#0000ff", variable.Value)
	assert.Equal(t, css.VariableDeclaration, variable.Type)

	// Check position information
	assert.Equal(t, uint32(1), variable.Range.Start.Line, "Variable should be on line 1 (0-indexed)")
	assert.Greater(t, variable.Range.Start.Character, uint32(0), "Variable should have character position")
}

// TestParseMultipleCSSVariables tests parsing multiple CSS custom properties
func TestParseMultipleCSSVariables(t *testing.T) {
	cssCode := `:root {
  --color-primary: #0000ff;
  --color-secondary: #ff0000;
  --spacing-small: 8px;
}`

	parser := css.AcquireParser()
	defer css.ReleaseParser(parser)
	result, err := parser.Parse(cssCode)
	require.NoError(t, err)

	variables := result.Variables
	require.Len(t, variables, 3, "Should find three variables")

	// Check each variable
	expectedVars := map[string]string{
		"--color-primary":   "#0000ff",
		"--color-secondary": "#ff0000",
		"--spacing-small":   "8px",
	}

	for _, v := range variables {
		expectedValue, ok := expectedVars[v.Name]
		require.True(t, ok, "Variable %s should be in expected list", v.Name)
		assert.Equal(t, expectedValue, v.Value, "Variable %s should have correct value", v.Name)
	}
}

// TestParseVarFunctionCall tests parsing var() function calls
func TestParseVarFunctionCall(t *testing.T) {
	cssCode := `.button {
  color: var(--color-primary);
}`

	parser := css.AcquireParser()
	defer css.ReleaseParser(parser)
	result, err := parser.Parse(cssCode)
	require.NoError(t, err)

	varCalls := result.VarCalls
	require.Len(t, varCalls, 1, "Should find one var() call")

	varCall := varCalls[0]
	assert.Equal(t, "--color-primary", varCall.TokenName)
	assert.Nil(t, varCall.Fallback, "Should have no fallback")
	assert.Equal(t, css.VarReference, varCall.Type)
}

// TestParseVarFunctionWithFallback tests parsing var() with fallback values
func TestParseVarFunctionWithFallback(t *testing.T) {
	cssCode := `.button {
  color: var(--color-primary, #000);
}`

	parser := css.AcquireParser()
	defer css.ReleaseParser(parser)
	result, err := parser.Parse(cssCode)
	require.NoError(t, err)

	varCalls := result.VarCalls
	require.Len(t, varCalls, 1, "Should find one var() call")

	varCall := varCalls[0]
	assert.Equal(t, "--color-primary", varCall.TokenName)
	require.NotNil(t, varCall.Fallback, "Should have fallback")
	assert.Equal(t, "#000", *varCall.Fallback)
}

// TestParseNestedVarCalls tests parsing nested var() calls (fallback contains var())
func TestParseNestedVarCalls(t *testing.T) {
	cssCode := `.button {
  color: var(--color-primary, var(--color-base));
}`

	parser := css.AcquireParser()
	defer css.ReleaseParser(parser)
	result, err := parser.Parse(cssCode)
	require.NoError(t, err)

	// Should find both var() calls
	varCalls := result.VarCalls
	assert.GreaterOrEqual(t, len(varCalls), 2, "Should find at least two var() calls")

	// First call should be --color-primary
	found := false
	for _, call := range varCalls {
		if call.TokenName == "--color-primary" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find var(--color-primary)")

	// Second call should be --color-base
	found = false
	for _, call := range varCalls {
		if call.TokenName == "--color-base" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find var(--color-base)")
}

// TestParseMixedContent tests parsing CSS with both declarations and var() calls
func TestParseMixedContent(t *testing.T) {
	cssCode := `:root {
  --color-primary: #0000ff;
}

.button {
  color: var(--color-primary, #000);
  background: var(--color-secondary);
}`

	parser := css.AcquireParser()
	defer css.ReleaseParser(parser)
	result, err := parser.Parse(cssCode)
	require.NoError(t, err)

	// Should find variable declarations
	assert.Len(t, result.Variables, 1, "Should find one variable declaration")
	assert.Equal(t, "--color-primary", result.Variables[0].Name)

	// Should find var() calls
	assert.Len(t, result.VarCalls, 2, "Should find two var() calls")
}

// TestParseInvalidCSS tests that invalid CSS is handled gracefully
func TestParseInvalidCSS(t *testing.T) {
	cssCode := `this is not valid css {{{`

	parser := css.AcquireParser()
	defer css.ReleaseParser(parser)
	result, err := parser.Parse(cssCode)

	// Parser should not crash, but may return an error or empty result
	// Tree-sitter is error-tolerant, so it might still parse partially
	if err != nil {
		t.Logf("Parser returned error (acceptable): %v", err)
	}
	if result != nil {
		t.Logf("Parser returned result with %d variables and %d var calls",
			len(result.Variables), len(result.VarCalls))
	}
}

// TestParseEmptyCSS tests parsing empty CSS
func TestParseEmptyCSS(t *testing.T) {
	cssCode := ``

	parser := css.AcquireParser()
	defer css.ReleaseParser(parser)
	result, err := parser.Parse(cssCode)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Variables, "Should find no variables")
	assert.Empty(t, result.VarCalls, "Should find no var() calls")
}

// TestParseVarFunctionWithCommaSeparatedFallback tests parsing var() with comma-separated fallback values
// This covers font-family lists with mixed quoted (with spaces) and unquoted identifiers
func TestParseVarFunctionWithCommaSeparatedFallback(t *testing.T) {
	cssCode := `.element {
  font-family: var(--font-family, FooFont, 'Bar Font', BazFont, QuxFont, sans-serif);
}`

	parser := css.AcquireParser()
	defer css.ReleaseParser(parser)
	result, err := parser.Parse(cssCode)
	require.NoError(t, err)

	varCalls := result.VarCalls
	require.Len(t, varCalls, 1, "Should find one var() call")

	varCall := varCalls[0]
	assert.Equal(t, "--font-family", varCall.TokenName)
	require.NotNil(t, varCall.Fallback, "Should have fallback")
	assert.Equal(t, "FooFont, 'Bar Font', BazFont, QuxFont, sans-serif", *varCall.Fallback)
}

// TestParseVarFunctionWithNestedCommasInFallback tests parsing var() with fallback containing nested commas (rgba, box-shadow)
func TestParseVarFunctionWithNestedCommasInFallback(t *testing.T) {
	cssCode := `.element {
  box-shadow: var(--shadow, 1px 2px rgba(0, 0, 0, 0.5));
}`

	parser := css.AcquireParser()
	defer css.ReleaseParser(parser)
	result, err := parser.Parse(cssCode)
	require.NoError(t, err)

	varCalls := result.VarCalls
	require.Len(t, varCalls, 1, "Should find one var() call")

	varCall := varCalls[0]
	assert.Equal(t, "--shadow", varCall.TokenName)
	require.NotNil(t, varCall.Fallback, "Should have fallback")
	assert.Equal(t, "1px 2px rgba(0, 0, 0, 0.5)", *varCall.Fallback)
}

// TestParserClose tests the Close method
func TestParserClose(t *testing.T) {
	parser := css.AcquireParser()
	// Close should not panic
	parser.Close()
}

// TestClosePool tests the ClosePool method drains the pool
func TestClosePool(t *testing.T) {
	// Put a couple parsers in the pool
	p1 := css.AcquireParser()
	p2 := css.AcquireParser()
	css.ReleaseParser(p1)
	css.ReleaseParser(p2)

	// Drain pool - should not panic
	css.ClosePool()

	// Pool should still work after draining (New func is restored)
	p3 := css.AcquireParser()
	defer css.ReleaseParser(p3)
	result, err := p3.Parse(`.btn { color: var(--c); }`)
	require.NoError(t, err)
	assert.NotEmpty(t, result.VarCalls)
}

// TestReleaseNilParser tests that releasing a nil parser does not panic
func TestReleaseNilParser(t *testing.T) {
	// Should not panic
	css.ReleaseParser(nil)
}

// TestParseNonCustomPropertyDeclaration tests that regular CSS properties are skipped
func TestParseNonCustomPropertyDeclaration(t *testing.T) {
	cssCode := `.button {
  color: red;
  font-size: 16px;
}`

	parser := css.AcquireParser()
	defer css.ReleaseParser(parser)
	result, err := parser.Parse(cssCode)
	require.NoError(t, err)

	// Should find no variables (only custom properties count)
	assert.Empty(t, result.Variables, "Regular CSS properties should not be counted as variables")
	assert.Empty(t, result.VarCalls, "No var() calls in this CSS")
}

// TestParseNonVarFunctionCall tests that non-var() function calls are ignored
func TestParseNonVarFunctionCall(t *testing.T) {
	cssCode := `.button {
  color: rgb(255, 0, 0);
  background: linear-gradient(to right, red, blue);
  transform: translateX(10px);
}`

	parser := css.AcquireParser()
	defer css.ReleaseParser(parser)
	result, err := parser.Parse(cssCode)
	require.NoError(t, err)

	// Should find no var() calls (rgb, linear-gradient, translateX are not var)
	assert.Empty(t, result.VarCalls, "Non-var() function calls should be ignored")
}

// TestParseVarWithoutArguments tests var() with empty arguments
func TestParseVarWithoutArguments(t *testing.T) {
	// This is technically invalid CSS but tree-sitter is tolerant
	cssCode := `.button {
  color: var();
}`

	parser := css.AcquireParser()
	defer css.ReleaseParser(parser)
	result, err := parser.Parse(cssCode)
	require.NoError(t, err)

	// var() with no arguments should be skipped (tokenName will be "")
	assert.Empty(t, result.VarCalls, "var() with no arguments should produce no var calls")
}

// TestParseDeclarationWithNoValue tests a custom property with no value node
func TestParseDeclarationWithNoValue(t *testing.T) {
	// Custom property declaration without a recognized value node
	cssCode := `:root {
  --empty-var: ;
}`

	parser := css.AcquireParser()
	defer css.ReleaseParser(parser)
	result, err := parser.Parse(cssCode)
	require.NoError(t, err)

	// Should still find the variable declaration even with empty value
	found := false
	for _, v := range result.Variables {
		if v.Name == "--empty-var" {
			found = true
			assert.Equal(t, "", v.Value, "Empty value should be empty string")
		}
	}
	assert.True(t, found, "Should find --empty-var declaration")
}

// TestParseMultilineDeclaration tests CSS with multi-line content
func TestParseMultilineDeclaration(t *testing.T) {
	cssCode := `:root {
  --color-primary: #0000ff;
}

.button {
  color: var(--color-primary);
}

.card {
  background: var(--color-primary);
  border: 1px solid var(--color-primary);
}`

	parser := css.AcquireParser()
	defer css.ReleaseParser(parser)
	result, err := parser.Parse(cssCode)
	require.NoError(t, err)

	assert.Len(t, result.Variables, 1)
	assert.Len(t, result.VarCalls, 3, "Should find 3 var() calls across multiple rules")
}

// TestParseVarInsideCSSFunctions tests var() calls nested inside CSS functions like calc(), min(), max(), clamp()
func TestParseVarInsideCSSFunctions(t *testing.T) {
	source, err := os.ReadFile("testdata/var-in-functions.css")
	require.NoError(t, err)

	parser := css.AcquireParser()
	defer css.ReleaseParser(parser)

	result, err := parser.Parse(string(source))
	require.NoError(t, err)
	require.NotNil(t, result)

	golden := "testdata/golden/var-in-functions.json"

	if *update {
		data, marshalErr := json.MarshalIndent(result, "", "  ")
		require.NoError(t, marshalErr)
		writeErr := os.WriteFile(golden, append(data, '\n'), 0o644)
		require.NoError(t, writeErr)
		return
	}

	goldenData, err := os.ReadFile(golden)
	require.NoError(t, err)

	var expected css.ParseResult
	err = json.Unmarshal(goldenData, &expected)
	require.NoError(t, err)

	require.Equal(t, len(expected.VarCalls), len(result.VarCalls), "var call count")
	assert.Empty(t, result.Variables, "fixture has no custom property declarations")

	for i, vc := range result.VarCalls {
		assert.Equal(t, *expected.VarCalls[i], *vc, "var call %d", i)
	}
}

// TestParseVarFunctionWithComplexFontFallback tests various font-family patterns
func TestParseVarFunctionWithComplexFontFallback(t *testing.T) {
	testCases := []struct {
		name             string
		css              string
		expectedToken    string
		expectedFallback string
	}{
		{
			name:             "Simple font list",
			css:              `font-family: var(--font, FooFont, sans-serif);`,
			expectedToken:    "--font",
			expectedFallback: "FooFont, sans-serif",
		},
		{
			name:             "Quoted font with spaces in list",
			css:              `font-family: var(--font, 'My Font', BarFont, sans-serif);`,
			expectedToken:    "--font",
			expectedFallback: "'My Font', BarFont, sans-serif",
		},
		{
			name:             "Single font fallback",
			css:              `font-family: var(--font, monospace);`,
			expectedToken:    "--font",
			expectedFallback: "monospace",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cssCode := `.element { ` + tc.css + ` }`

			parser := css.AcquireParser()
			defer css.ReleaseParser(parser)
			result, err := parser.Parse(cssCode)
			require.NoError(t, err)

			varCalls := result.VarCalls
			require.Len(t, varCalls, 1, "Should find one var() call")

			varCall := varCalls[0]
			assert.Equal(t, tc.expectedToken, varCall.TokenName)
			require.NotNil(t, varCall.Fallback, "Should have fallback")
			assert.Equal(t, tc.expectedFallback, *varCall.Fallback)
		})
	}
}
