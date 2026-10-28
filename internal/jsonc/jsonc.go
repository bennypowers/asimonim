/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

// Package jsonc normalizes JSONC input for the schema, parser, and LSP
// packages. It is internal: the public API is schema.NormalizeJSON.
package jsonc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	tidwalljsonc "github.com/tidwall/jsonc"
	"gopkg.in/yaml.v3"
)

// utf8BOM is the UTF-8 byte order mark.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// NormalizeJSON strips a UTF-8 BOM and JSONC comments and trailing commas
// from JSON content. It never preserves a YAML interpretation, so callers
// that decode the result with encoding/json should use it. Content that does
// not become valid JSON is returned unchanged.
func NormalizeJSON(content []byte) []byte {
	content = bytes.TrimPrefix(content, utf8BOM)
	if json.Valid(content) {
		return content
	}
	if !looksLikeJSON(content) {
		return content
	}
	normalized := tidwalljsonc.ToJSON(content)
	if !json.Valid(normalized) {
		return content
	}
	return normalized
}

// NormalizeContent strips a UTF-8 BOM and JSONC comments and trailing commas
// from content that begins as a JSON object or array. YAML documents are
// returned unchanged, including flow mappings whose scalar values contain
// "//". Callers that parse content with a YAML decoder should use this
// function, because that decoder cannot parse JSONC comments.
func NormalizeContent(content []byte) []byte {
	content = bytes.TrimPrefix(content, utf8BOM)
	if json.Valid(content) {
		return content
	}
	if !looksLikeJSON(content) {
		return content
	}
	normalized := tidwalljsonc.ToJSON(content)
	if !json.Valid(normalized) {
		return content
	}

	// A YAML flow mapping starting with '{' can also be valid JSONC. Prefer
	// YAML only when the two decoders disagree about the value and YAML did
	// not turn a bare comment into a map key or scalar. Disagreement alone is
	// ambiguous because YAML misreads a trailing "//" comment as a
	// null-valued key.
	var yamlProbe any
	if err := yaml.Unmarshal(content, &yamlProbe); err == nil {
		switch yamlProbe.(type) {
		case map[string]any, map[any]any, []any:
			if hasCommentArtifact(yamlProbe, content) {
				return normalized
			}
			var jsonProbe any
			if err := json.Unmarshal(normalized, &jsonProbe); err == nil &&
				reflect.DeepEqual(normalizeProbe(yamlProbe), jsonProbe) {
				return normalized
			}
			return content
		}
	}
	return normalized
}

// MaskJSONCComments replaces JSONC comment text with spaces while preserving
// line breaks and UTF-16 column width. Byte offsets can change, but UTF-16
// columns stay aligned with the original document, which is what LSP
// positions use.
func MaskJSONCComments(content []byte) []byte {
	var out bytes.Buffer
	out.Grow(len(content))
	inString := false
	escaped := false
	for i := 0; i < len(content); {
		c := content[i]
		if inString {
			if escaped {
				escaped = false
				out.WriteByte(c)
				i++
				continue
			}
			switch c {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			out.WriteByte(c)
			i++
			continue
		}
		if c == '"' {
			inString = true
			out.WriteByte(c)
			i++
			continue
		}
		if c == '/' && i+1 < len(content) && content[i+1] == '/' {
			out.WriteString("  ")
			i += 2
			for i < len(content) && content[i] != '\n' {
				r, size := utf8.DecodeRune(content[i:])
				writeUTF16Spaces(&out, r)
				i += size
			}
			continue
		}
		if c == '/' && i+1 < len(content) && content[i+1] == '*' {
			out.WriteString("  ")
			i += 2
			for i < len(content) {
				if content[i] == '*' && i+1 < len(content) && content[i+1] == '/' {
					out.WriteString("  ")
					i += 2
					break
				}
				if content[i] == '\n' {
					out.WriteByte('\n')
					i++
					continue
				}
				r, size := utf8.DecodeRune(content[i:])
				writeUTF16Spaces(&out, r)
				i += size
			}
			continue
		}
		out.WriteByte(c)
		i++
	}
	return out.Bytes()
}

// writeUTF16Spaces writes one space per UTF-16 code unit occupied by r, so
// masked comments keep the same UTF-16 column width as the original text.
func writeUTF16Spaces(out *bytes.Buffer, r rune) {
	n := utf16.RuneLen(r)
	if n < 1 {
		n = 1
	}
	for i := 0; i < n; i++ {
		out.WriteByte(' ')
	}
}

// hasCommentArtifact reports whether a YAML decode contains a key or scalar
// value that looks like a bare JSONC comment consumed as data. YAML has no
// "//" or "/* */" comment syntax, so a key or value carrying one outside
// double-quoted text means YAML misread a JSONC comment.
func hasCommentArtifact(v any, content []byte) bool {
	switch x := v.(type) {
	case string:
		return commentArtifactScalar(content, x)
	case map[string]any:
		for k, val := range x {
			if commentArtifactScalar(content, k) || hasCommentArtifact(val, content) {
				return true
			}
		}
	case map[any]any:
		for k, val := range x {
			if key, ok := k.(string); ok && commentArtifactScalar(content, key) {
				return true
			}
			if hasCommentArtifact(val, content) {
				return true
			}
		}
	case []any:
		for _, val := range x {
			if hasCommentArtifact(val, content) {
				return true
			}
		}
	}
	return false
}

// commentArtifactScalar reports whether s is a comment artifact that appears
// outside double-quoted text in content.
func commentArtifactScalar(content []byte, s string) bool {
	trimmed := strings.TrimSpace(s)
	if !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "/*") && !strings.Contains(trimmed, "/*") {
		return false
	}
	return occursOutsideQuotes(content, trimmed)
}

// occursOutsideQuotes reports whether value occurs in content at a position
// that is not inside a double-quoted string.
func occursOutsideQuotes(content []byte, value string) bool {
	if value == "" {
		return false
	}
	for offset := 0; offset < len(content); {
		idx := bytes.Index(content[offset:], []byte(value))
		if idx < 0 {
			return false
		}
		if !insideDoubleQuoted(content, offset+idx) {
			return true
		}
		offset += idx + 1
	}
	return false
}

// insideDoubleQuoted reports whether pos is inside a double-quoted string.
func insideDoubleQuoted(content []byte, pos int) bool {
	inString := false
	escaped := false
	for i := 0; i < pos; i++ {
		c := content[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch c {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
		}
	}
	return inString
}

// normalizeProbe converts YAML and JSON decode shapes to a common form so
// they can be compared with reflect.DeepEqual.
func normalizeProbe(v any) any {
	switch x := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(x))
		for k, val := range x {
			m[k] = normalizeProbe(val)
		}
		return m
	case map[any]any:
		m := make(map[string]any, len(x))
		for k, val := range x {
			m[fmt.Sprint(k)] = normalizeProbe(val)
		}
		return m
	case []any:
		s := make([]any, len(x))
		for i, val := range x {
			s[i] = normalizeProbe(val)
		}
		return s
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case uint64:
		return float64(x)
	default:
		return v
	}
}

// looksLikeJSON reports whether content begins with a JSON object or array,
// skipping leading whitespace and JSONC comments.
func looksLikeJSON(content []byte) bool {
	for i := 0; i < len(content); {
		switch content[i] {
		case ' ', '\t', '\n', '\r':
			i++
		case '/':
			switch {
			case i+1 < len(content) && content[i+1] == '/':
				for i < len(content) && content[i] != '\n' {
					i++
				}
			case i+1 < len(content) && content[i+1] == '*':
				end := bytes.Index(content[i+2:], []byte("*/"))
				if end < 0 {
					return false
				}
				i += 2 + end + 2
			default:
				return false
			}
		case '{', '[':
			return true
		default:
			return false
		}
	}
	return false
}
