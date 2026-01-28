/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package convert

import (
	"fmt"
	"strings"

	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/convert/formatter/android"
	"bennypowers.dev/asimonim/convert/formatter/css"
	"bennypowers.dev/asimonim/convert/formatter/dtcg"
	"bennypowers.dev/asimonim/convert/formatter/flatjson"
	"bennypowers.dev/asimonim/convert/formatter/js"
	"bennypowers.dev/asimonim/convert/formatter/scss"
	"bennypowers.dev/asimonim/convert/formatter/snippets"
	"bennypowers.dev/asimonim/convert/formatter/swift"
	"bennypowers.dev/asimonim/token"
)

// Format represents an output format for token serialization.
type Format string

const (
	// FormatDTCG outputs DTCG-compliant JSON (default).
	FormatDTCG Format = "dtcg"

	// FormatFlatJSON outputs flat key-value JSON.
	FormatFlatJSON Format = "json"

	// FormatAndroid outputs Android-style XML resources.
	FormatAndroid Format = "android"

	// FormatSwift outputs iOS Swift constants.
	FormatSwift Format = "swift"

	// FormatJS outputs JavaScript/TypeScript modules.
	// Use JSModule, JSTypes, and JSExport options to customize output.
	FormatJS Format = "js"

	// FormatSCSS outputs SCSS variables with kebab-case names.
	FormatSCSS Format = "scss"

	// FormatCSS outputs CSS custom properties.
	// Use CSSSelector and CSSModule options to customize output.
	FormatCSS Format = "css"

	// FormatSnippets outputs editor snippets (VSCode, TextMate, etc).
	// Use SnippetType option to specify the output format.
	FormatSnippets Format = "snippets"
)

// ValidFormats returns all valid format strings.
func ValidFormats() []string {
	return []string{
		string(FormatDTCG),
		string(FormatFlatJSON),
		string(FormatAndroid),
		string(FormatSwift),
		string(FormatJS),
		string(FormatSCSS),
		string(FormatCSS),
		string(FormatSnippets),
	}
}

// ParseFormat converts a string to a Format.
// Note: For format aliases (typescript, cts, typescript-map), the caller must also
// set appropriate JSModule, JSTypes, and JSStyle options.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "dtcg", "":
		return FormatDTCG, nil
	case "json", "flat", "flat-json":
		return FormatFlatJSON, nil
	case "android", "xml":
		return FormatAndroid, nil
	case "swift", "ios":
		return FormatSwift, nil
	case "js", "javascript":
		return FormatJS, nil
	case "scss", "sass":
		return FormatSCSS, nil
	case "css":
		return FormatCSS, nil
	case "snippets":
		return FormatSnippets, nil
	default:
		return "", fmt.Errorf("unknown format: %s (valid: %s)", s, strings.Join(ValidFormats(), ", "))
	}
}

// FormatTokens converts tokens to the specified output format.
func FormatTokens(tokens []*token.Token, format Format, opts Options) ([]byte, error) {
	fmtOpts := formatter.Options{
		Prefix:    opts.Prefix,
		Delimiter: opts.Delimiter,
		Header:    opts.Header,
	}

	var f formatter.Formatter
	switch format {
	case FormatDTCG:
		f = dtcg.New(func(t []*token.Token) map[string]any {
			return Serialize(t, opts)
		})
	case FormatFlatJSON:
		f = flatjson.New()
	case FormatAndroid:
		f = android.New()
	case FormatSwift:
		f = swift.New()
	case FormatJS:
		f = js.NewWithOptions(js.Options{
			Module:    js.Module(opts.JSModule),
			Types:     js.Types(opts.JSTypes),
			Export:    js.Export(opts.JSExport),
			MapMode:   js.MapMode(opts.JSMapMode),
			TypesPath: opts.JSMapTypesPath,
			ClassName: opts.JSMapClassName,
		})
	case FormatSCSS:
		f = scss.New()
	case FormatCSS:
		f = css.NewWithOptions(css.Options{
			Selector: css.Selector(opts.CSSSelector),
			Module:   css.Module(opts.CSSModule),
		})
	case FormatSnippets:
		f = snippets.NewWithOptions(snippets.Options{
			Type: snippets.Type(opts.SnippetType),
		})
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	return f.Format(tokens, fmtOpts)
}
