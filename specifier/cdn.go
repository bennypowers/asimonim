/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package specifier

// CDNURL returns the unpkg.com URL for an npm: or jsr: specifier.
// Returns ("", false) for local paths or specifiers without a file component.
func CDNURL(spec string) (string, bool) {
	parsed := Parse(spec)
	if parsed.Kind == KindLocal {
		return "", false
	}
	if parsed.Package == "" || parsed.File == "" {
		return "", false
	}
	return "https://unpkg.com/" + parsed.Package + "/" + parsed.File, true
}
