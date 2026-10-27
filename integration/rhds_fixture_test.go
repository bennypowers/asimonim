/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

// Package integration_test verifies asimonim against the normalized RHDS token
// fixture.
//
// The fixture is derived from the published @rhds/tokens@3.2.0
// json/rhds.tokens.json Style Dictionary document. It is normalized to DTCG
// forms before asimonim sees it, except for the RHDS vendor type
// "mediaQuery", which no DTCG version defines:
//
//   - $value comes from original.$value so aliases survive
//   - $type is recovered from attributes with an explicit mapping
//   - $description, $deprecated, and $extensions are kept
//   - Style Dictionary fields are dropped
//   - literal values are canonicalized to draft DTCG forms (duration in ms,
//     number percentages as unitless numbers, literal colors as hex, shadow
//     dimensions with explicit units) while alias references are untouched
//
// Because "mediaQuery" is a vendor type, the fixture is not a strictly
// DTCG-valid document. asimonim's consistency validator accepts it only
// because that validator does not check $type membership.
//
// Generated tokens in the source document:
//
//   - The source contains 160 generated -rgb and -hsl color variants; this
//     fixture retains two representative aliases, color.red-orange.60-rgb and
//     color.red-orange.60-hsl. Their original values alias the base color, so
//     the normalized tokens resolve through those aliases.
//   - 57 light-dark(var(...)) composite tokens are excluded. 56 original
//     values are arrays of two per-theme aliases; color.surface._ is a
//     six-element palette of surface aliases. Neither is a DTCG color value.
//     The @rhds/mcp generator keeps them in separate computed theme and
//     palette maps instead.
//
// The fixture carries the same counts in its root $extensions so the
// contract is inspectable without this test file.
package integration_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"bennypowers.dev/asimonim/parser"
	"bennypowers.dev/asimonim/resolver"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/testutil"
	"bennypowers.dev/asimonim/token"
	"bennypowers.dev/asimonim/validator"
)

const (
	rhdsFixtureDir  = "fixtures/rhds/v3_2_0"
	rhdsFixtureFile = rhdsFixtureDir + "/tokens.json"

	// rhdsFixtureTokenCount is the number of curated tokens in the fixture.
	rhdsFixtureTokenCount = 25
)

var (
	rhdsAliasForm     = regexp.MustCompile(`^\{[^}]+\}$`)
	rhdsDurationForm  = regexp.MustCompile(`^\d+(?:\.\d+)?ms$`)
	rhdsColorForm     = regexp.MustCompile(`^#[0-9A-Fa-f]{6}([0-9A-Fa-f]{2})?$`)
	rhdsDimensionForm = regexp.MustCompile(`^-?\d+(?:\.\d+)?(?:px|rem)$`)
)

// parseRHDS parses the fixture without an explicit schema version, so the
// parser and schema detector have to identify it on their own.
func parseRHDS(t *testing.T) []*token.Token {
	t.Helper()

	mfs := testutil.NewFixtureFS(t, rhdsFixtureDir, "/test")
	tokens, err := parser.NewJSONParser().ParseFile(mfs, "/test/tokens.json", parser.Options{
		Prefix:        "rh",
		SkipPositions: true,
	})
	if err != nil {
		t.Fatalf("failed to parse RHDS fixture: %v", err)
	}
	return tokens
}

func loadRHDSRaw(t *testing.T) map[string]any {
	t.Helper()

	var raw map[string]any
	if err := json.Unmarshal(testutil.LoadFixtureFile(t, rhdsFixtureFile), &raw); err != nil {
		t.Fatalf("failed to unmarshal RHDS fixture: %v", err)
	}
	return raw
}

func TestRHDSFixture_ParserAcceptsAndAutoDetects(t *testing.T) {
	tokens := parseRHDS(t)

	if len(tokens) != rhdsFixtureTokenCount {
		t.Fatalf("expected %d tokens, got %d", rhdsFixtureTokenCount, len(tokens))
	}

	wantTypes := map[string]bool{
		token.TypeColor:       true,
		token.TypeDimension:   true,
		token.TypeNumber:      true,
		token.TypeFontFamily:  true,
		token.TypeFontWeight:  true,
		"mediaQuery":          true,
		token.TypeShadow:      true,
		token.TypeCubicBezier: true,
		token.TypeDuration:    true,
	}

	seenTypes := make(map[string]bool, len(wantTypes))
	for _, tok := range tokens {
		if tok.Type == "" {
			t.Errorf("token %q has no $type", tok.DotPath())
		}
		if tok.SchemaVersion != schema.Draft {
			t.Errorf("token %q schema version = %s, want draft (auto-detected)", tok.DotPath(), tok.SchemaVersion)
		}
		if !wantTypes[tok.Type] {
			t.Errorf("token %q has unexpected type %q", tok.DotPath(), tok.Type)
		}
		seenTypes[tok.Type] = true
	}
	for typ := range wantTypes {
		if !seenTypes[typ] {
			t.Errorf("fixture does not cover token type %q", typ)
		}
	}

	// The five tokens in the four source groups without $type must be
	// recovered explicitly.
	recovered := map[string]string{
		"border.radius.default":    token.TypeDimension,
		"border.radius.sharp":      token.TypeDimension,
		"letter-spacing.body.text": token.TypeDimension,
		"line-height.body-text":    token.TypeNumber,
		"opacity.50":               token.TypeNumber,
	}
	for dotPath, wantType := range recovered {
		tok := testutil.TokenByPath(t, tokens, dotPath)
		if tok.Type != wantType {
			t.Errorf("%s: type = %q, want %q", dotPath, tok.Type, wantType)
		}
	}
}

func TestRHDSFixture_PrefixProducesRHCustomProperties(t *testing.T) {
	tokens := parseRHDS(t)

	tests := map[string]string{
		"color.red-orange.60":   "--rh-color-red-orange-60",
		"border.radius.default": "--rh-border-radius-default",
		"font.family.heading":   "--rh-font-family-heading",
		"media.sm":              "--rh-media-sm",
	}
	for dotPath, wantName := range tests {
		tok := testutil.TokenByPath(t, tokens, dotPath)
		if got := tok.CSSVariableName(); got != wantName {
			t.Errorf("%s: CSS variable = %q, want %q", dotPath, got, wantName)
		}
	}

	tokenMap := token.NewMap(tokens, "rh")
	if got := tokenMap.Len(); got != len(tokens) {
		t.Errorf("token map length = %d, want %d", got, len(tokens))
	}
	want := testutil.TokenByPath(t, tokens, "color.red-orange.60")
	for _, name := range []string{"color-red-orange-60", "color.red-orange.60", "--rh-color-red-orange-60"} {
		tok, ok := tokenMap.Get(name)
		if !ok {
			t.Errorf("token map lookup %q missed", name)
			continue
		}
		if tok.DotPath() != want.DotPath() {
			t.Errorf("token map lookup %q returned %q, want %q", name, tok.DotPath(), want.DotPath())
		}
		if tok.Prefix != "rh" {
			t.Errorf("token map lookup %q prefix = %q, want rh", name, tok.Prefix)
		}
	}
	if _, ok := tokenMap.Get("does-not-exist"); ok {
		t.Error("token map lookup of an unknown name returned a token")
	}
}

func TestRHDSFixture_AliasesSurviveAndResolve(t *testing.T) {
	tokens := parseRHDS(t)
	if err := resolver.ResolveAliases(tokens, schema.Draft); err != nil {
		t.Fatalf("failed to resolve aliases: %v", err)
	}

	// original.$value wins over the resolved Style Dictionary value. The -rgb
	// variant's published value is "177 56 11", which is not a DTCG color, but
	// its original value is an alias to the base color.
	rgb := testutil.TokenByPath(t, tokens, "color.red-orange.60-rgb")
	if rgb.Value != "{color.red-orange.60}" {
		t.Errorf("-rgb value = %q, want alias {color.red-orange.60}", rgb.Value)
	}
	if rgb.ResolvedValue != "#B1380B" {
		t.Errorf("-rgb resolved value = %v, want #B1380B", rgb.ResolvedValue)
	}
	if got, want := rgb.ResolutionChain, []string{"color-red-orange-60"}; !reflect.DeepEqual(got, want) {
		t.Errorf("-rgb resolution chain = %v, want %v", got, want)
	}

	// A three-hop alias chain: border destructive -> border status -> status ->
	// crayon.
	chain := testutil.TokenByPath(t, tokens, "color.border.destructive.on-light")
	if chain.Value != "{color.border.status.danger.on-light}" {
		t.Errorf("chain value = %q", chain.Value)
	}
	if chain.ResolvedValue != "#B1380B" {
		t.Errorf("chain resolved value = %v, want #B1380B", chain.ResolvedValue)
	}
	wantChain := []string{
		"color-border-status-danger-on-light",
		"color-status-danger-on-light",
		"color-red-orange-60",
	}
	if !reflect.DeepEqual(chain.ResolutionChain, wantChain) {
		t.Errorf("resolution chain = %v, want %v", chain.ResolutionChain, wantChain)
	}

	dims := map[string]struct {
		value string
		chain []string
	}{
		"border.radius.default": {"3px", []string{"length-2xs"}},
		"border.width.sm":       {"1px", []string{"length-4xs"}},
	}
	for dotPath, want := range dims {
		tok := testutil.TokenByPath(t, tokens, dotPath)
		if tok.ResolvedValue != want.value {
			t.Errorf("%s: resolved value = %v, want %s", dotPath, tok.ResolvedValue, want.value)
		}
		if !reflect.DeepEqual(tok.ResolutionChain, want.chain) {
			t.Errorf("%s: chain = %v, want %v", dotPath, tok.ResolutionChain, want.chain)
		}
	}

	// Composite values survive as composite values. DTCG composite types do
	// allow sub-value references, and tools must resolve them when they need
	// the value. asimonim does not resolve sub-values inside composites; that
	// is an asimonim limitation, not a DTCG rule. The reference therefore
	// survives in both RawValue and ResolvedValue.
	shadow := testutil.TokenByPath(t, tokens, "box-shadow.md")
	rawShadow, ok := shadow.RawValue.(map[string]any)
	if !ok {
		t.Fatalf("shadow raw value type = %T, want map", shadow.RawValue)
	}
	if rawShadow["offsetY"] != "{length.xs}" {
		t.Errorf("shadow raw offsetY = %v, want {length.xs}", rawShadow["offsetY"])
	}
	shadowMap, ok := shadow.ResolvedValue.(map[string]any)
	if !ok {
		t.Fatalf("shadow resolved value type = %T, want map", shadow.ResolvedValue)
	}
	if shadowMap["offsetY"] != "{length.xs}" {
		t.Errorf("shadow offsetY = %v, want {length.xs}", shadowMap["offsetY"])
	}
	if shadowMap["color"] != "#15151540" {
		t.Errorf("shadow color = %v, want canonical #15151540", shadowMap["color"])
	}
}

func TestRHDSFixture_MetadataSurvives(t *testing.T) {
	tokens := parseRHDS(t)

	color := testutil.TokenByPath(t, tokens, "color.red-orange.60")
	if color.Value != "#B1380B" {
		t.Errorf("color value = %q, want original $value #B1380B", color.Value)
	}
	printExt, ok := color.Extensions["com.redhat.print"].(map[string]any)
	if !ok {
		t.Fatalf("color extensions missing com.redhat.print: %v", color.Extensions)
	}
	if printExt["colorSpace"] != "cmyk" {
		t.Errorf("print extension colorSpace = %v, want cmyk", printExt["colorSpace"])
	}

	breakpoint := testutil.TokenByPath(t, tokens, "breakpoint.xs")
	uxExt, ok := breakpoint.Extensions["com.redhat.ux"].(map[string]any)
	if !ok {
		t.Fatalf("breakpoint extensions missing com.redhat.ux: %v", breakpoint.Extensions)
	}
	if uxExt["title"] != "Extra Small" {
		t.Errorf("com.redhat.ux title = %v, want Extra Small", uxExt["title"])
	}
	if breakpoint.Description != "Mobile portrait" {
		t.Errorf("breakpoint description = %q", breakpoint.Description)
	}

	deprecated := testutil.TokenByPath(t, tokens, "color.red-orange.60-hsl")
	if !deprecated.Deprecated {
		t.Error("hsl variant should be deprecated")
	}
	wantDeprecation := "Use color transforms instead e.g. hsla(from var(--rh-color-red-orange-60) h s l / 10%)"
	if deprecated.DeprecationMessage != wantDeprecation {
		t.Errorf("deprecation message = %q, want %q", deprecated.DeprecationMessage, wantDeprecation)
	}

	family := testutil.TokenByPath(t, tokens, "font.family.heading")
	wantFamily := []any{"RedHatDisplay", "Red Hat Display", "Helvetica", "Arial", "sans-serif"}
	if got := family.RawValue; !reflect.DeepEqual(got, wantFamily) {
		t.Errorf("font family raw value = %#v, want %#v", got, wantFamily)
	}

	media := testutil.TokenByPath(t, tokens, "media.sm")
	mediaValue, ok := media.RawValue.(map[string]any)
	if !ok {
		t.Fatalf("media raw value type = %T, want map", media.RawValue)
	}
	// mediaQuery is an RHDS vendor type. asimonim does not resolve sub-value
	// references inside its object value; DTCG does not govern this shape. The
	// reference is preserved for consumers to resolve.
	if mediaValue["min-width"] != "{breakpoint.sm}" {
		t.Errorf("media min-width = %v, want {breakpoint.sm}", mediaValue["min-width"])
	}
}

func TestRHDSFixture_CanonicalValueForms(t *testing.T) {
	tokens := parseRHDS(t)

	duration := testutil.TokenByPath(t, tokens, "animation.speed")
	if duration.Value != "300ms" {
		t.Errorf("duration value = %q, want 300ms", duration.Value)
	}
	assertSourceValue(t, duration, "0.3s")

	opacity := testutil.TokenByPath(t, tokens, "opacity.50")
	if opacity.Value != "0.5" {
		t.Errorf("opacity value = %q, want 0.5", opacity.Value)
	}
	if got, ok := opacity.RawValue.(float64); !ok || got != 0.5 {
		t.Errorf("opacity raw value = %#v, want float64 0.5", opacity.RawValue)
	}
	assertSourceValue(t, opacity, "50%")

	lineHeight := testutil.TokenByPath(t, tokens, "line-height.body-text")
	if got, ok := lineHeight.RawValue.(float64); !ok || got != 1.5 {
		t.Errorf("line-height raw value = %#v, want float64 1.5", lineHeight.RawValue)
	}

	weight := testutil.TokenByPath(t, tokens, "font.weight.heading.bold")
	if got, ok := weight.RawValue.(float64); !ok || got != 700 {
		t.Errorf("font weight raw value = %#v, want float64 700", weight.RawValue)
	}

	bezier := testutil.TokenByPath(t, tokens, "animation.timing")
	wantBezier := []any{0.465, 0.183, 0.153, 0.946}
	if !reflect.DeepEqual(bezier.RawValue, wantBezier) {
		t.Errorf("cubic bezier raw value = %#v, want %#v", bezier.RawValue, wantBezier)
	}

	shadow := testutil.TokenByPath(t, tokens, "box-shadow.md")
	shadowMap, ok := shadow.RawValue.(map[string]any)
	if !ok {
		t.Fatalf("shadow raw value type = %T, want map", shadow.RawValue)
	}
	if shadowMap["offsetX"] != "0px" {
		t.Errorf("shadow offsetX = %v, want 0px", shadowMap["offsetX"])
	}
	if shadowMap["offsetY"] != "{length.xs}" {
		t.Errorf("shadow offsetY = %v, alias should be preserved", shadowMap["offsetY"])
	}
	if shadowMap["blur"] != "{length.sm}" {
		t.Errorf("shadow blur = %v, alias should be preserved", shadowMap["blur"])
	}
	if shadowMap["spread"] != "{length.4xs}" {
		t.Errorf("shadow spread = %v, alias should be preserved", shadowMap["spread"])
	}
	if shadowMap["color"] != "#15151540" {
		t.Errorf("shadow color = %v, want #15151540", shadowMap["color"])
	}

	// Canonicalization changed offsetX and color, so the exact original shadow
	// object is recorded alongside the other canonicalized source values.
	wantShadowSource := map[string]any{
		"offsetX": "0",
		"offsetY": "{length.xs}",
		"blur":    "{length.sm}",
		"spread":  "{length.4xs}",
		"color":   "rgba(21, 21, 21, 0.25)",
	}
	assertSourceValue(t, shadow, wantShadowSource)

	assertCanonicalForms(t, tokens)
}

// assertCanonicalForms checks that literal values use draft DTCG forms. Alias
// references are exempt because the fixture never rewrites them.
func assertCanonicalForms(t *testing.T, tokens []*token.Token) {
	t.Helper()

	for _, problem := range canonicalFormProblems(tokens) {
		t.Error(problem)
	}
}

// canonicalFormProblems returns the canonical-form violations in tokens. A
// whole-value alias is exempt; a literal that merely embeds a reference is not.
func canonicalFormProblems(tokens []*token.Token) []string {
	var problems []string
	for _, tok := range tokens {
		if rhdsAliasForm.MatchString(tok.Value) {
			continue
		}
		switch tok.Type {
		case token.TypeNumber:
			if _, ok := tok.RawValue.(float64); !ok {
				problems = append(problems, fmt.Sprintf("number token %q has non-numeric value %#v", tok.DotPath(), tok.RawValue))
			}
		case token.TypeDuration:
			if !rhdsDurationForm.MatchString(tok.Value) {
				problems = append(problems, fmt.Sprintf("duration token %q value = %q, want milliseconds", tok.DotPath(), tok.Value))
			}
		case token.TypeColor:
			if !rhdsColorForm.MatchString(tok.Value) {
				problems = append(problems, fmt.Sprintf("color token %q value = %q, want hex", tok.DotPath(), tok.Value))
			}
		case token.TypeDimension:
			if !rhdsDimensionForm.MatchString(tok.Value) {
				problems = append(problems, fmt.Sprintf("dimension token %q value = %q, want px or rem", tok.DotPath(), tok.Value))
			}
		}
	}
	return problems
}

func assertSourceValue(t *testing.T, tok *token.Token, want any) {
	t.Helper()

	ext, ok := tok.Extensions["com.rhds.mcp"].(map[string]any)
	if !ok {
		t.Fatalf("%s: missing com.rhds.mcp extension: %v", tok.DotPath(), tok.Extensions)
	}
	if got := ext["sourceValue"]; !reflect.DeepEqual(got, want) {
		t.Errorf("%s: sourceValue = %#v, want %#v", tok.DotPath(), got, want)
	}
}

func TestCanonicalFormsSkipAliases(t *testing.T) {
	tokens := []*token.Token{
		{Name: "alias-number", Path: []string{"alias-number"}, Type: token.TypeNumber, Value: "{opacity.50}", RawValue: "{opacity.50}"},
		{Name: "alias-duration", Path: []string{"alias-duration"}, Type: token.TypeDuration, Value: "{animation.speed}", RawValue: "{animation.speed}"},
	}
	if problems := canonicalFormProblems(tokens); len(problems) != 0 {
		t.Errorf("whole-value aliases produced problems: %v", problems)
	}
}

func TestCanonicalFormsRejectBraceContainingLiterals(t *testing.T) {
	tokens := []*token.Token{
		{Name: "bad-dimension", Path: []string{"bad-dimension"}, Type: token.TypeDimension, Value: "calc({length.xs})", RawValue: "calc({length.xs})"},
		{Name: "bad-duration", Path: []string{"bad-duration"}, Type: token.TypeDuration, Value: "calc({animation.speed})", RawValue: "calc({animation.speed})"},
	}
	if problems := canonicalFormProblems(tokens); len(problems) != 2 {
		t.Errorf("brace-embedded literals produced %d problems, want 2: %v", len(problems), problems)
	}
}

func TestRHDSFixture_GroupExtensionsAndProvenance(t *testing.T) {
	raw := loadRHDSRaw(t)

	if _, ok := raw["$schema"]; ok {
		t.Error("fixture must omit $schema so the parser auto-detects the draft schema")
	}

	provenance, ok := raw["$extensions"].(map[string]any)
	if !ok {
		t.Fatal("fixture is missing root $extensions provenance")
	}
	rhds, ok := provenance["com.rhds.mcp"].(map[string]any)
	if !ok {
		t.Fatalf("fixture provenance missing com.rhds.mcp: %v", provenance)
	}
	const wantIntegrity = "sha512-YyeRPr2ThFGclyky4PCruj9f7e15yP2xnT3ZTJeU64+k5ICb/YezaKjJs8Dc48WHDhINHCgMt+VecfM86GExuQ=="
	wantProvenance := map[string]any{
		"sourcePackage":         "@rhds/tokens",
		"sourceVersion":         "3.2.0",
		"sourceFile":            "json/rhds.tokens.json",
		"sourceIntegrity":       wantIntegrity,
		"publishedTokens":       float64(547),
		"normalizedSchema":      "draft",
		"styleDictionaryFields": "dropped",
		"generatedColorAliases": float64(160),
		"excludedLightDark":     float64(57),
		"recoveredTypes":        float64(18),
	}
	for key, want := range wantProvenance {
		if got := rhds[key]; got != want {
			t.Errorf("provenance %s = %v, want %v", key, got, want)
		}
	}

	mapping, ok := rhds["typeMapping"].(map[string]any)
	if !ok {
		t.Fatalf("typeMapping = %v, want object", rhds["typeMapping"])
	}
	for key, want := range map[string]any{
		"radius":         "dimension",
		"letter-spacing": "dimension",
		"line-height":    "number",
		"opacity":        "number",
	} {
		if got := mapping[key]; got != want {
			t.Errorf("typeMapping %s = %v, want %v", key, got, want)
		}
	}

	vendorTypes, ok := rhds["vendorTypes"].(map[string]any)
	if !ok {
		t.Fatalf("vendorTypes = %v, want object", rhds["vendorTypes"])
	}
	if got, want := vendorTypes["mediaQuery"], "RHDS vendor type not defined by DTCG"; got != want {
		t.Errorf("vendorTypes.mediaQuery = %v, want %q", got, want)
	}

	retained, ok := rhds["retainedGeneratedAliases"].([]any)
	if !ok || len(retained) != 2 {
		t.Fatalf("retainedGeneratedAliases = %v, want two entries", rhds["retainedGeneratedAliases"])
	}
	for i, want := range []string{"color.red-orange.60-rgb", "color.red-orange.60-hsl"} {
		if retained[i] != want {
			t.Errorf("retainedGeneratedAliases[%d] = %v, want %s", i, retained[i], want)
		}
	}

	normalization, ok := rhds["valueNormalization"].(map[string]any)
	if !ok {
		t.Fatalf("valueNormalization = %v, want object", rhds["valueNormalization"])
	}
	for _, key := range []string{"aliases", "duration", "number", "color", "dimension"} {
		if _, ok := normalization[key]; !ok {
			t.Errorf("valueNormalization is missing %q", key)
		}
	}

	groupFields := map[string]map[string]any{
		"color":               {"heading": "Global color"},
		"color.border":        {"heading": "Color", "order": float64(80)},
		"color.red-orange":    {"order": float64(301)},
		"color.status":        {"order": float64(500)},
		"border.radius":       {"heading": "Radius"},
		"border.width":        {"heading": "Width"},
		"breakpoint":          {"heading": "Breakpoints", "descriptionFile": "breakpoints.md"},
		"media":               {"heading": "Media Queries", "descriptionFile": "media.md"},
		"font":                {"heading": "Typography"},
		"font.family":         {"heading": "Font family"},
		"font.size.body-text": {"heading": "Font size (text)", "example": "Aa"},
		"font.weight":         {"heading": "Font weights"},
		"font.weight.heading": {"heading": "Font weights (headings)", "fontFamily": "var(--rh-font-family-heading)"},
		"line-height":         {"heading": "Line height", "example": "Line<br/>height"},
	}
	for groupPath, want := range groupFields {
		ux := groupUXExtension(t, raw, groupPath)
		for key, wantValue := range want {
			if got := ux[key]; got != wantValue {
				t.Errorf("group %s com.redhat.ux %s = %v, want %v", groupPath, key, got, wantValue)
			}
		}
	}

	groupDescriptions := map[string]string{
		"color":            "Color helps create consistent experiences",
		"color.red-orange": "reserved for danger",
		"font":             "typography page",
		"font.family":      "Red Hat Fonts",
	}
	for groupPath, want := range groupDescriptions {
		ux := groupUXExtension(t, raw, groupPath)
		description, ok := ux["description"].(string)
		if !ok {
			t.Errorf("group %s com.redhat.ux description = %v, want string", groupPath, ux["description"])
			continue
		}
		if !strings.Contains(description, want) {
			t.Errorf("group %s description = %q, want substring %q", groupPath, description, want)
		}
	}

	redOrangeExt, ok := rawGroup(t, raw, "color.red-orange")["$extensions"].(map[string]any)
	if !ok {
		t.Fatal("color.red-orange has no $extensions")
	}
	if redOrangeExt["isCrayon"] != true {
		t.Errorf("color.red-orange isCrayon = %v, want true", redOrangeExt["isCrayon"])
	}
}

// rawGroup returns the group at a dot-separated path, failing the test instead
// of panicking when the fixture has an unexpected shape.
func rawGroup(t *testing.T, raw map[string]any, path string) map[string]any {
	t.Helper()

	current := raw
	for _, part := range strings.Split(path, ".") {
		next, ok := current[part].(map[string]any)
		if !ok {
			t.Fatalf("group %s: %q is not an object", path, part)
		}
		current = next
	}
	return current
}

// groupUXExtension returns a group's com.redhat.ux extension, failing the test
// instead of panicking when it is missing or malformed.
func groupUXExtension(t *testing.T, raw map[string]any, path string) map[string]any {
	t.Helper()

	ext, ok := rawGroup(t, raw, path)["$extensions"].(map[string]any)
	if !ok {
		t.Fatalf("group %s has no $extensions", path)
	}
	ux, ok := ext["com.redhat.ux"].(map[string]any)
	if !ok {
		t.Fatalf("group %s extensions missing com.redhat.ux: %v", path, ext)
	}
	return ux
}

func TestRHDSFixture_NoStyleDictionaryFields(t *testing.T) {
	raw := loadRHDSRaw(t)

	if fields := styleDictionaryFields(raw); len(fields) != 0 {
		t.Errorf("Style Dictionary fields survived: %v", fields)
	}
}

// styleDictionaryFields returns the paths of Style Dictionary fields that
// survived normalization. Vendor $extensions subtrees, composite $value
// members, and token or group names that happen to match a field name are not
// violations. A com.style-dictionary namespace under $extensions is reported
// because the fixture contract requires those fields to be dropped, not moved.
func styleDictionaryFields(raw map[string]any) []string {
	var found []string
	var walk func(value any, path string)
	walk = func(value any, path string) {
		switch v := value.(type) {
		case map[string]any:
			_, parentIsToken := v["$value"]
			for key, child := range v {
				switch key {
				case "$value":
					continue
				case "$extensions":
					if ext, ok := child.(map[string]any); ok {
						if _, moved := ext["com.style-dictionary"]; moved {
							found = append(found, path+".$extensions.com.style-dictionary")
						}
					}
					continue
				}
				_, childIsObject := child.(map[string]any)
				if isStyleDictionaryField(key) && (parentIsToken || !childIsObject) {
					found = append(found, path+"."+key)
				}
				walk(child, path+"."+key)
			}
		case []any:
			for i, child := range v {
				walk(child, path+"["+strconv.Itoa(i)+"]")
			}
		}
	}
	walk(raw, "$")
	slices.Sort(found)
	return found
}

func isStyleDictionaryField(key string) bool {
	switch key {
	case "name", "attributes", "filePath", "isSource", "original", "path", "value", "comment":
		return true
	default:
		return false
	}
}

func TestStyleDictionaryFieldsScope(t *testing.T) {
	allowed := map[string]any{
		"$extensions": map[string]any{
			"com.example": map[string]any{"name": "vendor data", "path": []any{"a", "b"}},
		},
		"path": map[string]any{"$value": "#000000", "$type": "color"},
		"name": map[string]any{"$value": "#ffffff", "$type": "color"},
		"media": map[string]any{
			"sm": map[string]any{
				"$value": map[string]any{"name": "sm", "min-width": "768px"},
				"$type":  "mediaQuery",
			},
		},
	}
	if got := styleDictionaryFields(allowed); len(got) != 0 {
		t.Errorf("styleDictionaryFields flagged allowed data: %v", got)
	}

	survived := map[string]any{
		"color": map[string]any{
			"primary": map[string]any{
				"$value":   "#000000",
				"name":     "color-primary",
				"path":     []any{"color", "primary"},
				"value":    "177 56 11",
				"comment":  "Style Dictionary metadata",
				"original": map[string]any{"$value": "#000000", "$type": "color"},
			},
		},
	}
	got := styleDictionaryFields(survived)
	want := []string{
		"$.color.primary.comment",
		"$.color.primary.name",
		"$.color.primary.original",
		"$.color.primary.path",
		"$.color.primary.value",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("styleDictionaryFields = %v, want %v", got, want)
	}

	moved := map[string]any{
		"color": map[string]any{
			"primary": map[string]any{
				"$value": "#000000",
				"$extensions": map[string]any{
					"com.style-dictionary": map[string]any{"name": "color-primary"},
				},
			},
		},
	}
	if got := styleDictionaryFields(moved); !reflect.DeepEqual(got, []string{"$.color.primary.$extensions.com.style-dictionary"}) {
		t.Errorf("styleDictionaryFields moved = %v, want com.style-dictionary path", got)
	}
}

// TestRHDSFixture_ConsistencyValidatorAccepts shows that asimonim's
// consistency validator accepts the fixture. It does not prove the document is
// DTCG-valid: the fixture contains the RHDS vendor type "mediaQuery", and the
// validator does not check $type membership against a DTCG type registry.
func TestRHDSFixture_ConsistencyValidatorAccepts(t *testing.T) {
	data := testutil.LoadFixtureFile(t, rhdsFixtureFile)

	detected, err := schema.DetectVersion(data, nil)
	if err != nil {
		t.Fatalf("schema detection failed: %v", err)
	}
	if detected != schema.Draft {
		t.Errorf("detected schema = %s, want draft", detected)
	}

	if problems := validator.ValidateConsistency(data, detected); len(problems) != 0 {
		t.Errorf("validator rejected fixture: %v", problems)
	}
}
