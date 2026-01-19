/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package validate provides the validate command for asimonim.
package validate

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/fs"
	"bennypowers.dev/asimonim/parser"
	"bennypowers.dev/asimonim/resolver"
	"bennypowers.dev/asimonim/schema"
)

// Cmd is the validate cobra command.
var Cmd = &cobra.Command{
	Use:   "validate [files...]",
	Short: "Validate design token files",
	Long:  `Validate design token files for correctness and schema compliance.`,
	Args:  cobra.ArbitraryArgs,
	RunE:  run,
}

func init() {
	Cmd.Flags().Bool("strict", false, "Fail on warnings")
	Cmd.Flags().Bool("quiet", false, "Only output errors")
}

func run(cmd *cobra.Command, args []string) error {
	quiet, _ := cmd.Flags().GetBool("quiet")
	schemaFlag, _ := cmd.Flags().GetString("schema")

	filesystem := fs.NewOSFileSystem()
	jsonParser := parser.NewJSONParser()

	// Load config from .config/design-tokens.{yaml,json}
	cfg := config.LoadOrDefault(filesystem, ".")

	// Use config files if no args provided
	files := args
	if len(files) == 0 {
		expanded, err := cfg.ExpandFiles(filesystem, ".")
		if err != nil {
			return fmt.Errorf("error expanding config files: %w", err)
		}
		files = expanded
	}

	if len(files) == 0 {
		return fmt.Errorf("no files specified and no files found in config")
	}

	var schemaVersion schema.Version
	if schemaFlag != "" {
		var err error
		schemaVersion, err = schema.FromString(schemaFlag)
		if err != nil {
			return fmt.Errorf("invalid schema version: %s", schemaFlag)
		}
	} else if cfg.SchemaVersion() != schema.Unknown {
		schemaVersion = cfg.SchemaVersion()
	}

	hasErrors := false

	for _, file := range files {
		if !quiet {
			fmt.Printf("Validating %s...\n", file)
		}

		data, err := filesystem.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", file, err)
			hasErrors = true
			continue
		}

		version := schemaVersion
		if version == schema.Unknown {
			version, err = schema.DetectVersion(data, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error detecting schema for %s: %v\n", file, err)
				hasErrors = true
				continue
			}
		}

		// Get per-file options from config
		opts := cfg.OptionsForFile(file)
		opts.SkipPositions = true // CLI doesn't need LSP position tracking
		if version != schema.Unknown {
			opts.SchemaVersion = version
		}
		tokens, err := jsonParser.ParseFile(filesystem, file, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", file, err)
			hasErrors = true
			continue
		}

		graph := resolver.BuildDependencyGraph(tokens)
		if cycle := graph.FindCycle(); cycle != nil {
			fmt.Fprintf(os.Stderr, "Circular reference in %s: %v\n", file, cycle)
			hasErrors = true
			continue
		}

		if err := resolver.ResolveAliases(tokens, version); err != nil {
			fmt.Fprintf(os.Stderr, "Resolution error in %s: %v\n", file, err)
			hasErrors = true
			continue
		}

		if !quiet {
			fmt.Printf("  %d tokens, schema: %s\n", len(tokens), version)
		}
	}

	if hasErrors {
		return fmt.Errorf("validation failed")
	}

	if !quiet {
		fmt.Println("All files valid.")
	}
	return nil
}
