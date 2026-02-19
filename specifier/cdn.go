/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package specifier

import (
	"fmt"
	"slices"
)

// CDN identifies a CDN provider for package specifier URL generation.
type CDN string

const (
	// CDNUnpkg uses unpkg.com. Supports npm specifiers only.
	CDNUnpkg CDN = "unpkg"

	// CDNEsmSh uses esm.sh. Supports both npm and jsr specifiers.
	CDNEsmSh CDN = "esm.sh"

	// CDNEsmRun uses esm.run. Supports npm specifiers only.
	CDNEsmRun CDN = "esm.run"

	// CDNJspm uses jspm.dev. Supports npm specifiers only.
	CDNJspm CDN = "jspm"

	// CDNJsdelivr uses cdn.jsdelivr.net. Supports npm specifiers only.
	CDNJsdelivr CDN = "jsdelivr"
)

// ValidCDNs returns the list of supported CDN provider names.
func ValidCDNs() []string {
	return []string{
		string(CDNUnpkg),
		string(CDNEsmSh),
		string(CDNEsmRun),
		string(CDNJspm),
		string(CDNJsdelivr),
	}
}

// ParseCDN parses a string into a CDN value.
// Returns an error if the string is not a recognized CDN provider.
func ParseCDN(s string) (CDN, error) {
	cdn := CDN(s)
	if !slices.Contains(ValidCDNs(), s) {
		return "", fmt.Errorf("unknown CDN provider %q, valid values: %v", s, ValidCDNs())
	}
	return cdn, nil
}

// CDNURL returns the CDN URL for a package specifier.
// The cdn parameter selects the CDN provider; an empty value defaults to unpkg.
// Returns ("", false) for local paths, specifiers without a file component,
// or CDN/specifier combinations that are not supported (e.g., jsr on unpkg).
func CDNURL(spec string, cdn CDN) (string, bool) {
	parsed := Parse(spec)
	if parsed.Package == "" || parsed.File == "" {
		return "", false
	}

	// Default to unpkg for backward compatibility
	if cdn == "" {
		cdn = CDNUnpkg
	}

	switch parsed.Kind {
	case KindNPM:
		return npmCDNURL(parsed, cdn)
	case KindJSR:
		return jsrCDNURL(parsed, cdn)
	default:
		return "", false
	}
}

// npmCDNURL returns a CDN URL for an npm specifier.
func npmCDNURL(parsed *Specifier, cdn CDN) (string, bool) {
	switch cdn {
	case CDNUnpkg:
		return "https://unpkg.com/" + parsed.Package + "/" + parsed.File, true
	case CDNEsmSh:
		return "https://esm.sh/" + parsed.Package + "/" + parsed.File, true
	case CDNEsmRun:
		return "https://esm.run/" + parsed.Package + "/" + parsed.File, true
	case CDNJspm:
		return "https://jspm.dev/" + parsed.Package + "/" + parsed.File, true
	case CDNJsdelivr:
		return "https://cdn.jsdelivr.net/npm/" + parsed.Package + "/" + parsed.File, true
	default:
		return "", false
	}
}

// jsrCDNURL returns a CDN URL for a jsr specifier.
// Only esm.sh supports jsr specifiers.
func jsrCDNURL(parsed *Specifier, cdn CDN) (string, bool) {
	switch cdn {
	case CDNEsmSh:
		return "https://esm.sh/jsr/" + parsed.Package + "/" + parsed.File, true
	default:
		return "", false
	}
}
