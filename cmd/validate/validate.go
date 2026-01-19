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
	"bennypowers.dev/asimonim/specifier"
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
	strict, _ := cmd.Flags().GetBool("strict")
	schemaFlag, _ := cmd.Flags().GetString("schema")

	filesystem := fs.NewOSFileSystem()
	jsonParser := parser.NewJSONParser()

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	specResolver, err := specifier.NewDefaultResolver(filesystem, cwd)
	if err != nil {
		return fmt.Errorf("failed to create resolver: %w", err)
	}

	// Load config from .config/design-tokens.{yaml,json}
	cfg := config.LoadOrDefault(filesystem, ".")

	// Use config files if no args provided
	var resolvedFiles []*specifier.ResolvedFile
	if len(args) == 0 {
		var err error
		resolvedFiles, err = cfg.ResolveFiles(specResolver, filesystem, ".")
		if err != nil {
			return fmt.Errorf("error resolving config files: %w", err)
		}
	} else {
		for _, arg := range args {
			rf, err := specResolver.Resolve(arg)
			if err != nil {
				return fmt.Errorf("error resolving %s: %w", arg, err)
			}
			resolvedFiles = append(resolvedFiles, rf)
		}
	}

	if len(resolvedFiles) == 0 {
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
	hasWarnings := false

	for _, rf := range resolvedFiles {
		if !quiet {
			fmt.Printf("Validating %s...\n", rf.Specifier)
		}

		data, err := filesystem.ReadFile(rf.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", rf.Specifier, err)
			hasErrors = true
			continue
		}

		version := schemaVersion
		if version == schema.Unknown {
			version, err = schema.DetectVersion(data, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error detecting schema for %s: %v\n", rf.Specifier, err)
				hasErrors = true
				continue
			}
		}

		// Get per-file options from config (use original specifier for matching)
		opts := cfg.OptionsForFile(rf.Specifier)
		opts.SkipPositions = true // CLI doesn't need LSP position tracking
		if version != schema.Unknown {
			opts.SchemaVersion = version
		}
		tokens, err := jsonParser.ParseFile(filesystem, rf.Path, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", rf.Specifier, err)
			hasErrors = true
			continue
		}

		graph := resolver.BuildDependencyGraph(tokens)
		if cycle := graph.FindCycle(); cycle != nil {
			fmt.Fprintf(os.Stderr, "Circular reference in %s: %v\n", rf.Specifier, cycle)
			hasErrors = true
			continue
		}

		if err := resolver.ResolveAliases(tokens, version); err != nil {
			fmt.Fprintf(os.Stderr, "Resolution error in %s: %v\n", rf.Specifier, err)
			hasErrors = true
			continue
		}

		// Check for deprecated tokens (warnings)
		deprecatedCount := 0
		for _, tok := range tokens {
			if tok.Deprecated {
				deprecatedCount++
			}
		}
		if deprecatedCount > 0 {
			hasWarnings = true
			if !quiet {
				fmt.Fprintf(os.Stderr, "Warning: %s contains %d deprecated token(s)\n", rf.Specifier, deprecatedCount)
			}
		}

		if !quiet {
			fmt.Printf("  %d tokens, schema: %s\n", len(tokens), version)
		}
	}

	if hasErrors {
		return fmt.Errorf("validation failed")
	}

	if strict && hasWarnings {
		return fmt.Errorf("validation failed due to warnings (strict mode)")
	}

	if !quiet {
		fmt.Println("All files valid.")
	}
	return nil
}
