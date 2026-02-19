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
		cdn     CDN
		wantURL string
		wantOK  bool
	}{
		// Zero-value CDN defaults to unpkg
		{
			name:    "zero-value CDN defaults to unpkg",
			spec:    "npm:@rhds/tokens/json/rhds.tokens.json",
			cdn:     "",
			wantURL: "https://unpkg.com/@rhds/tokens/json/rhds.tokens.json",
			wantOK:  true,
		},

		// unpkg
		{
			name:    "unpkg npm scoped package",
			spec:    "npm:@rhds/tokens/json/rhds.tokens.json",
			cdn:     CDNUnpkg,
			wantURL: "https://unpkg.com/@rhds/tokens/json/rhds.tokens.json",
			wantOK:  true,
		},
		{
			name:    "unpkg npm unscoped package",
			spec:    "npm:some-tokens/tokens.json",
			cdn:     CDNUnpkg,
			wantURL: "https://unpkg.com/some-tokens/tokens.json",
			wantOK:  true,
		},
		{
			name:    "unpkg npm versioned scoped",
			spec:    "npm:@scope/pkg@1.2.3/tokens.json",
			cdn:     CDNUnpkg,
			wantURL: "https://unpkg.com/@scope/pkg@1.2.3/tokens.json",
			wantOK:  true,
		},
		{
			name:    "unpkg npm deep path",
			spec:    "npm:@scope/pkg/a/b/c.json",
			cdn:     CDNUnpkg,
			wantURL: "https://unpkg.com/@scope/pkg/a/b/c.json",
			wantOK:  true,
		},
		{
			name:   "unpkg jsr not supported",
			spec:   "jsr:@scope/pkg/tokens.json",
			cdn:    CDNUnpkg,
			wantOK: false,
		},

		// esm.sh
		{
			name:    "esm.sh npm scoped",
			spec:    "npm:@rhds/tokens/json/rhds.tokens.json",
			cdn:     CDNEsmSh,
			wantURL: "https://esm.sh/@rhds/tokens/json/rhds.tokens.json",
			wantOK:  true,
		},
		{
			name:    "esm.sh npm unscoped",
			spec:    "npm:some-tokens/tokens.json",
			cdn:     CDNEsmSh,
			wantURL: "https://esm.sh/some-tokens/tokens.json",
			wantOK:  true,
		},
		{
			name:    "esm.sh jsr scoped",
			spec:    "jsr:@scope/pkg/tokens.json",
			cdn:     CDNEsmSh,
			wantURL: "https://esm.sh/jsr/@scope/pkg/tokens.json",
			wantOK:  true,
		},
		{
			name:   "esm.sh jsr unscoped not supported",
			spec:   "jsr:pkg/tokens.json",
			cdn:    CDNEsmSh,
			wantOK: false,
		},

		// esm.run
		{
			name:    "esm.run npm scoped",
			spec:    "npm:@rhds/tokens/json/rhds.tokens.json",
			cdn:     CDNEsmRun,
			wantURL: "https://esm.run/@rhds/tokens/json/rhds.tokens.json",
			wantOK:  true,
		},
		{
			name:   "esm.run jsr not supported",
			spec:   "jsr:@scope/pkg/tokens.json",
			cdn:    CDNEsmRun,
			wantOK: false,
		},

		// jspm
		{
			name:    "jspm npm scoped",
			spec:    "npm:@rhds/tokens/json/rhds.tokens.json",
			cdn:     CDNJspm,
			wantURL: "https://ga.jspm.io/npm:@rhds/tokens/json/rhds.tokens.json",
			wantOK:  true,
		},
		{
			name:   "jspm jsr not supported",
			spec:   "jsr:@scope/pkg/tokens.json",
			cdn:    CDNJspm,
			wantOK: false,
		},

		// jsdelivr
		{
			name:    "jsdelivr npm scoped",
			spec:    "npm:@rhds/tokens/json/rhds.tokens.json",
			cdn:     CDNJsdelivr,
			wantURL: "https://cdn.jsdelivr.net/npm/@rhds/tokens/json/rhds.tokens.json",
			wantOK:  true,
		},
		{
			name:   "jsdelivr jsr not supported",
			spec:   "jsr:@scope/pkg/tokens.json",
			cdn:    CDNJsdelivr,
			wantOK: false,
		},

		// Non-package specifiers always return false
		{
			name:   "local path",
			spec:   "tokens.json",
			cdn:    CDNUnpkg,
			wantOK: false,
		},
		{
			name:   "absolute local path",
			spec:   "/path/to/tokens.json",
			cdn:    CDNUnpkg,
			wantOK: false,
		},
		{
			name:   "relative local path",
			spec:   "./tokens.json",
			cdn:    CDNUnpkg,
			wantOK: false,
		},

		// Specifiers without file component
		{
			name:   "npm without file",
			spec:   "npm:@rhds/tokens",
			cdn:    CDNUnpkg,
			wantOK: false,
		},
		{
			name:   "jsr without file",
			spec:   "jsr:@scope/pkg",
			cdn:    CDNEsmSh,
			wantOK: false,
		},
		{
			name:   "npm versioned without file",
			spec:   "npm:@scope/pkg@1.2.3",
			cdn:    CDNUnpkg,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotOK := CDNURL(tt.spec, tt.cdn)
			if gotOK != tt.wantOK {
				t.Errorf("CDNURL(%q, %q) ok = %v, want %v", tt.spec, tt.cdn, gotOK, tt.wantOK)
			}
			if gotURL != tt.wantURL {
				t.Errorf("CDNURL(%q, %q) url = %q, want %q", tt.spec, tt.cdn, gotURL, tt.wantURL)
			}
		})
	}
}

func TestParseCDN(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    CDN
		wantErr bool
	}{
		{name: "unpkg", input: "unpkg", want: CDNUnpkg},
		{name: "esm.sh", input: "esm.sh", want: CDNEsmSh},
		{name: "esm.run", input: "esm.run", want: CDNEsmRun},
		{name: "jspm", input: "jspm", want: CDNJspm},
		{name: "jsdelivr", input: "jsdelivr", want: CDNJsdelivr},
		{name: "unknown", input: "cloudflare", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCDN(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCDN(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseCDN(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidCDNs(t *testing.T) {
	cdns := ValidCDNs()
	if len(cdns) != 5 {
		t.Errorf("ValidCDNs() returned %d entries, want 5", len(cdns))
	}
}
