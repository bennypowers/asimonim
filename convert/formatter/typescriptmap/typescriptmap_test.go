/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package typescriptmap_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/convert/formatter/typescriptmap"
	"bennypowers.dev/asimonim/parser"
	"bennypowers.dev/asimonim/resolver"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/testutil"
)

func TestFormat_Basic(t *testing.T) {
	runFixtureTest(t, "basic", formatter.Options{})
}

func TestFormat_WithPrefix(t *testing.T) {
	runFixtureTest(t, "with-prefix", formatter.Options{Prefix: "rh"})
}

func TestFormat_Empty(t *testing.T) {
	runFixtureTest(t, "empty", formatter.Options{})
}

func TestFormat_EscapesQuotes(t *testing.T) {
	runFixtureTest(t, "escapes-quotes", formatter.Options{})
}

func TestFormat_EscapesBackslash(t *testing.T) {
	runFixtureTest(t, "escapes-backslash", formatter.Options{})
}

// runFixtureTest runs a fixture-based test for the typescriptmap formatter.
func runFixtureTest(t *testing.T, fixtureName string, opts formatter.Options) {
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

	// Check for options.json to override opts
	if optData, err := mfs.ReadFile("/test/options.json"); err == nil {
		var fileOpts struct {
			Prefix    string `json:"prefix"`
			Delimiter string `json:"delimiter"`
		}
		if err := json.Unmarshal(optData, &fileOpts); err == nil {
			if fileOpts.Prefix != "" {
				opts.Prefix = fileOpts.Prefix
			}
			if fileOpts.Delimiter != "" {
				opts.Delimiter = fileOpts.Delimiter
			}
		}
	}

	f := typescriptmap.New()
	result, err := f.Format(tokens, opts)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	goldenRelPath := filepath.Join(fixturePath, "expected.ts")

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
