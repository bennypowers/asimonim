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
