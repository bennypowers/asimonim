/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package js provides unified JavaScript/TypeScript formatting for design tokens.
// It supports ESM/CommonJS modules, TypeScript/JSDoc types, and simple/map output styles.
package js

import (
	"bennypowers.dev/asimonim/convert/formatter"
	"bennypowers.dev/asimonim/token"
)

// Module specifies the JavaScript module system.
type Module string

const (
	// ModuleESM uses ES Modules (default).
	ModuleESM Module = "esm"
	// ModuleCJS uses CommonJS.
	ModuleCJS Module = "cjs"
)

// Types specifies the type annotation system.
type Types string

const (
	// TypesTS uses TypeScript annotations (default).
	TypesTS Types = "ts"
	// TypesJSDoc uses JSDoc annotations.
	TypesJSDoc Types = "jsdoc"
)

// Export specifies what form the exports take.
type Export string

const (
	// ExportValues uses simple const value exports (default).
	ExportValues Export = "values"
	// ExportMap uses a TokenMap class.
	ExportMap Export = "map"
)

// MapMode specifies TokenMap output mode (only for StyleMap with --split-by).
type MapMode string

const (
	// MapModeFull outputs complete content (types + class + tokens).
	MapModeFull MapMode = ""
	// MapModeTypes outputs only shared types and base class.
	MapModeTypes MapMode = "types"
	// MapModeModule outputs a split module that imports from shared types.
	MapModeModule MapMode = "module"
)

// Options configures the JS formatter.
type Options struct {
	// Module specifies the module format: "esm" (default), "cjs".
	Module Module
	// Types specifies the type system: "ts" (default), "jsdoc".
	Types Types
	// Export specifies what form the exports take: "values" (default), "map".
	Export Export
	// MapMode specifies the map mode: "" (full), "types", "module".
	// Only used when Export is ExportMap.
	MapMode MapMode
	// TypesPath is the import path for shared types (used with MapModeModule).
	TypesPath string
	// ClassName is the class name for extended TokenMap (used with MapModeModule).
	ClassName string
}

// Formatter outputs JavaScript/TypeScript with configurable options.
type Formatter struct {
	opts Options
}

// New creates a new JS formatter with default options (ESM, TypeScript, value exports).
func New() *Formatter {
	return &Formatter{opts: Options{
		Module: ModuleESM,
		Types:  TypesTS,
		Export: ExportValues,
	}}
}

// NewWithOptions creates a new JS formatter with the specified options.
func NewWithOptions(opts Options) *Formatter {
	// Apply defaults
	if opts.Module == "" {
		opts.Module = ModuleESM
	}
	if opts.Types == "" {
		opts.Types = TypesTS
	}
	if opts.Export == "" {
		opts.Export = ExportValues
	}
	return &Formatter{opts: opts}
}

// Format converts tokens to JavaScript/TypeScript format.
func (f *Formatter) Format(tokens []*token.Token, opts formatter.Options) ([]byte, error) {
	switch f.opts.Export {
	case ExportMap:
		return f.formatMap(tokens, opts)
	default:
		return f.formatSimple(tokens, opts)
	}
}

// Extension returns the appropriate file extension for the configured options.
func (f *Formatter) Extension() string {
	switch {
	case f.opts.Module == ModuleCJS && f.opts.Types == TypesTS:
		return ".cts"
	case f.opts.Module == ModuleCJS && f.opts.Types == TypesJSDoc:
		return ".cjs"
	case f.opts.Types == TypesJSDoc:
		return ".js"
	default:
		return ".ts"
	}
}
