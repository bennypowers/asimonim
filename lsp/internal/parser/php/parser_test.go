package php_test

import (
	"encoding/json"
	"flag"
	"os"
	"testing"

	"bennypowers.dev/asimonim/lsp/internal/parser/css"
	"bennypowers.dev/asimonim/lsp/internal/parser/html"
	"bennypowers.dev/asimonim/lsp/internal/parser/php"
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
			// WordPress theme with multiple style blocks and style attributes,
			// PHP blocks between HTML elements
			name:     "wordpress theme",
			fixture:  "testdata/wordpress-theme.php",
			wantTags: 2,
			wantAttr: 2,
		},
		{
			// PHP interpolation inside <style> tags and style attributes,
			// PHP conditionals wrapping CSS rules, short echo syntax <?= ?>
			name:     "interpolated styles",
			fixture:  "testdata/php-interpolated-styles.php",
			wantTags: 2,
			wantAttr: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := os.ReadFile(tt.fixture)
			require.NoError(t, err)

			p := php.AcquireParser()
			defer php.ReleaseParser(p)

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

func TestParseCSS_Golden(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		golden  string
	}{
		{
			name:    "wordpress theme",
			fixture: "testdata/wordpress-theme.php",
			golden:  "testdata/golden/wordpress-theme.json",
		},
		{
			name:    "interpolated styles",
			fixture: "testdata/php-interpolated-styles.php",
			golden:  "testdata/golden/php-interpolated-styles.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := os.ReadFile(tt.fixture)
			require.NoError(t, err)

			p := php.AcquireParser()
			defer php.ReleaseParser(p)

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

func TestStyleTagPositionMapping(t *testing.T) {
	// PHP with style tag - verify positions account for PHP blocks
	source := `<?php $x = 1; ?>
<style>
.card {
  color: var(--color-primary);
}
</style>`

	p := php.AcquireParser()
	defer php.ReleaseParser(p)

	result, err := p.ParseCSS(source)
	require.NoError(t, err)
	require.Len(t, result.VarCalls, 1)

	vc := result.VarCalls[0]
	assert.Equal(t, "--color-primary", vc.TokenName)
	// var(--color-primary) is on line 3 (0-indexed)
	// "  color: var(--color-primary);" - var starts at column 9
	assert.Equal(t, uint32(3), vc.Range.Start.Line, "var call should be on line 3")
	assert.Equal(t, uint32(9), vc.Range.Start.Character, "var call should start at char 9")
}

func TestStyleAttributePositionMapping(t *testing.T) {
	// PHP with style attribute - positions should be correct
	source := `<?php $title = "Hello"; ?>
<div style="color: var(--text-color)"><?php echo $title; ?></div>`

	p := php.AcquireParser()
	defer php.ReleaseParser(p)

	result, err := p.ParseCSS(source)
	require.NoError(t, err)
	require.Len(t, result.VarCalls, 1)

	vc := result.VarCalls[0]
	assert.Equal(t, "--text-color", vc.TokenName)
	// style="color: var(--text-color)" on line 1
	// attribute value starts at col 12, var() at offset 7 within value -> col 19
	assert.Equal(t, uint32(1), vc.Range.Start.Line, "var call should be on line 1")
	assert.Equal(t, uint32(19), vc.Range.Start.Character, "var call should start at char 19")
}

func TestInterleavedPHPBlocks(t *testing.T) {
	// PHP blocks interleaved between style tags - both should be extracted
	source := `<?php get_header(); ?>
<style>
.header { background: var(--bg-header); }
</style>
<?php echo "content"; ?>
<style>
.footer { color: var(--color-footer); }
</style>
<?php get_footer(); ?>`

	p := php.AcquireParser()
	defer php.ReleaseParser(p)

	result, err := p.ParseCSS(source)
	require.NoError(t, err)
	require.Len(t, result.VarCalls, 2)

	// --bg-header from first style block
	assert.Equal(t, "--bg-header", result.VarCalls[0].TokenName)
	assert.Equal(t, uint32(2), result.VarCalls[0].Range.Start.Line)

	// --color-footer from second style block
	assert.Equal(t, "--color-footer", result.VarCalls[1].TokenName)
	assert.Equal(t, uint32(6), result.VarCalls[1].Range.Start.Line)
}

func TestInterpolatedStyles(t *testing.T) {
	// PHP interpolation inside <style> tags is common in WordPress themes.
	// tree-sitter-php correctly identifies HTML text nodes, allowing the
	// HTML parser to extract CSS regions with var() calls intact.
	source, err := os.ReadFile("testdata/php-interpolated-styles.php")
	require.NoError(t, err)

	p := php.AcquireParser()
	defer php.ReleaseParser(p)

	result, err := p.ParseCSS(string(source))
	require.NoError(t, err)
	require.NotNil(t, result)

	varNames := make([]string, len(result.VarCalls))
	for i, vc := range result.VarCalls {
		varNames[i] = vc.TokenName
	}

	// var() calls in the <style> tag (around PHP interpolation)
	assert.Contains(t, varNames, "--brand-color", "var(--brand-color) in style tag")
	assert.Contains(t, varNames, "--font-stack", "var(--font-stack) in style tag")
	assert.Contains(t, varNames, "--color-text", "var(--color-text) in style tag")

	// var() calls inside a PHP-conditional block within the style tag
	assert.Contains(t, varNames, "--gradient-start", "var(--gradient-start) in PHP conditional")
	assert.Contains(t, varNames, "--gradient-end", "var(--gradient-end) in PHP conditional")

	// var() call in a style attribute
	assert.Contains(t, varNames, "--color-primary", "var(--color-primary) in style attribute")

	// var() call in second style tag (short echo syntax <?= ?>)
	assert.Contains(t, varNames, "--spacing-section", "var(--spacing-section) in style tag with <?= ?>")

	// var() call in style attribute within PHP conditional
	assert.Contains(t, varNames, "--sidebar-width", "var(--sidebar-width) in conditional style attribute")
}

func TestComplexPHPBeforeStyles(t *testing.T) {
	// Multiple PHP statements with function calls (parentheses, commas, strings)
	// before HTML - this pattern breaks the HTML error-recovery approach
	// but works correctly with tree-sitter-php
	source := `<?php
$a = get_theme_mod('brand_color', '#0073aa');
$b = get_theme_mod('font_family', 'system-ui');
$c = array_merge($defaults, $overrides);
?>
<style>
.widget {
  color: var(--widget-color);
}
</style>`

	p := php.AcquireParser()
	defer php.ReleaseParser(p)

	result, err := p.ParseCSS(source)
	require.NoError(t, err)
	require.Len(t, result.VarCalls, 1)

	assert.Equal(t, "--widget-color", result.VarCalls[0].TokenName)
	// Style tag starts at line 5, var() on line 7 (0-indexed)
	assert.Equal(t, uint32(7), result.VarCalls[0].Range.Start.Line)
}

func TestNoPHPTags(t *testing.T) {
	// Pure HTML with no PHP - should still work through the pipeline
	source := `<style>.a { color: var(--x); }</style>`

	p := php.AcquireParser()
	defer php.ReleaseParser(p)

	result, err := p.ParseCSS(source)
	require.NoError(t, err)
	require.Len(t, result.VarCalls, 1)
	assert.Equal(t, "--x", result.VarCalls[0].TokenName)
}

func TestPHPOnly(t *testing.T) {
	// Pure PHP with no HTML or CSS - should return empty results
	source := `<?php
function render() {
  echo "hello";
  return 42;
}
?>`

	p := php.AcquireParser()
	defer php.ReleaseParser(p)

	result, err := p.ParseCSS(source)
	require.NoError(t, err)
	assert.Empty(t, result.Variables)
	assert.Empty(t, result.VarCalls)
}

func TestClosePool(t *testing.T) {
	// Exercise ClosePool - should not panic
	p := php.AcquireParser()
	php.ReleaseParser(p)
	php.ClosePool()

	// Pool is drained; acquiring again should still work (creates new parser)
	p2 := php.AcquireParser()
	defer php.ReleaseParser(p2)

	regions := p2.ParseCSSRegions(`<?php ?><style>.a{color:var(--x)}</style>`)
	assert.Len(t, regions, 1)
}
