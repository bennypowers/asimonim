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
	"bennypowers.dev/asimonim/convert/formatter/cts"
	"bennypowers.dev/asimonim/convert/formatter/dtcg"
	"bennypowers.dev/asimonim/convert/formatter/flatjson"
	"bennypowers.dev/asimonim/convert/formatter/scss"
	"bennypowers.dev/asimonim/convert/formatter/swift"
	"bennypowers.dev/asimonim/convert/formatter/typescript"
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

	// FormatTypeScript outputs a TypeScript ESM module with camelCase exports.
	FormatTypeScript Format = "typescript"

	// FormatCTS outputs a TypeScript CommonJS module with camelCase exports.
	FormatCTS Format = "cts"

	// FormatSCSS outputs SCSS variables with kebab-case names.
	FormatSCSS Format = "scss"

	// FormatCSS outputs CSS custom properties with :root selector.
	FormatCSS Format = "css"

	// FormatLitCSS outputs CSS custom properties wrapped in Lit's css template tag.
	FormatLitCSS Format = "lit-css"
)

// ValidFormats returns all valid format strings.
func ValidFormats() []string {
	return []string{
		string(FormatDTCG),
		string(FormatFlatJSON),
		string(FormatAndroid),
		string(FormatSwift),
		string(FormatTypeScript),
		string(FormatCTS),
		string(FormatSCSS),
		string(FormatCSS),
		string(FormatLitCSS),
	}
}

// ParseFormat converts a string to a Format.
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
	case "typescript", "ts":
		return FormatTypeScript, nil
	case "cts", "commonjs":
		return FormatCTS, nil
	case "scss", "sass":
		return FormatSCSS, nil
	case "css":
		return FormatCSS, nil
	case "lit-css", "lit":
		return FormatLitCSS, nil
	default:
		return "", fmt.Errorf("unknown format: %s (valid: %s)", s, strings.Join(ValidFormats(), ", "))
	}
}

// FormatTokens converts tokens to the specified output format.
func FormatTokens(tokens []*token.Token, format Format, opts Options) ([]byte, error) {
	fmtOpts := formatter.Options{
		Prefix:    opts.Prefix,
		Delimiter: opts.Delimiter,
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
	case FormatTypeScript:
		f = typescript.New()
	case FormatCTS:
		f = cts.New()
	case FormatSCSS:
		f = scss.New()
	case FormatCSS:
		f = css.NewWithOptions(css.Options{
			Options:   fmtOpts,
			Flavor:    css.FlavorPlain,
			LightDark: buildLightDarkConfig(opts),
		})
	case FormatLitCSS:
		f = css.NewWithOptions(css.Options{
			Options:   fmtOpts,
			Flavor:    css.FlavorLit,
			LightDark: buildLightDarkConfig(opts),
		})
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	return f.Format(tokens, fmtOpts)
}

// buildLightDarkConfig constructs a css.LightDarkConfig from Options.
func buildLightDarkConfig(opts Options) css.LightDarkConfig {
	config := css.LightDarkConfig{
		Enabled: opts.CSSLightDark,
	}

	if !config.Enabled {
		return config
	}

	// Parse patterns from string pairs (e.g., "on-light on-dark")
	if len(opts.CSSLightDarkPatterns) > 0 {
		for _, patternStr := range opts.CSSLightDarkPatterns {
			parts := strings.Fields(patternStr)
			if len(parts) == 2 {
				config.Patterns = append(config.Patterns, [2]string{parts[0], parts[1]})
			}
		}
	}

	// Use defaults if no patterns specified
	if len(config.Patterns) == 0 {
		config.Patterns = css.DefaultLightDarkPatterns()
	}

	return config
}
