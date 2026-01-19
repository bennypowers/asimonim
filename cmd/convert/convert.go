/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package convert provides the convert command for asimonim.
package convert

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"bennypowers.dev/asimonim/config"
	convertlib "bennypowers.dev/asimonim/convert"
	"bennypowers.dev/asimonim/fs"
	"bennypowers.dev/asimonim/parser"
	"bennypowers.dev/asimonim/resolver"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/specifier"
	"bennypowers.dev/asimonim/token"
)

// Cmd is the convert cobra command.
var Cmd = &cobra.Command{
	Use:   "convert [files...]",
	Short: "Convert and combine token files",
	Long: `Convert DTCG token files between formats, combine multiple files, and flatten structure.

Output Formats:
  dtcg       DTCG-compliant JSON (default)
  json       Flat key-value JSON
  android    Android-style XML resources
  swift      iOS Swift constants with native SwiftUI Color
  typescript TypeScript ESM module with 'as const' exports
  cts        TypeScript CommonJS module with 'as const' exports
  scss       SCSS variables with kebab-case names
  tailwind   Tailwind theme configuration

Examples:
  # Flatten to shallow structure
  asimonim convert --flatten tokens/*.yaml

  # Convert to TypeScript module
  asimonim convert --format typescript -o tokens.ts tokens/*.yaml

  # Convert to SCSS variables
  asimonim convert --format scss -o _tokens.scss tokens/*.yaml

  # Convert to Tailwind config
  asimonim convert --format tailwind -o tailwind.tokens.js tokens/*.yaml

  # Convert to Android XML resources
  asimonim convert --format android -o values/tokens.xml tokens/*.yaml

  # Convert to iOS Swift
  asimonim convert --format swift -o DesignTokens.swift tokens/*.yaml

  # In-place schema conversion
  asimonim convert --in-place --schema v2025.10 tokens/*.yaml`,
	Args: cobra.ArbitraryArgs,
	RunE: run,
}

func init() {
	Cmd.Flags().StringP("output", "o", "", "Output file (default: stdout)")
	Cmd.Flags().StringP("format", "f", "dtcg", "Output format: "+strings.Join(convertlib.ValidFormats(), ", "))
	Cmd.Flags().Bool("flatten", false, "Flatten to shallow structure (dtcg/json formats only)")
	Cmd.Flags().StringP("delimiter", "d", "-", "Delimiter for flattened keys")
	Cmd.Flags().BoolP("in-place", "i", false, "Overwrite input files with converted output")
}

func run(cmd *cobra.Command, args []string) error {
	output, _ := cmd.Flags().GetString("output")
	formatFlag, _ := cmd.Flags().GetString("format")
	flatten, _ := cmd.Flags().GetBool("flatten")
	delimiter, _ := cmd.Flags().GetString("delimiter")
	inPlace, _ := cmd.Flags().GetBool("in-place")
	schemaFlag, _ := cmd.Flags().GetString("schema")

	// Parse format
	format, err := convertlib.ParseFormat(formatFlag)
	if err != nil {
		return err
	}

	// Validate flag combinations
	if inPlace && output != "" {
		return fmt.Errorf("--in-place and --output are mutually exclusive")
	}
	if inPlace && flatten {
		return fmt.Errorf("--in-place and --flatten are mutually exclusive")
	}
	if inPlace && format != convertlib.FormatDTCG {
		return fmt.Errorf("--in-place only supports dtcg format")
	}

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

	var targetSchema schema.Version
	if schemaFlag != "" {
		var err error
		targetSchema, err = schema.FromString(schemaFlag)
		if err != nil {
			return fmt.Errorf("invalid schema version: %s", schemaFlag)
		}
	} else if cfg.SchemaVersion() != schema.Unknown {
		targetSchema = cfg.SchemaVersion()
	}

	if inPlace {
		return runInPlace(filesystem, jsonParser, cfg, resolvedFiles, targetSchema)
	}

	return runCombined(filesystem, jsonParser, cfg, resolvedFiles, targetSchema, output, format, flatten, delimiter)
}

func runInPlace(
	filesystem fs.FileSystem,
	jsonParser *parser.JSONParser,
	cfg *config.Config,
	resolvedFiles []*specifier.ResolvedFile,
	targetSchema schema.Version,
) error {
	for _, rf := range resolvedFiles {
		data, err := filesystem.ReadFile(rf.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", rf.Specifier, err)
			continue
		}

		detectedVersion, err := schema.DetectVersion(data, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error detecting schema for %s: %v\n", rf.Specifier, err)
			continue
		}

		outputSchema := targetSchema
		if outputSchema == schema.Unknown {
			outputSchema = detectedVersion
		}

		opts := cfg.OptionsForFile(rf.Specifier)
		opts.SkipPositions = true
		if detectedVersion != schema.Unknown {
			opts.SchemaVersion = detectedVersion
		}

		tokens, err := jsonParser.ParseFile(filesystem, rf.Path, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", rf.Specifier, err)
			continue
		}

		if err := resolver.ResolveAliases(tokens, detectedVersion); err != nil {
			fmt.Fprintf(os.Stderr, "Resolution error in %s: %v\n", rf.Specifier, err)
			continue
		}

		result := convertlib.Serialize(tokens, convertlib.Options{
			InputSchema:  detectedVersion,
			OutputSchema: outputSchema,
			Flatten:      false,
			Delimiter:    "-",
		})
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error serializing %s: %v\n", rf.Specifier, err)
			continue
		}

		if err := filesystem.WriteFile(rf.Path, jsonBytes, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", rf.Specifier, err)
			continue
		}
	}

	return nil
}

func runCombined(
	filesystem fs.FileSystem,
	jsonParser *parser.JSONParser,
	cfg *config.Config,
	resolvedFiles []*specifier.ResolvedFile,
	targetSchema schema.Version,
	output string,
	format convertlib.Format,
	flatten bool,
	delimiter string,
) error {
	var allTokens []*token.Token
	var detectedVersion schema.Version

	// Phase 1: Parse all files
	for _, rf := range resolvedFiles {
		data, err := filesystem.ReadFile(rf.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", rf.Specifier, err)
			continue
		}

		version, err := schema.DetectVersion(data, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error detecting schema for %s: %v\n", rf.Specifier, err)
			continue
		}
		if detectedVersion == schema.Unknown {
			detectedVersion = version
		}

		opts := cfg.OptionsForFile(rf.Specifier)
		opts.SkipPositions = true
		if version != schema.Unknown {
			opts.SchemaVersion = version
		}

		tokens, err := jsonParser.ParseFile(filesystem, rf.Path, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", rf.Specifier, err)
			continue
		}

		allTokens = append(allTokens, tokens...)
	}

	// Phase 2: Resolve aliases across all tokens
	if detectedVersion == schema.Unknown {
		detectedVersion = schema.Draft
	}
	if err := resolver.ResolveAliases(allTokens, detectedVersion); err != nil {
		return fmt.Errorf("error resolving aliases: %w", err)
	}

	// Determine output schema
	outputSchema := targetSchema
	if outputSchema == schema.Unknown {
		outputSchema = detectedVersion
	}

	// Get prefix from viper (CLI flag or config file)
	prefix := viper.GetString("prefix")
	if prefix == "" {
		prefix = cfg.Prefix
	}

	// Phase 3: Serialize tokens to requested format
	opts := convertlib.Options{
		InputSchema:  detectedVersion,
		OutputSchema: outputSchema,
		Flatten:      flatten,
		Delimiter:    delimiter,
		Format:       format,
		Prefix:       prefix,
	}

	outputBytes, err := convertlib.FormatTokens(allTokens, format, opts)
	if err != nil {
		return fmt.Errorf("error formatting output: %w", err)
	}

	// Append newline for proper file formatting (if not already present)
	if len(outputBytes) > 0 && outputBytes[len(outputBytes)-1] != '\n' {
		outputBytes = append(outputBytes, '\n')
	}

	// Phase 4: Write output
	if output != "" {
		if err := filesystem.WriteFile(output, outputBytes, 0644); err != nil {
			return fmt.Errorf("error writing to %s: %w", output, err)
		}
		return nil
	}

	// Write to stdout
	fmt.Print(string(outputBytes))
	return nil
}
