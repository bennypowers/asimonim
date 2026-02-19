/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package specifier

import "testing"

func TestCDNURL(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		wantURL string
		wantOK  bool
	}{
		{
			name:    "npm scoped package",
			spec:    "npm:@rhds/tokens/json/rhds.tokens.json",
			wantURL: "https://unpkg.com/@rhds/tokens/json/rhds.tokens.json",
			wantOK:  true,
		},
		{
			name:    "npm unscoped package",
			spec:    "npm:some-tokens/tokens.json",
			wantURL: "https://unpkg.com/some-tokens/tokens.json",
			wantOK:  true,
		},
		{
			name:   "jsr scoped package",
			spec:   "jsr:@scope/pkg/tokens.json",
			wantOK: false,
		},
		{
			name:   "jsr unscoped package",
			spec:   "jsr:pkg/tokens.json",
			wantOK: false,
		},
		{
			name:   "local path",
			spec:   "tokens.json",
			wantOK: false,
		},
		{
			name:   "absolute local path",
			spec:   "/path/to/tokens.json",
			wantOK: false,
		},
		{
			name:   "relative local path",
			spec:   "./tokens.json",
			wantOK: false,
		},
		{
			name:   "npm without file",
			spec:   "npm:@rhds/tokens",
			wantOK: false,
		},
		{
			name:   "jsr without file",
			spec:   "jsr:@scope/pkg",
			wantOK: false,
		},
		{
			name:    "npm deep path",
			spec:    "npm:@scope/pkg/a/b/c.json",
			wantURL: "https://unpkg.com/@scope/pkg/a/b/c.json",
			wantOK:  true,
		},
		{
			name:    "npm versioned scoped package",
			spec:    "npm:@scope/pkg@1.2.3/tokens.json",
			wantURL: "https://unpkg.com/@scope/pkg@1.2.3/tokens.json",
			wantOK:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotOK := CDNURL(tt.spec)
			if gotOK != tt.wantOK {
				t.Errorf("CDNURL(%q) ok = %v, want %v", tt.spec, gotOK, tt.wantOK)
			}
			if gotURL != tt.wantURL {
				t.Errorf("CDNURL(%q) url = %q, want %q", tt.spec, gotURL, tt.wantURL)
			}
		})
	}
}
