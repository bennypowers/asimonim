/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package config provides configuration loading for design tokens tooling.
package config

import (
	"encoding/json"

	"gopkg.in/yaml.v3"

	"bennypowers.dev/asimonim/parser"
	"bennypowers.dev/asimonim/schema"
)

// Config represents the design tokens configuration.
type Config struct {
	// Prefix is the global CSS variable prefix.
	Prefix string `yaml:"prefix" json:"prefix"`

	// Files specifies token files to load (paths or specs).
	Files []FileSpec `yaml:"files" json:"files"`

	// GroupMarkers are token names that can be both tokens and groups (draft only).
	GroupMarkers []string `yaml:"groupMarkers" json:"groupMarkers"`

	// Schema forces a specific schema version (optional).
	// Valid values: "draft", "v2025.10"
	Schema string `yaml:"schema" json:"schema"`

	// Formats contains format-specific configuration.
	Formats FormatsConfig `yaml:"formats" json:"formats"`

	// Header is the file header to prepend to output.
	// Can be a string or a file path prefixed with "@" (e.g., "@LICENSE_HEADER.txt").
	Header string `yaml:"header" json:"header"`

	// Outputs specifies multiple output files to generate.
	// When set, the convert command will generate all specified outputs in a single pass.
	Outputs []OutputSpec `yaml:"outputs" json:"outputs"`
}

// FormatsConfig contains format-specific configuration.
type FormatsConfig struct {
	// CSS contains CSS-specific output configuration.
	CSS CSSConfig `yaml:"css" json:"css"`
}

// CSSConfig contains CSS-specific output configuration.
type CSSConfig struct {
	// Placeholder for future CSS-specific options.
}

// OutputSpec represents a single output file specification.
type OutputSpec struct {
	// Format is the output format (required).
	// Valid values: dtcg, json, android, swift, typescript, cts, scss, css, lit-css
	Format string `yaml:"format" json:"format"`

	// Path is the output file path (required).
	// Supports template variables: {group} for split key.
	// Example: "js/{group}.ts" generates "js/color.ts", "js/animation.ts", etc.
	Path string `yaml:"path" json:"path"`

	// Prefix overrides the global prefix for this output.
	Prefix string `yaml:"prefix" json:"prefix"`

	// Flatten produces a shallow structure with delimiter-separated keys.
	Flatten bool `yaml:"flatten" json:"flatten"`

	// Delimiter is the separator for flattened keys.
	Delimiter string `yaml:"delimiter" json:"delimiter"`

	// SplitBy specifies how to split tokens into separate files.
	// Valid values:
	//   - "topLevel" or "" (default): split by first path segment
	//   - "type": split by token $type
	//   - "path[N]": split by Nth path segment (0-indexed)
	// Only applies when Path contains {group} template.
	SplitBy string `yaml:"splitBy" json:"splitBy"`
}

// FileSpec represents a token file specification.
// It can be specified as a simple string path or as an object with overrides.
type FileSpec struct {
	// Path is the file path (supports globs and npm: protocol).
	Path string `yaml:"path" json:"path"`

	// Prefix overrides the global CSS variable prefix for this file.
	Prefix string `yaml:"prefix" json:"prefix"`

	// GroupMarkers overrides the global group markers for this file.
	GroupMarkers []string `yaml:"groupMarkers" json:"groupMarkers"`
}

// UnmarshalYAML handles both string and object forms for FileSpec.
func (f *FileSpec) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		f.Path = node.Value
		return nil
	}

	type rawFileSpec FileSpec
	return node.Decode((*rawFileSpec)(f))
}

// UnmarshalJSON handles both string and object forms for FileSpec.
func (f *FileSpec) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		f.Path = s
		return nil
	}

	type rawFileSpec FileSpec
	return json.Unmarshal(data, (*rawFileSpec)(f))
}

// Default returns a config with default values.
func Default() *Config {
	return &Config{
		Prefix:       "",
		Files:        nil,
		GroupMarkers: nil,
		Schema:       "",
	}
}

// SchemaVersion returns the parsed schema version from the Schema field.
// Returns schema.Unknown if the field is empty or invalid.
func (c *Config) SchemaVersion() schema.Version {
	if c.Schema == "" {
		return schema.Unknown
	}
	v, err := schema.FromString(c.Schema)
	if err != nil {
		return schema.Unknown
	}
	return v
}

// OptionsForFile returns parser.Options with configuration applied.
// File-level overrides take precedence over global config.
func (c *Config) OptionsForFile(path string) parser.Options {
	opts := parser.Options{
		Prefix:        c.Prefix,
		GroupMarkers:  c.GroupMarkers,
		SchemaVersion: c.SchemaVersion(),
	}

	// Find matching file spec and apply overrides
	for _, spec := range c.Files {
		if spec.Path == path {
			if spec.Prefix != "" {
				opts.Prefix = spec.Prefix
			}
			if len(spec.GroupMarkers) > 0 {
				opts.GroupMarkers = spec.GroupMarkers
			}
			break
		}
	}

	return opts
}

// FilePaths returns the list of file paths from all FileSpecs.
func (c *Config) FilePaths() []string {
	paths := make([]string, 0, len(c.Files))
	for _, spec := range c.Files {
		paths = append(paths, spec.Path)
	}
	return paths
}
