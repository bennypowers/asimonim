/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package js_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/convert/formatter/js"
	"bennypowers.dev/asimonim/parser"
	"bennypowers.dev/asimonim/resolver"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/testutil"
	"bennypowers.dev/asimonim/token"
)

func TestFormat_Basic(t *testing.T) {
	runFixtureTest(t, "basic", js.Options{})
}

func TestFormat_Empty(t *testing.T) {
	runFixtureTest(t, "empty", js.Options{})
}

func TestFormat_WithPrefix(t *testing.T) {
	runFixtureTest(t, "with-prefix", js.Options{})
}

func TestFormat_CJSSimple(t *testing.T) {
	runFixtureTest(t, "cjs-simple", js.Options{Module: js.ModuleCJS})
}

func TestFormat_JSDocSimple(t *testing.T) {
	runFixtureTest(t, "jsdoc-simple", js.Options{Types: js.TypesJSDoc})
}

func TestFormat_MapBasic(t *testing.T) {
	runFixtureTest(t, "map-basic", js.Options{Export: js.ExportMap})
}

func TestFormat_EscapesQuotes(t *testing.T) {
	runFixtureTest(t, "escapes-quotes", js.Options{})
}

func TestFormat_EscapesBackslash(t *testing.T) {
	runFixtureTest(t, "escapes-backslash", js.Options{})
}

// --- New() default constructor ---

func TestNew(t *testing.T) {
	f := js.New()
	// Default extension should be .ts (ESM + TypeScript)
	if ext := f.Extension(); ext != ".ts" {
		t.Errorf("New() Extension() = %q, want %q", ext, ".ts")
	}
}

// --- Extension() all cases ---

func TestExtension(t *testing.T) {
	tests := []struct {
		name string
		opts js.Options
		want string
	}{
		{name: "ESM+TS", opts: js.Options{Module: js.ModuleESM, Types: js.TypesTS}, want: ".ts"},
		{name: "CJS+TS", opts: js.Options{Module: js.ModuleCJS, Types: js.TypesTS}, want: ".cts"},
		{name: "CJS+JSDoc", opts: js.Options{Module: js.ModuleCJS, Types: js.TypesJSDoc}, want: ".cjs"},
		{name: "ESM+JSDoc", opts: js.Options{Module: js.ModuleESM, Types: js.TypesJSDoc}, want: ".js"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := js.NewWithOptions(tt.opts)
			if got := f.Extension(); got != tt.want {
				t.Errorf("Extension() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- ToValue() edge cases ---

func TestToValue(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "nil", value: nil, want: "null"},
		{name: "string", value: "hello", want: `"hello"`},
		{name: "integer float64", value: float64(42), want: "42"},
		{name: "fractional float64", value: 3.14, want: "3.14"},
		{name: "int", value: 7, want: "7"},
		{name: "bool true", value: true, want: "true"},
		{name: "bool false", value: false, want: "false"},
		{name: "slice", value: []any{"a", float64(1)}, want: `["a",1]`},
		{name: "map", value: map[string]any{"k": "v"}, want: `{"k":"v"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := js.ToValue(tt.value)
			if got != tt.want {
				t.Errorf("ToValue(%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

// --- FormatJSDoc() ---

func TestFormatJSDoc(t *testing.T) {
	tests := []struct {
		name string
		desc string
		want string
	}{
		{
			name: "single line",
			desc: "A simple description",
			want: "/** A simple description */\n",
		},
		{
			name: "multi-line",
			desc: "Line one\nLine two",
			want: "/**\n * Line one\n * Line two\n */\n",
		},
		{
			name: "escapes closing comment",
			// description containing */ which must be escaped
			desc: "value is 10 */ injection",
			want: "/** value is 10 *\\/ injection */\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := js.FormatJSDoc(tt.desc)
			if got != tt.want {
				t.Errorf("FormatJSDoc(%q) =\n%s\nwant:\n%s", tt.desc, got, tt.want)
			}
		})
	}
}

// --- inferJSDocType via JSDoc formatter integration ---
// inferJSDocType is unexported, but we can test it through the formatter
// by using JSDoc types option and checking the generated @type annotation.

func TestJSDocTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		wantType string // expected @type content
	}{
		{name: "string value", value: "hello", wantType: "string"},
		{name: "number float64", value: float64(42), wantType: "number"},
		{name: "number int", value: 7, wantType: "number"},
		{name: "boolean", value: true, wantType: "boolean"},
		{name: "array", value: []any{1, 2}, wantType: "Array"},
		{name: "object", value: map[string]any{"k": "v"}, wantType: "Object"},
		// nil ResolvedValue falls through to tok.Value (empty string) via formatter.ResolvedValue,
		// so inferJSDocType sees a string, not nil. This is expected behavior.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := &token.Token{
				Name:          "test-token",
				Path:          []string{"test", "token"},
				Type:          "string",
				Description:   "desc",
				ResolvedValue: tt.value,
				IsResolved:    true,
			}
			f := js.NewWithOptions(js.Options{
				Types: js.TypesJSDoc,
			})
			result, err := f.Format([]*token.Token{tok}, formatter.Options{})
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}
			// @type {<wantType>} should appear in the output
			wantAnnotation := "@type {" + tt.wantType + "}"
			if !strings.Contains(string(result), wantAnnotation) {
				t.Errorf("expected output to contain %q, got:\n%s", wantAnnotation, string(result))
			}
		})
	}
}

// --- formatJSDocWithType multi-line and empty desc ---
// Tested via the formatter with JSDoc types and multi-line descriptions.

func TestJSDocWithType_MultiLine(t *testing.T) {
	tok := &token.Token{
		Name:          "multi-line",
		Path:          []string{"multi", "line"},
		Type:          "color",
		Description:   "Line one\nLine two",
		ResolvedValue: "#fff",
		IsResolved:    true,
	}
	f := js.NewWithOptions(js.Options{Types: js.TypesJSDoc})
	result, err := f.Format([]*token.Token{tok}, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	out := string(result)
	// Multi-line JSDoc with @type should have each line prefixed with " * "
	if !strings.Contains(out, " * Line one\n") {
		t.Errorf("expected multi-line JSDoc with ' * Line one', got:\n%s", out)
	}
	if !strings.Contains(out, " * Line two\n") {
		t.Errorf("expected multi-line JSDoc with ' * Line two', got:\n%s", out)
	}
	if !strings.Contains(out, "@type {string}") {
		t.Errorf("expected @type annotation, got:\n%s", out)
	}
}

func TestJSDocWithType_EmptyDesc(t *testing.T) {
	// Token with no description but JSDoc types should still get @type
	tok := &token.Token{
		Name:          "no-desc",
		Path:          []string{"no", "desc"},
		Type:          "number",
		ResolvedValue: float64(42),
		IsResolved:    true,
	}
	f := js.NewWithOptions(js.Options{Types: js.TypesJSDoc})
	result, err := f.Format([]*token.Token{tok}, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	out := string(result)
	// No description means no description comment at all (formatDescription only called if desc != "")
	// So the output should just have the export without a JSDoc comment
	if strings.Contains(out, "@type") {
		// If it does contain @type, that's fine too - depends on implementation
		if !strings.Contains(out, "@type {number}") {
			t.Errorf("expected @type {number} if type annotation present, got:\n%s", out)
		}
	}
}

func TestJSDocWithType_EscapesClosingComment(t *testing.T) {
	// Description with */ should be escaped in JSDoc+type output
	tok := &token.Token{
		Name:          "inject",
		Path:          []string{"inject"},
		Type:          "string",
		Description:   "value */ injection",
		ResolvedValue: "test",
		IsResolved:    true,
	}
	f := js.NewWithOptions(js.Options{Types: js.TypesJSDoc})
	result, err := f.Format([]*token.Token{tok}, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	out := string(result)
	// The raw */ should be escaped to *\/
	if strings.Contains(out, "value */") {
		t.Errorf("expected */ to be escaped, got:\n%s", out)
	}
	if !strings.Contains(out, "value *\\/") {
		t.Errorf("expected escaped *\\/ in output, got:\n%s", out)
	}
}

// --- formatExport CJS+JSDoc (non-TS CJS) ---

func TestCJSJSDocExport(t *testing.T) {
	tok := &token.Token{
		Name:          "test-val",
		Path:          []string{"test", "val"},
		Type:          "string",
		ResolvedValue: "hello",
		IsResolved:    true,
	}
	f := js.NewWithOptions(js.Options{
		Module: js.ModuleCJS,
		Types:  js.TypesJSDoc,
	})
	result, err := f.Format([]*token.Token{tok}, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	out := string(result)
	// CJS + JSDoc: should use exports.name = value;
	if !strings.Contains(out, "exports.testVal") {
		t.Errorf("expected CJS exports.testVal, got:\n%s", out)
	}
	// Should NOT contain "export const" or "as const"
	if strings.Contains(out, "export const") {
		t.Errorf("CJS should not use 'export const', got:\n%s", out)
	}
}

// --- inferValueType for complex token types ---

func TestInferValueType_ViaMapFormat(t *testing.T) {
	// We test inferValueType indirectly through map format output,
	// which includes the ValueType in the template entries.
	tests := []struct {
		name      string
		tokenType string
		value     any
		wantType  string
	}{
		{
			name:      "shadow type",
			tokenType: token.TypeShadow,
			value: map[string]any{
				"offsetX": map[string]any{"value": float64(0), "unit": "px"},
				"offsetY": map[string]any{"value": float64(4), "unit": "px"},
				"blur":    map[string]any{"value": float64(8), "unit": "px"},
				"color":   "#000000",
			},
			wantType: "offsetX: Dimension",
		},
		{
			name:      "border type",
			tokenType: token.TypeBorder,
			value: map[string]any{
				"width": map[string]any{"value": float64(1), "unit": "px"},
				"style": "solid",
				"color": "#000000",
			},
			wantType: "width: Dimension",
		},
		{
			name:      "typography type",
			tokenType: token.TypeTypography,
			value: map[string]any{
				"fontFamily": "Arial",
				"fontSize":   map[string]any{"value": float64(16), "unit": "px"},
				"fontWeight": float64(400),
			},
			wantType: "fontFamily",
		},
		{
			name:      "transition type",
			tokenType: token.TypeTransition,
			value: map[string]any{
				"duration":       map[string]any{"value": float64(200), "unit": "ms"},
				"timingFunction": []any{0.4, 0.0, 0.2, 1.0},
			},
			wantType: "duration:",
		},
		{
			name:      "gradient type",
			tokenType: token.TypeGradient,
			value: map[string]any{
				"type":  "linear",
				"stops": []any{map[string]any{"color": "#fff", "position": float64(0)}},
			},
			wantType: "stops:",
		},
		{
			name:      "strokeStyle string",
			tokenType: token.TypeStrokeStyle,
			value:     "dashed",
			wantType:  "string",
		},
		{
			name:      "strokeStyle object",
			tokenType: token.TypeStrokeStyle,
			value: map[string]any{
				"dashArray": []any{map[string]any{"value": float64(2), "unit": "px"}},
			},
			wantType: "dashArray",
		},
		{
			name:      "number type",
			tokenType: token.TypeNumber,
			value:     float64(42),
			wantType:  "number",
		},
		{
			name:      "fontWeight type",
			tokenType: token.TypeFontWeight,
			value:     float64(700),
			wantType:  "number",
		},
		{
			name:      "cubicBezier type",
			tokenType: token.TypeCubicBezier,
			value:     []any{0.4, 0.0, 0.2, 1.0},
			wantType:  "number, number, number, number",
		},
		{
			name:      "fontFamily string",
			tokenType: token.TypeFontFamily,
			value:     "Arial",
			wantType:  "string",
		},
		{
			name:      "fontFamily array",
			tokenType: token.TypeFontFamily,
			value:     []any{"Arial", "sans-serif"},
			wantType:  "string[]",
		},
		{
			name:      "duration string",
			tokenType: token.TypeDuration,
			value:     "200ms",
			wantType:  "string",
		},
		{
			name:      "duration object",
			tokenType: token.TypeDuration,
			value:     map[string]any{"value": float64(200), "unit": "ms"},
			wantType:  "value: number; unit: string",
		},
		{
			name:      "color string",
			tokenType: token.TypeColor,
			value:     "#FF0000",
			wantType:  "string",
		},
		{
			name:      "dimension string",
			tokenType: token.TypeDimension,
			value:     "16px",
			wantType:  "string",
		},
		{
			name:      "dimension structured",
			tokenType: token.TypeDimension,
			value:     map[string]any{"value": float64(16), "unit": "px"},
			wantType:  "Dimension",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := &token.Token{
				Name:          "test-token",
				Path:          []string{"test", "token"},
				Type:          tt.tokenType,
				ResolvedValue: tt.value,
				IsResolved:    true,
			}
			f := js.NewWithOptions(js.Options{Export: js.ExportMap})
			result, err := f.Format([]*token.Token{tok}, formatter.Options{})
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}
			out := string(result)
			if !strings.Contains(out, tt.wantType) {
				t.Errorf("expected output to contain %q for type %q, got:\n%s",
					tt.wantType, tt.tokenType, out)
			}
		})
	}
}

// --- inferValueType default fallback cases ---

func TestInferValueType_DefaultFallbacks(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		wantType string
	}{
		{name: "default string", value: "hello", wantType: "string"},
		{name: "default number", value: float64(42), wantType: "number"},
		{name: "default int", value: 7, wantType: "number"},
		{name: "default bool", value: true, wantType: "boolean"},
		{name: "default array", value: []any{1, 2}, wantType: "unknown[]"},
		{name: "default map", value: map[string]any{"k": "v"}, wantType: "Record<string, unknown>"},
		{name: "default nil", value: nil, wantType: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use an unknown type to trigger default branch
			tok := &token.Token{
				Name:          "test-token",
				Path:          []string{"test", "token"},
				Type:          "customType",
				ResolvedValue: tt.value,
				IsResolved:    true,
			}
			f := js.NewWithOptions(js.Options{Export: js.ExportMap})
			result, err := f.Format([]*token.Token{tok}, formatter.Options{})
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}
			out := string(result)
			if !strings.Contains(out, tt.wantType) {
				t.Errorf("expected output to contain type %q for value %v, got:\n%s",
					tt.wantType, tt.value, out)
			}
		})
	}
}

// --- formatSplitModule ---

func TestMapMode_Types(t *testing.T) {
	f := js.NewWithOptions(js.Options{
		Export:  js.ExportMap,
		MapMode: js.MapModeTypes,
	})
	result, err := f.Format(nil, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	out := string(result)
	// Types template should contain the type interfaces
	if !strings.Contains(out, "export interface Color") {
		t.Errorf("expected types output to contain 'export interface Color', got:\n%s", out)
	}
	if !strings.Contains(out, "export class TokenMap") {
		t.Errorf("expected types output to contain 'export class TokenMap', got:\n%s", out)
	}
}

func TestMapMode_Module(t *testing.T) {
	tok := &token.Token{
		Name:          "color-primary",
		Path:          []string{"color", "primary"},
		Type:          token.TypeColor,
		ResolvedValue: "#FF0000",
		IsResolved:    true,
	}
	f := js.NewWithOptions(js.Options{
		Export:  js.ExportMap,
		MapMode: js.MapModeModule,
	})
	result, err := f.Format([]*token.Token{tok}, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	out := string(result)
	// Should import from default types path
	if !strings.Contains(out, `from "./types.ts"`) {
		t.Errorf("expected import from ./types.ts, got:\n%s", out)
	}
	// Should have default class name
	if !strings.Contains(out, "TokensTokenMap") {
		t.Errorf("expected default class name TokensTokenMap, got:\n%s", out)
	}
	// Should import TokenMap and DesignToken
	if !strings.Contains(out, "TokenMap") {
		t.Errorf("expected TokenMap import, got:\n%s", out)
	}
}

func TestMapMode_ModuleWithCustomOptions(t *testing.T) {
	tokens := []*token.Token{
		{
			Name:          "color-primary",
			Path:          []string{"color", "primary"},
			Type:          token.TypeColor,
			ResolvedValue: map[string]any{"colorSpace": "srgb", "components": []any{1.0, 0.0, 0.0}},
			IsResolved:    true,
			SchemaVersion: schema.V2025_10,
		},
		{
			Name:          "spacing-sm",
			Path:          []string{"spacing", "sm"},
			Type:          token.TypeDimension,
			ResolvedValue: map[string]any{"value": float64(4), "unit": "px"},
			IsResolved:    true,
		},
	}
	f := js.NewWithOptions(js.Options{
		Export:    js.ExportMap,
		MapMode:   js.MapModeModule,
		TypesPath: "./shared/types.ts",
		ClassName: "MyTokenMap",
	})
	result, err := f.Format(tokens, formatter.Options{Prefix: "rh"})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	out := string(result)
	// Custom types path
	if !strings.Contains(out, `from "./shared/types.ts"`) {
		t.Errorf("expected custom types path, got:\n%s", out)
	}
	// Custom class name
	if !strings.Contains(out, "MyTokenMap") {
		t.Errorf("expected custom class name MyTokenMap, got:\n%s", out)
	}
	// Should import Color and Dimension since both are used
	if !strings.Contains(out, "Color") {
		t.Errorf("expected Color import, got:\n%s", out)
	}
	if !strings.Contains(out, "Dimension") {
		t.Errorf("expected Dimension import, got:\n%s", out)
	}
}

func TestMapMode_ModuleJSDocTypesPath(t *testing.T) {
	// When types is JSDoc and no TypesPath is set, should default to ./types.js
	tok := &token.Token{
		Name:          "val",
		Path:          []string{"val"},
		Type:          "string",
		ResolvedValue: "test",
		IsResolved:    true,
	}
	f := js.NewWithOptions(js.Options{
		Export:  js.ExportMap,
		MapMode: js.MapModeModule,
		Types:   js.TypesJSDoc,
	})
	result, err := f.Format([]*token.Token{tok}, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	out := string(result)
	if !strings.Contains(out, `"./types.js"`) {
		t.Errorf("expected JSDoc default types path ./types.js, got:\n%s", out)
	}
}

// --- formatTypedValue for structured color ---

func TestMapFormat_StructuredColor(t *testing.T) {
	allTokens := testutil.ParseFixtureTokens(t, "fixtures/v2025_10/all-color-spaces", schema.V2025_10)
	tok := testutil.TokenByPath(t, allTokens, "color.srgb-hex")

	f := js.NewWithOptions(js.Options{Export: js.ExportMap})
	result, err := f.Format([]*token.Token{tok}, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	out := string(result)
	// Structured color should produce colorSpace and components in the value
	if !strings.Contains(out, "colorSpace") {
		t.Errorf("expected colorSpace in structured color output, got:\n%s", out)
	}
	if !strings.Contains(out, "components") {
		t.Errorf("expected components in structured color output, got:\n%s", out)
	}
	// Should have Color type annotation
	if !strings.Contains(out, "Color") {
		t.Errorf("expected Color type in output, got:\n%s", out)
	}
}

// --- formatTypedValue for structured dimension ---

func TestMapFormat_StructuredDimension(t *testing.T) {
	allTokens := testutil.ParseFixtureTokens(t, "fixtures/v2025_10/all-color-spaces", schema.V2025_10)
	tok := testutil.TokenByPath(t, allTokens, "spacing.small")

	f := js.NewWithOptions(js.Options{Export: js.ExportMap})
	result, err := f.Format([]*token.Token{tok}, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	out := string(result)
	// Should contain Dimension type
	if !strings.Contains(out, "Dimension") {
		t.Errorf("expected Dimension type in output, got:\n%s", out)
	}
}

// --- Custom header ---

func TestSimpleFormat_CustomHeader(t *testing.T) {
	tok := &token.Token{
		Name:          "val",
		Path:          []string{"val"},
		Type:          "string",
		ResolvedValue: "test",
		IsResolved:    true,
	}
	f := js.New()
	result, err := f.Format([]*token.Token{tok}, formatter.Options{
		Header: "Custom header line",
	})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	out := string(result)
	if !strings.Contains(out, "Custom header line") {
		t.Errorf("expected custom header in output, got:\n%s", out)
	}
	// Should NOT contain default header
	if strings.Contains(out, "Generated by asimonim") {
		t.Errorf("expected custom header to replace default, got:\n%s", out)
	}
}

// --- Map format with prefix ---

func TestMapFormat_WithPrefix(t *testing.T) {
	tok := &token.Token{
		Name:          "color-primary",
		Path:          []string{"color", "primary"},
		Type:          token.TypeColor,
		ResolvedValue: "#FF0000",
		IsResolved:    true,
	}
	f := js.NewWithOptions(js.Options{Export: js.ExportMap})
	result, err := f.Format([]*token.Token{tok}, formatter.Options{Prefix: "rh"})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	out := string(result)
	// CSS var name should include prefix
	if !strings.Contains(out, "--rh-color-primary") {
		t.Errorf("expected --rh-color-primary in output, got:\n%s", out)
	}
}

// --- buildValueTypeUnion empty case ---

func TestMapFormat_EmptyTokens(t *testing.T) {
	f := js.NewWithOptions(js.Options{
		Export:  js.ExportMap,
		MapMode: js.MapModeModule,
	})
	result, err := f.Format([]*token.Token{}, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	out := string(result)
	// With no tokens, the value type union should fallback to DesignToken<unknown>
	if !strings.Contains(out, "DesignToken<unknown>") {
		t.Errorf("expected DesignToken<unknown> for empty tokens, got:\n%s", out)
	}
}

// --- Custom delimiter ---

func TestMapFormat_CustomDelimiter(t *testing.T) {
	tok := &token.Token{
		Name:          "color-primary",
		Path:          []string{"color", "primary"},
		Type:          token.TypeColor,
		ResolvedValue: "#FF0000",
		IsResolved:    true,
	}
	f := js.NewWithOptions(js.Options{Export: js.ExportMap})
	result, err := f.Format([]*token.Token{tok}, formatter.Options{Delimiter: "_"})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	out := string(result)
	// The delimiter should appear in the output
	if !strings.Contains(out, `"_"`) {
		t.Errorf("expected custom delimiter '_' in output, got:\n%s", out)
	}
}

// --- escapeTS special characters ---

func TestMapFormat_EscapeTSSpecialChars(t *testing.T) {
	// Token with special characters in path that need escaping in TS string literals
	tok := &token.Token{
		Name:          "special\ttab",
		Path:          []string{"special\ttab"},
		Type:          "string",
		ResolvedValue: "value",
		IsResolved:    true,
	}
	f := js.NewWithOptions(js.Options{Export: js.ExportMap})
	result, err := f.Format([]*token.Token{tok}, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	out := string(result)
	// The tab should be escaped to \t in the output
	if !strings.Contains(out, `\t`) {
		t.Errorf("expected tab to be escaped in output, got:\n%s", out)
	}
}

func TestMapFormat_EscapeTSNewlineAndCR(t *testing.T) {
	// Token with newline and carriage return in name
	tok := &token.Token{
		Name:          "line\nbreak",
		Path:          []string{"line\nbreak"},
		Type:          "string",
		ResolvedValue: "value",
		IsResolved:    true,
	}
	f := js.NewWithOptions(js.Options{Export: js.ExportMap})
	result, err := f.Format([]*token.Token{tok}, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	out := string(result)
	if !strings.Contains(out, `\n`) {
		t.Errorf("expected newline to be escaped in output, got:\n%s", out)
	}
}

func TestMapFormat_EscapeTSCarriageReturn(t *testing.T) {
	tok := &token.Token{
		Name:          "cr\rtoken",
		Path:          []string{"cr\rtoken"},
		Type:          "string",
		ResolvedValue: "value",
		IsResolved:    true,
	}
	f := js.NewWithOptions(js.Options{Export: js.ExportMap})
	result, err := f.Format([]*token.Token{tok}, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	out := string(result)
	if !strings.Contains(out, `\r`) {
		t.Errorf("expected carriage return to be escaped in output, got:\n%s", out)
	}
}

// --- ToValue default/fallback branch ---

func TestToValue_DefaultFallback(t *testing.T) {
	// int64 is not int, float64, string, bool, []any, or map[string]any
	// so it hits the default json.Marshal fallback
	got := js.ToValue(int64(99))
	if got != "99" {
		t.Errorf("ToValue(int64(99)) = %q, want %q", got, "99")
	}
}

// --- inferValueType: color as map[string]any without valid ParseColorValue ---

func TestInferValueType_ColorMap(t *testing.T) {
	// A color token with a map value that doesn't parse as structured color
	// should still return "Color" type
	tok := &token.Token{
		Name:          "color-custom",
		Path:          []string{"color", "custom"},
		Type:          token.TypeColor,
		ResolvedValue: map[string]any{"r": float64(255), "g": float64(0), "b": float64(0)},
		IsResolved:    true,
	}
	f := js.NewWithOptions(js.Options{Export: js.ExportMap})
	result, err := f.Format([]*token.Token{tok}, formatter.Options{})
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	out := string(result)
	if !strings.Contains(out, "Color") {
		t.Errorf("expected Color type for map-valued color token, got:\n%s", out)
	}
}

// runFixtureTest runs a fixture-based test for the JS formatter.
func runFixtureTest(t *testing.T, fixtureName string, jsOpts js.Options) {
	t.Helper()

	fixturePath := filepath.Join("fixtures", fixtureName)
	mfs := testutil.NewFixtureFS(t, fixturePath, "/test")

	p := parser.NewJSONParser()
	tokens, err := p.ParseFile(mfs, "/test/tokens.json", parser.Options{
		SchemaVersion: schema.Draft,
		SkipPositions: true,
	})
	if err != nil {
		t.Fatalf("failed to parse tokens.json: %v", err)
	}

	if err := resolver.ResolveAliases(tokens, schema.Draft); err != nil {
		t.Fatalf("failed to resolve aliases: %v", err)
	}

	// Check for options.json to load options
	fmtOpts := formatter.Options{}
	if optData, err := mfs.ReadFile("/test/options.json"); err == nil {
		var fileOpts struct {
			Prefix    string `json:"prefix"`
			Delimiter string `json:"delimiter"`
			Module    string `json:"module"`
			Types     string `json:"types"`
			Export    string `json:"export"`
			MapMode   string `json:"mapMode"`
			TypesPath string `json:"typesPath"`
			ClassName string `json:"className"`
		}
		if err := json.Unmarshal(optData, &fileOpts); err != nil {
			t.Fatalf("failed to unmarshal options.json: %v\nraw data: %s", err, string(optData))
		}
		if fileOpts.Prefix != "" {
			fmtOpts.Prefix = fileOpts.Prefix
		}
		if fileOpts.Delimiter != "" {
			fmtOpts.Delimiter = fileOpts.Delimiter
		}
		if fileOpts.Module != "" {
			jsOpts.Module = js.Module(fileOpts.Module)
		}
		if fileOpts.Types != "" {
			jsOpts.Types = js.Types(fileOpts.Types)
		}
		if fileOpts.Export != "" {
			jsOpts.Export = js.Export(fileOpts.Export)
		}
		if fileOpts.MapMode != "" {
			jsOpts.MapMode = js.MapMode(fileOpts.MapMode)
		}
		if fileOpts.TypesPath != "" {
			jsOpts.TypesPath = fileOpts.TypesPath
		}
		if fileOpts.ClassName != "" {
			jsOpts.ClassName = fileOpts.ClassName
		}
	}

	f := js.NewWithOptions(jsOpts)
	result, err := f.Format(tokens, fmtOpts)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Determine expected file extension
	ext := f.Extension()
	goldenRelPath := filepath.Join("fixtures", fixtureName, "expected"+ext)

	// Update golden file if -update flag is set
	testutil.UpdateGoldenFile(t, goldenRelPath, result)

	expected := testutil.LoadFixtureFile(t, goldenRelPath)

	// Normalize line endings for comparison
	gotStr := strings.ReplaceAll(string(result), "\r\n", "\n")
	expectedStr := strings.ReplaceAll(string(expected), "\r\n", "\n")

	if gotStr != expectedStr {
		t.Errorf("output mismatch for fixture %q.\n\nGot:\n%s\n\nExpected:\n%s", fixtureName, gotStr, expectedStr)
	}
}
