/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package formatter provides the interface and common utilities for token formatters.
package formatter

import (
	"sort"
	"strings"
	"unicode"

	"bennypowers.dev/asimonim/token"
)

// Formatter defines the interface for output formatters.
type Formatter interface {
	// Format converts tokens to the target format.
	Format(tokens []*token.Token, opts Options) ([]byte, error)
}

// Options configures formatter behavior.
type Options struct {
	// Prefix is added to output variable names.
	Prefix string

	// Delimiter is the separator for flattened keys.
	// Zero value is empty string; consuming code should set "-" if needed.
	Delimiter string
}

// ResolvedValue returns the resolved value for a token, falling back to raw or original value.
func ResolvedValue(tok *token.Token) any {
	if tok == nil {
		return nil
	}
	if tok.ResolvedValue != nil {
		return tok.ResolvedValue
	}
	if tok.RawValue != nil {
		return tok.RawValue
	}
	return tok.Value
}

// SortTokens returns a copy of tokens sorted by name.
func SortTokens(tokens []*token.Token) []*token.Token {
	sorted := make([]*token.Token, len(tokens))
	copy(sorted, tokens)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}

// GroupByType groups tokens by their type.
func GroupByType(tokens []*token.Token) map[string][]*token.Token {
	groups := make(map[string][]*token.Token)
	for _, tok := range tokens {
		groups[tok.Type] = append(groups[tok.Type], tok)
	}
	return groups
}

// ApplyPrefix adds a prefix to a name with the given delimiter.
func ApplyPrefix(name, prefix, delimiter string) string {
	if prefix == "" {
		return name
	}
	return prefix + delimiter + name
}

// ApplyPrefixCamel applies a prefix in camelCase style.
func ApplyPrefixCamel(name, prefix string) string {
	if prefix == "" {
		return name
	}
	if name == "" {
		return ToCamelCase(prefix)
	}
	return ToCamelCase(prefix) + strings.ToUpper(name[:1]) + name[1:]
}

// ToCamelCase converts a string to camelCase.
func ToCamelCase(s string) string {
	words := SplitIntoWords(s)
	if len(words) == 0 {
		return ""
	}

	result := strings.ToLower(words[0])
	for i := 1; i < len(words); i++ {
		if len(words[i]) > 0 {
			result += strings.ToUpper(words[i][:1]) + strings.ToLower(words[i][1:])
		}
	}
	return result
}

// ToPascalCase converts a string to PascalCase.
func ToPascalCase(s string) string {
	words := SplitIntoWords(s)
	var result string
	for _, word := range words {
		if len(word) > 0 {
			result += strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return result
}

// ToSnakeCase converts a string to snake_case.
func ToSnakeCase(s string) string {
	words := SplitIntoWords(s)
	return strings.ToLower(strings.Join(words, "_"))
}

// ToKebabCase converts a string to kebab-case.
func ToKebabCase(s string) string {
	words := SplitIntoWords(s)
	return strings.ToLower(strings.Join(words, "-"))
}

// ToTitleCase converts a string to Title Case.
func ToTitleCase(s string) string {
	words := SplitIntoWords(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// SplitIntoWords splits a string on hyphens, underscores, dots, and camelCase boundaries.
func SplitIntoWords(s string) []string {
	var words []string
	var current strings.Builder

	for i, r := range s {
		if r == '-' || r == '_' || r == '.' || r == ' ' {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		} else if unicode.IsUpper(r) && i > 0 {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			current.WriteRune(r)
		} else {
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

// EscapeXML escapes special XML characters.
func EscapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
