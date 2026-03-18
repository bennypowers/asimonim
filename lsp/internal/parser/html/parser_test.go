package html_test

import (
	"encoding/json"
	"flag"
	"os"
	"testing"

	"bennypowers.dev/asimonim/lsp/internal/parser/css"
	"bennypowers.dev/asimonim/lsp/internal/parser/html"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update golden files")

func TestParseCSSRegions(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		wantTags int
		wantAttr int
	}{
		{
			name:     "style tag",
			fixture:  "testdata/style-tag.html",
			wantTags: 1,
			wantAttr: 0,
		},
		{
			name:     "style attributes",
			fixture:  "testdata/style-attribute.html",
			wantTags: 0,
			wantAttr: 2,
		},
		{
			name:     "multiple styles",
			fixture:  "testdata/multiple-styles.html",
			wantTags: 2,
			wantAttr: 2,
		},
		{
			name:     "no CSS",
			fixture:  "testdata/no-css.html",
			wantTags: 0,
			wantAttr: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := os.ReadFile(tt.fixture)
			require.NoError(t, err)

			parser := html.AcquireParser()
			defer html.ReleaseParser(parser)

			regions := parser.ParseCSSRegions(string(source))

			tags := 0
			attrs := 0
			for _, r := range regions {
				switch r.Type {
				case html.StyleTag:
					tags++
				case html.StyleAttribute:
					attrs++
				}
			}

			assert.Equal(t, tt.wantTags, tags, "style tag count")
			assert.Equal(t, tt.wantAttr, attrs, "style attribute count")
		})
	}
}

func TestParseCSS(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		golden  string
	}{
		{
			name:    "style tag",
			fixture: "testdata/style-tag.html",
			golden:  "testdata/golden/style-tag.json",
		},
		{
			name:    "style attribute",
			fixture: "testdata/style-attribute.html",
			golden:  "testdata/golden/style-attribute.json",
		},
		{
			name:    "multiple styles",
			fixture: "testdata/multiple-styles.html",
			golden:  "testdata/golden/multiple-styles.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := os.ReadFile(tt.fixture)
			require.NoError(t, err)

			parser := html.AcquireParser()
			defer html.ReleaseParser(parser)

			result, err := parser.ParseCSS(string(source))
			require.NoError(t, err)
			require.NotNil(t, result)

			if *update {
				data, marshalErr := json.MarshalIndent(result, "", "  ")
				require.NoError(t, marshalErr)
				writeErr := os.WriteFile(tt.golden, append(data, '\n'), 0o644)
				require.NoError(t, writeErr)
				return
			}

			golden, err := os.ReadFile(tt.golden)
			require.NoError(t, err)

			var expected css.ParseResult
			err = json.Unmarshal(golden, &expected)
			require.NoError(t, err)

			require.Equal(t, len(expected.Variables), len(result.Variables), "variable count")
			require.Equal(t, len(expected.VarCalls), len(result.VarCalls), "var call count")

			for i, v := range result.Variables {
				assert.Equal(t, expected.Variables[i].Name, v.Name, "variable %d name", i)
				assert.Equal(t, expected.Variables[i].Range, v.Range, "variable %d range", i)
			}

			for i, vc := range result.VarCalls {
				assert.Equal(t, expected.VarCalls[i].TokenName, vc.TokenName, "var call %d token name", i)
				assert.Equal(t, expected.VarCalls[i].Range, vc.Range, "var call %d range", i)
			}
		})
	}
}

func TestParseCSSNoCSS(t *testing.T) {
	source, err := os.ReadFile("testdata/no-css.html")
	require.NoError(t, err)

	parser := html.AcquireParser()
	defer html.ReleaseParser(parser)

	result, err := parser.ParseCSS(string(source))
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Variables)
	assert.Empty(t, result.VarCalls)
}

func TestParseCSSEmptyStyleTag(t *testing.T) {
	source := `<style></style>`

	parser := html.AcquireParser()
	defer html.ReleaseParser(parser)

	result, err := parser.ParseCSS(source)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Variables)
	assert.Empty(t, result.VarCalls)
}

func TestParseCSSEmptyStyleAttribute(t *testing.T) {
	source := `<div style=""></div>`

	parser := html.AcquireParser()
	defer html.ReleaseParser(parser)

	result, err := parser.ParseCSS(source)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Variables)
	assert.Empty(t, result.VarCalls)
}

func TestStyleTagPositionMapping(t *testing.T) {
	// Verify position mapping accuracy for style tags
	source := `<html>
<head>
<style>
.button {
  color: var(--color-primary);
}
</style>
</head>
</html>`

	parser := html.AcquireParser()
	defer html.ReleaseParser(parser)

	result, err := parser.ParseCSS(source)
	require.NoError(t, err)
	require.Len(t, result.VarCalls, 1)

	vc := result.VarCalls[0]
	assert.Equal(t, "--color-primary", vc.TokenName)
	// var(--color-primary) is on line 4 (0-indexed) in the HTML document
	// "  color: var(--color-primary);" — var starts at column 9
	assert.Equal(t, uint32(4), vc.Range.Start.Line, "var call should be on line 4")
	assert.Equal(t, uint32(9), vc.Range.Start.Character, "var call should start at char 9")
}

func TestStyleAttributePositionMapping(t *testing.T) {
	// Verify position mapping accuracy for style attributes
	source := `<div style="color: var(--text-color)">Hello</div>`

	parser := html.AcquireParser()
	defer html.ReleaseParser(parser)

	result, err := parser.ParseCSS(source)
	require.NoError(t, err)
	require.Len(t, result.VarCalls, 1)

	vc := result.VarCalls[0]
	assert.Equal(t, "--text-color", vc.TokenName)
	// style="color: var(--text-color)" — attribute value starts at col 12
	// "color: var(--text-color)" — var() starts at offset 7 within the attribute value
	// So in the HTML document, var() starts at col 12 + 7 = 19
	assert.Equal(t, uint32(0), vc.Range.Start.Line, "var call should be on line 0")
	assert.Equal(t, uint32(19), vc.Range.Start.Character, "var call should start at char 19")
}

func TestMultilineStyleTagPositionMapping(t *testing.T) {
	// Verify that CSS on lines after the first line of a style tag
	// gets only line offset (not column offset)
	source := `<html>
<head>
  <style>
    :root {
      --color-primary: #00f;
    }
    .card {
      color: var(--color-primary);
    }
  </style>
</head>
</html>`

	parser := html.AcquireParser()
	defer html.ReleaseParser(parser)

	result, err := parser.ParseCSS(source)
	require.NoError(t, err)
	require.Len(t, result.VarCalls, 1)

	vc := result.VarCalls[0]
	assert.Equal(t, "--color-primary", vc.TokenName)
	// The var() is on line 7 (0-indexed), and its column should match
	// the indentation in the CSS content itself (not offset by style tag indent)
	assert.Equal(t, uint32(7), vc.Range.Start.Line)
	assert.Greater(t, vc.Range.Start.Character, uint32(0))
}

func TestMultilineStyleAttributePositionMapping(t *testing.T) {
	// Style attribute where CSS parsing produces positions on line 0
	source := `<div style="color: var(--a); background: var(--b, #fff)">x</div>`

	parser := html.AcquireParser()
	defer html.ReleaseParser(parser)

	result, err := parser.ParseCSS(source)
	require.NoError(t, err)
	require.Len(t, result.VarCalls, 2)

	assert.Equal(t, "--a", result.VarCalls[0].TokenName)
	assert.Equal(t, "--b", result.VarCalls[1].TokenName)
	// Both on line 0
	assert.Equal(t, uint32(0), result.VarCalls[0].Range.Start.Line)
	assert.Equal(t, uint32(0), result.VarCalls[1].Range.Start.Line)
	// Second var call should be after the first
	assert.Greater(t, result.VarCalls[1].Range.Start.Character, result.VarCalls[0].Range.Start.Character)
}

func TestAdjustAttributePositionUnderflow(t *testing.T) {
	// When the wrapped CSS "x{...}" produces a position with Character < 2,
	// the underflow guard should clamp to region.StartCol.
	// Use a short property name so positions near the start are tested.
	source := `<div style="a:var(--x)">y</div>`

	parser := html.AcquireParser()
	defer html.ReleaseParser(parser)

	result, err := parser.ParseCSS(source)
	require.NoError(t, err)
	require.Len(t, result.VarCalls, 1)

	vc := result.VarCalls[0]
	assert.Equal(t, "--x", vc.TokenName)
	assert.Equal(t, uint32(0), vc.Range.Start.Line)
	// "a:var(--x)" — var starts at offset 2 in the attribute value
	// attribute value starts at col 12 in the HTML → 12 + 2 = 14
	assert.Equal(t, uint32(14), vc.Range.Start.Character)
}

func TestClosePool(t *testing.T) {
	// Exercise ClosePool — should not panic
	// First, put a parser into the pool
	p := html.AcquireParser()
	html.ReleaseParser(p)
	html.ClosePool()
	// Pool is drained; acquiring again should still work (creates new parser)
	p2 := html.AcquireParser()
	defer html.ReleaseParser(p2)
	regions := p2.ParseCSSRegions(`<style>.a{}</style>`)
	assert.Len(t, regions, 1)
}

// Twig Template Tests
// ============================================================================
// Twig syntax ({% %}, {{ }}) is valid text content in HTML, so tree-sitter-html
// parses Twig templates correctly without a dedicated grammar. These tests verify
// that all edge cases work: blocks, control flow, interpolation inside CSS, etc.

func TestTwigParseCSSRegions(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		wantTags int
		wantAttr int
	}{
		{
			// Drupal theme: Twig blocks, loops, filters, multiple style blocks,
			// style attributes interleaved with Twig variables
			name:     "drupal theme",
			fixture:  "testdata/drupal-theme.html.twig",
			wantTags: 2,
			wantAttr: 2,
		},
		{
			// Twig interpolation inside style tags and attributes,
			// Twig conditionals wrapping CSS rules, Twig filters
			name:     "interpolated styles",
			fixture:  "testdata/twig-interpolated-styles.html.twig",
			wantTags: 2,
			wantAttr: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := os.ReadFile(tt.fixture)
			require.NoError(t, err)

			p := html.AcquireParser()
			defer html.ReleaseParser(p)

			regions := p.ParseCSSRegions(string(source))

			tags := 0
			attrs := 0
			for _, r := range regions {
				switch r.Type {
				case html.StyleTag:
					tags++
				case html.StyleAttribute:
					attrs++
				}
			}

			assert.Equal(t, tt.wantTags, tags, "style tag count")
			assert.Equal(t, tt.wantAttr, attrs, "style attribute count")
		})
	}
}

func TestTwigParseCSS_Golden(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		golden  string
	}{
		{
			name:    "drupal theme",
			fixture: "testdata/drupal-theme.html.twig",
			golden:  "testdata/golden/drupal-theme.json",
		},
		{
			name:    "interpolated styles",
			fixture: "testdata/twig-interpolated-styles.html.twig",
			golden:  "testdata/golden/twig-interpolated-styles.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := os.ReadFile(tt.fixture)
			require.NoError(t, err)

			p := html.AcquireParser()
			defer html.ReleaseParser(p)

			result, err := p.ParseCSS(string(source))
			require.NoError(t, err)
			require.NotNil(t, result)

			if *update {
				data, marshalErr := json.MarshalIndent(result, "", "  ")
				require.NoError(t, marshalErr)
				writeErr := os.WriteFile(tt.golden, append(data, '\n'), 0o644)
				require.NoError(t, writeErr)
				return
			}

			golden, readErr := os.ReadFile(tt.golden)
			require.NoError(t, readErr)

			var expected css.ParseResult
			err = json.Unmarshal(golden, &expected)
			require.NoError(t, err)

			assert.Equal(t, expected.Variables, result.Variables, "variables")
			assert.Equal(t, expected.VarCalls, result.VarCalls, "var calls")
		})
	}
}

func TestTwigStyleTagPositionMapping(t *testing.T) {
	// Twig block before style tag - positions should be correct
	source := `{% extends "base.html.twig" %}
<style>
.card {
  color: var(--color-primary);
}
</style>`

	p := html.AcquireParser()
	defer html.ReleaseParser(p)

	result, err := p.ParseCSS(source)
	require.NoError(t, err)
	require.Len(t, result.VarCalls, 1)

	vc := result.VarCalls[0]
	assert.Equal(t, "--color-primary", vc.TokenName)
	// "  color: var(--color-primary);" on line 3 (0-indexed), var at col 9
	assert.Equal(t, uint32(3), vc.Range.Start.Line)
	assert.Equal(t, uint32(9), vc.Range.Start.Character)
}

func TestTwigStyleAttributePositionMapping(t *testing.T) {
	// Twig variable in HTML, style attribute on same element
	source := `<h1>{{ title }}</h1>
<div style="color: var(--text-color)">{{ body }}</div>`

	p := html.AcquireParser()
	defer html.ReleaseParser(p)

	result, err := p.ParseCSS(source)
	require.NoError(t, err)
	require.Len(t, result.VarCalls, 1)

	vc := result.VarCalls[0]
	assert.Equal(t, "--text-color", vc.TokenName)
	// style="color: var(--text-color)" on line 1, attr value at col 12, var at +7 = 19
	assert.Equal(t, uint32(1), vc.Range.Start.Line)
	assert.Equal(t, uint32(19), vc.Range.Start.Character)
}

func TestTwigInterleavedBlocks(t *testing.T) {
	// Twig blocks interleaved with style tags
	source := `{% block header %}
<style>
.header { background: var(--bg-header); }
</style>
{% endblock %}
{% block footer %}
<style>
.footer { color: var(--color-footer); }
</style>
{% endblock %}`

	p := html.AcquireParser()
	defer html.ReleaseParser(p)

	result, err := p.ParseCSS(source)
	require.NoError(t, err)
	require.Len(t, result.VarCalls, 2)

	assert.Equal(t, "--bg-header", result.VarCalls[0].TokenName)
	assert.Equal(t, uint32(2), result.VarCalls[0].Range.Start.Line)

	assert.Equal(t, "--color-footer", result.VarCalls[1].TokenName)
	assert.Equal(t, uint32(7), result.VarCalls[1].Range.Start.Line)
}

func TestTwigInterpolationInsideStyle(t *testing.T) {
	// Twig {{ }} inside a style tag - HTML parser treats it as text,
	// var() calls around the interpolation should still be extracted
	source := `<style>
:root {
  --brand: {{ brand_color }};
}
.card {
  color: var(--brand);
  background: var(--bg-card);
}
</style>`

	p := html.AcquireParser()
	defer html.ReleaseParser(p)

	result, err := p.ParseCSS(source)
	require.NoError(t, err)

	varNames := make([]string, len(result.VarCalls))
	for i, vc := range result.VarCalls {
		varNames[i] = vc.TokenName
	}
	assert.ElementsMatch(t, []string{"--brand", "--bg-card"}, varNames)
}

func TestTwigConditionalInsideStyle(t *testing.T) {
	// Twig {% if %} wrapping CSS rules inside a style tag
	source := `<style>
.base { color: var(--base-color); }
{% if has_dark_mode %}
.dark { background: var(--dark-bg); }
{% endif %}
</style>`

	p := html.AcquireParser()
	defer html.ReleaseParser(p)

	result, err := p.ParseCSS(source)
	require.NoError(t, err)

	varNames := make([]string, len(result.VarCalls))
	for i, vc := range result.VarCalls {
		varNames[i] = vc.TokenName
	}
	assert.ElementsMatch(t, []string{"--base-color", "--dark-bg"}, varNames)
}

func TestTwigForLoopInsideStyle(t *testing.T) {
	// Twig {% for %} generating CSS rules
	source := `<style>
{% for color in colors %}
.text-{{ color.name }} {
  color: var(--color-{{ color.name }});
}
{% endfor %}
.fallback { color: var(--color-default); }
</style>`

	p := html.AcquireParser()
	defer html.ReleaseParser(p)

	result, err := p.ParseCSS(source)
	require.NoError(t, err)

	// Only --color-default is a complete var() call; the Twig-interpolated
	// var(--color-{{ color.name }}) is not valid CSS, so CSS parser may or
	// may not extract it. The important thing is --color-default is found.
	varNames := make([]string, len(result.VarCalls))
	for i, vc := range result.VarCalls {
		varNames[i] = vc.TokenName
	}
	assert.Contains(t, varNames, "--color-default")
}

func TestTwigMacroWithStyles(t *testing.T) {
	// Twig macros alongside style blocks
	source := `{% macro card(title, body) %}
<div class="card" style="padding: var(--card-padding)">
  <h3>{{ title }}</h3>
  <p>{{ body }}</p>
</div>
{% endmacro %}
<style>
.card { border: 1px solid var(--card-border); }
</style>`

	p := html.AcquireParser()
	defer html.ReleaseParser(p)

	result, err := p.ParseCSS(source)
	require.NoError(t, err)

	varNames := make([]string, len(result.VarCalls))
	for i, vc := range result.VarCalls {
		varNames[i] = vc.TokenName
	}
	assert.ElementsMatch(t, []string{"--card-padding", "--card-border"}, varNames)
}

func TestTwigNoStyles(t *testing.T) {
	// Twig template with no CSS at all
	source := `{% extends "base.html.twig" %}
{% block content %}
  <h1>{{ title }}</h1>
  <p>{{ body|raw }}</p>
{% endblock %}`

	p := html.AcquireParser()
	defer html.ReleaseParser(p)

	result, err := p.ParseCSS(source)
	require.NoError(t, err)
	assert.Empty(t, result.Variables)
	assert.Empty(t, result.VarCalls)
}

func TestTwigEmptyStyleTag(t *testing.T) {
	source := `{% block styles %}<style></style>{% endblock %}`

	p := html.AcquireParser()
	defer html.ReleaseParser(p)

	result, err := p.ParseCSS(source)
	require.NoError(t, err)
	assert.Empty(t, result.Variables)
	assert.Empty(t, result.VarCalls)
}

func TestStyleTagWithVariableDeclaration(t *testing.T) {
	source := `<style>
:root {
  --my-color: blue;
}
</style>`

	parser := html.AcquireParser()
	defer html.ReleaseParser(parser)

	result, err := parser.ParseCSS(source)
	require.NoError(t, err)
	require.Len(t, result.Variables, 1)

	v := result.Variables[0]
	assert.Equal(t, "--my-color", v.Name)
	assert.Equal(t, "blue", v.Value)
}
