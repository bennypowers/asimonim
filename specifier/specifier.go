/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package specifier parses npm and jsr package specifiers.
package specifier

import (
	"regexp"
	"strings"
)

// Kind indicates the type of specifier.
type Kind int

const (
	// KindLocal is a local file path.
	KindLocal Kind = iota
	// KindNPM is an npm package specifier.
	KindNPM
	// KindJSR is a jsr package specifier.
	KindJSR
)

// Specifier represents a parsed package specifier.
type Specifier struct {
	// Kind is the type of specifier (local, npm, jsr).
	Kind Kind

	// Package is the package name (e.g., "@scope/pkg" or "pkg").
	Package string

	// File is the file path within the package.
	File string

	// Raw is the original specifier string.
	Raw string
}

var (
	// npmPattern matches npm:@scope/pkg/path, npm:pkg/path, or bare npm:pkg
	npmPattern = regexp.MustCompile(`^npm:(@[^/]+/[^/]+|[^/]+)(/.*)?$`)

	// jsrPattern matches jsr:@scope/pkg/path, jsr:pkg/path, or bare jsr:pkg
	jsrPattern = regexp.MustCompile(`^jsr:(@[^/]+/[^/]+|[^/]+)(/.*)?$`)
)

// Parse parses a specifier string into a Specifier struct.
func Parse(spec string) *Specifier {
	// Check for npm specifier
	if strings.HasPrefix(spec, "npm:") {
		matches := npmPattern.FindStringSubmatch(spec)
		if len(matches) == 3 {
			return &Specifier{
				Kind:    KindNPM,
				Package: matches[1],
				File:    strings.TrimPrefix(matches[2], "/"),
				Raw:     spec,
			}
		}
	}

	// Check for jsr specifier
	if strings.HasPrefix(spec, "jsr:") {
		matches := jsrPattern.FindStringSubmatch(spec)
		if len(matches) == 3 {
			return &Specifier{
				Kind:    KindJSR,
				Package: matches[1],
				File:    strings.TrimPrefix(matches[2], "/"),
				Raw:     spec,
			}
		}
	}

	// Local file path
	return &Specifier{
		Kind: KindLocal,
		File: spec,
		Raw:  spec,
	}
}

// IsPackageSpecifier returns true if the string is a valid npm or jsr specifier.
// It uses the same validation as Parse to ensure consistency.
func IsPackageSpecifier(spec string) bool {
	parsed := Parse(spec)
	return parsed.Kind == KindNPM || parsed.Kind == KindJSR
}

// IsNPM returns true if this is an npm specifier.
func (s *Specifier) IsNPM() bool {
	return s.Kind == KindNPM
}

// IsJSR returns true if this is a jsr specifier.
func (s *Specifier) IsJSR() bool {
	return s.Kind == KindJSR
}

// IsLocal returns true if this is a local file path.
func (s *Specifier) IsLocal() bool {
	return s.Kind == KindLocal
}
