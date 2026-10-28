/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

package jsonc_test

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf16"

	"bennypowers.dev/asimonim/internal/jsonc"
)

func TestMaskJSONCComments(t *testing.T) {
	tests := []struct {
		name    string
		comment string
	}{
		{"BMP multibyte comment", "/* héllo */"},
		{"emoji comment", "/* 😀 */"},
		{"non-BMP symbol comment", "/* 𝄞 */"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.comment + ` {"color": "#fff"} // tail`)
			got := jsonc.MaskJSONCComments(input)

			if strings.Contains(string(got), "tail") || strings.Contains(string(got), strings.TrimSpace(strings.Trim(tt.comment, "/* "))) {
				t.Errorf("MaskJSONCComments() = %q, want comment text masked", got)
			}
			if !strings.Contains(string(got), `{"color": "#fff"}`) {
				t.Errorf("MaskJSONCComments() = %q, want JSON body preserved", got)
			}

			origKey := strings.Index(string(input), `{"color"`)
			maskedKey := strings.Index(string(got), `{"color"`)
			if origKey < 0 || maskedKey < 0 {
				t.Fatal("expected JSON body in input and output")
			}
			origUnits := utf16Width(string(input[:origKey]))
			maskedUnits := utf16Width(string(got[:maskedKey]))
			if origUnits != maskedUnits {
				t.Errorf("comment mask shifted UTF-16 columns: original %d, masked %d", origUnits, maskedUnits)
			}
		})
	}
}

// utf16Width returns the number of UTF-16 code units in s.
func utf16Width(s string) int {
	return len(utf16.Encode([]rune(s)))
}

func TestNormalizeContent(t *testing.T) {
	t.Run("YAML flow mapping keeps // as text", func(t *testing.T) {
		input := []byte(`{"$value": 2 // px}`)
		if got := string(jsonc.NormalizeContent(input)); got != string(input) {
			t.Errorf("NormalizeContent() = %q, want unchanged %q", got, input)
		}
	})

	t.Run("multi-line YAML flow mapping keeps // as text", func(t *testing.T) {
		input := []byte("{\"$value\": 2 // px\n}")
		if got := string(jsonc.NormalizeContent(input)); got != string(input) {
			t.Errorf("NormalizeContent() = %q, want unchanged %q", got, input)
		}
	})

	t.Run("trailing comment after nested object is normalized", func(t *testing.T) {
		input := []byte(`{"name":"p","asimonim":{"prefix":"rh"}, // note
}`)
		got := jsonc.NormalizeContent(input)
		if !json.Valid(got) {
			t.Errorf("NormalizeContent() = %q, want valid JSON", got)
		}
	})

	t.Run("JSONC trailing comma is normalized", func(t *testing.T) {
		input := []byte(`{"asimonim":{"prefix":"rh",},}`)
		got := jsonc.NormalizeContent(input)
		if !json.Valid(got) {
			t.Errorf("NormalizeContent() = %q, want valid JSON", got)
		}
	})

	t.Run("numeric JSONC trailing comma is normalized", func(t *testing.T) {
		input := []byte(`{"count": 1,}`)
		got := jsonc.NormalizeContent(input)
		if !json.Valid(got) {
			t.Errorf("NormalizeContent() = %q, want valid JSON", got)
		}
	})

	t.Run("quoted line comment in YAML string is preserved", func(t *testing.T) {
		input := []byte("{\"header\":\"// keep\",\"prefix\": 2 // px\n}")
		if got := string(jsonc.NormalizeContent(input)); got != string(input) {
			t.Errorf("NormalizeContent() = %q, want unchanged %q", got, input)
		}
	})

	t.Run("quoted block comment in YAML string is preserved", func(t *testing.T) {
		input := []byte("{\"header\":\"/* keep */\",\"prefix\": 2 // px\n}")
		if got := string(jsonc.NormalizeContent(input)); got != string(input) {
			t.Errorf("NormalizeContent() = %q, want unchanged %q", got, input)
		}
	})

	t.Run("inline block comment in a value is normalized", func(t *testing.T) {
		input := []byte(`{"theme":{"$extends": /* base */ "#/base"}}`)
		got := jsonc.NormalizeContent(input)
		if !json.Valid(got) {
			t.Errorf("NormalizeContent() = %q, want valid JSON", got)
		}
		if strings.Contains(string(got), "/*") {
			t.Errorf("NormalizeContent() = %q, want block comment removed", got)
		}
	})

	t.Run("JSONC comments are removed", func(t *testing.T) {
		input := []byte("// header\n{\"color\": {\"$value\": \"#fff\"}}")
		got := string(jsonc.NormalizeContent(input))
		if strings.Contains(got, "//") {
			t.Errorf("NormalizeContent() = %q, want comments removed", got)
		}
		if !strings.Contains(got, `"color"`) {
			t.Errorf("NormalizeContent() = %q, want color key preserved", got)
		}
	})
}
