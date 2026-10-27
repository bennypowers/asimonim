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
// before asimonim sees it:
//
//   - $value comes from original.$value so aliases survive
//   - $type is recovered from attributes with an explicit mapping
//   - $description, $deprecated, and $extensions are kept
//   - Style Dictionary fields are dropped
//   - literal values are canonicalized to draft DTCG forms (duration in ms,
//     number percentages as unitless numbers, literal colors as hex, shadow
//     dimensions with explicit units) while alias references are untouched
//
// Generated tokens in the source document:
//
//   - 160 -rgb and -hsl color variants are kept as color tokens. Their
//     original values are aliases to the base color, so the normalized tokens
//     resolve through those aliases.
//   - 57 light-dark(var(...)) composite theme tokens are excluded. Their
//     original values are arrays of per-theme aliases, which is not a DTCG
//     color value. The @rhds/mcp generator keeps them in a separate computed
//     theme map instead.
//
// The fixture carries the same counts in its root $extensions so the
// contract is inspectable without this test file.
package integration_test

import (
	"encoding/json"
	"reflect"
	"regexp"
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

	// The four source groups without $type must be recovered explicitly.
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
	for _, name := range []string{"color-red-orange-60", "color.red-orange.60", "--rh-color-red-orange-60"} {
		tok, ok := tokenMap.Get(name)
		if !ok {
			t.Errorf("token map lookup %q missed", name)
			continue
		}
		if tok.Prefix != "rh" {
			t.Errorf("token map lookup %q prefix = %q, want rh", name, tok.Prefix)
		}
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

	// Composite values survive as composite values. DTCG does not allow
	// resolving nested aliases inside a shadow, so they remain references.
	shadow := testutil.TokenByPath(t, tokens, "box-shadow.md")
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
	if !strings.Contains(deprecated.DeprecationMessage, "Use color transforms instead") {
		t.Errorf("deprecation message = %q", deprecated.DeprecationMessage)
	}

	family := testutil.TokenByPath(t, tokens, "font.family.heading")
	familyValue, ok := family.RawValue.([]any)
	if !ok || len(familyValue) != 5 {
		t.Errorf("font family raw value = %#v, want 5-element array", family.RawValue)
	}

	media := testutil.TokenByPath(t, tokens, "media.sm")
	mediaValue, ok := media.RawValue.(map[string]any)
	if !ok {
		t.Fatalf("media raw value type = %T, want map", media.RawValue)
	}
	if mediaValue["min-width"] != "{breakpoint.sm}" {
		t.Errorf("media min-width = %v", mediaValue["min-width"])
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
	if shadowMap["color"] != "#15151540" {
		t.Errorf("shadow color = %v, want #15151540", shadowMap["color"])
	}

	durationForm := regexp.MustCompile(`^\d+(?:\.\d+)?ms$`)
	colorForm := regexp.MustCompile(`^#[0-9A-Fa-f]{6}([0-9A-Fa-f]{2})?$`)
	dimensionForm := regexp.MustCompile(`^-?\d+(?:\.\d+)?(?:px|rem)$`)

	for _, tok := range tokens {
		switch tok.Type {
		case token.TypeNumber:
			if _, ok := tok.RawValue.(float64); !ok {
				t.Errorf("number token %q has non-numeric value %#v", tok.DotPath(), tok.RawValue)
			}
		case token.TypeDuration:
			if !durationForm.MatchString(tok.Value) {
				t.Errorf("duration token %q value = %q, want milliseconds", tok.DotPath(), tok.Value)
			}
		case token.TypeColor:
			if strings.HasPrefix(tok.Value, "{") {
				continue
			}
			if !colorForm.MatchString(tok.Value) {
				t.Errorf("color token %q value = %q, want hex", tok.DotPath(), tok.Value)
			}
		case token.TypeDimension:
			if strings.HasPrefix(tok.Value, "{") {
				continue
			}
			if !dimensionForm.MatchString(tok.Value) {
				t.Errorf("dimension token %q value = %q, want px or rem", tok.DotPath(), tok.Value)
			}
		}
	}
}

func assertSourceValue(t *testing.T, tok *token.Token, want string) {
	t.Helper()

	ext, ok := tok.Extensions["com.rhds.mcp"].(map[string]any)
	if !ok {
		t.Fatalf("%s: missing com.rhds.mcp extension: %v", tok.DotPath(), tok.Extensions)
	}
	if ext["sourceValue"] != want {
		t.Errorf("%s: sourceValue = %v, want %s", tok.DotPath(), ext["sourceValue"], want)
	}
}

func TestRHDSFixture_GroupExtensionsAndProvenance(t *testing.T) {
	raw := loadRHDSRaw(t)

	provenance, ok := raw["$extensions"].(map[string]any)
	if !ok {
		t.Fatal("fixture is missing root $extensions provenance")
	}
	rhds, ok := provenance["com.rhds.mcp"].(map[string]any)
	if !ok {
		t.Fatalf("fixture provenance missing com.rhds.mcp: %v", provenance)
	}
	wantProvenance := map[string]any{
		"sourcePackage":         "@rhds/tokens",
		"sourceVersion":         "3.2.0",
		"normalizedSchema":      "draft",
		"generatedColorAliases": float64(160),
		"excludedLightDark":     float64(57),
		"recoveredTypes":        float64(18),
	}
	for key, want := range wantProvenance {
		if got := rhds[key]; got != want {
			t.Errorf("provenance %s = %v, want %v", key, got, want)
		}
	}
	if integrity, _ := rhds["sourceIntegrity"].(string); !strings.HasPrefix(integrity, "sha512-") {
		t.Errorf("sourceIntegrity = %q, want sha512- prefix", integrity)
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

	groups := map[string]map[string]any{
		"color":            raw["color"].(map[string]any),
		"color.red-orange": raw["color"].(map[string]any)["red-orange"].(map[string]any),
		"border.radius":    raw["border"].(map[string]any)["radius"].(map[string]any),
		"font.family":      raw["font"].(map[string]any)["family"].(map[string]any),
	}
	for groupPath, group := range groups {
		ext, ok := group["$extensions"].(map[string]any)
		if !ok {
			t.Errorf("group %s has no $extensions", groupPath)
			continue
		}
		if _, ok := ext["com.redhat.ux"]; !ok {
			t.Errorf("group %s extensions missing com.redhat.ux: %v", groupPath, ext)
		}
	}

	redOrangeExt := raw["color"].(map[string]any)["red-orange"].(map[string]any)["$extensions"].(map[string]any)
	if redOrangeExt["isCrayon"] != true {
		t.Errorf("color.red-orange isCrayon = %v, want true", redOrangeExt["isCrayon"])
	}
}

func TestRHDSFixture_NoStyleDictionaryFields(t *testing.T) {
	raw := loadRHDSRaw(t)

	forbidden := map[string]bool{
		"attributes": true,
		"filePath":   true,
		"isSource":   true,
		"original":   true,
		"path":       true,
	}
	var walk func(value any, path string)
	walk = func(value any, path string) {
		switch v := value.(type) {
		case map[string]any:
			for key, child := range v {
				if forbidden[key] {
					t.Errorf("Style Dictionary field %q survived at %s", key, path)
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
}

func TestRHDSFixture_ValidatorAccepts(t *testing.T) {
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
