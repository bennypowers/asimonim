/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package android provides Android XML resource formatting for design tokens.
package android

import (
	"fmt"
	"strings"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/token"
)

// Formatter outputs Android-style XML resources.
type Formatter struct{}

// New creates a new Android formatter.
func New() *Formatter {
	return &Formatter{}
}

// Format converts tokens to Android XML resource format.
func (f *Formatter) Format(tokens []*token.Token, opts formatter.Options) ([]byte, error) {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="utf-8"?>`)
	sb.WriteString("\n<resources>\n")

	sorted := formatter.SortTokens(tokens)

	for _, tok := range sorted {
		baseName := formatter.ToSnakeCase(strings.Join(tok.Path, "_"))
		name := formatter.ApplyPrefix(baseName, opts.Prefix, "_")
		value := formatter.ResolvedValue(tok)
		xmlType := xmlType(tok.Type)

		sb.WriteString(fmt.Sprintf("    <%s name=\"%s\">%s</%s>\n",
			xmlType, formatter.EscapeXML(name), formatter.EscapeXML(fmt.Sprintf("%v", value)), xmlType))
	}

	sb.WriteString("</resources>\n")
	return []byte(sb.String()), nil
}

func xmlType(tokenType string) string {
	switch tokenType {
	case token.TypeColor:
		return "color"
	case token.TypeDimension:
		return "dimen"
	case token.TypeNumber:
		return "integer"
	case token.TypeString, token.TypeFontFamily:
		return "string"
	default:
		return "string"
	}
}
