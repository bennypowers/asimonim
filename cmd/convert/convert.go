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
	"path/filepath"
	"regexp"
	"strconv"
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

Examples:
  # Flatten to shallow structure
  asimonim convert --flatten tokens/*.yaml

  # Convert to TypeScript module
  asimonim convert --format typescript -o tokens.ts tokens/*.yaml

  # Convert to SCSS variables
  asimonim convert --format scss -o _tokens.scss tokens/*.yaml

  # Convert to Android XML resources
  asimonim convert --format android -o values/tokens.xml tokens/*.yaml

  # Convert to iOS Swift
  asimonim convert --format swift -o DesignTokens.swift tokens/*.yaml

  # In-place schema conversion
  asimonim convert --in-place --schema v2025.10 tokens/*.yaml

  # Multi-output mode: generate multiple formats at once
  asimonim convert --outputs scss:tokens.scss --outputs typescript:tokens.ts tokens/*.yaml

  # Split by category: generate one file per top-level group
  asimonim convert --outputs "typescript:js/{group}.ts" tokens/*.yaml
  # Produces: js/color.ts, js/animation.ts, js/border.ts, etc.

  # Split by token type
  asimonim convert --outputs "scss:css/{group}.scss" --split-by type tokens/*.yaml

  # Use outputs from config file (.config/design-tokens.yaml)
  asimonim convert  # reads outputs from config`,
	Args: cobra.ArbitraryArgs,
	RunE: run,
}

func init() {
	Cmd.Flags().StringP("output", "o", "", "Output file (default: stdout)")
	Cmd.Flags().StringP("format", "f", "dtcg", "Output format: "+strings.Join(convertlib.ValidFormats(), ", "))
	Cmd.Flags().Bool("flatten", false, "Flatten to shallow structure (dtcg/json formats only)")
	Cmd.Flags().StringP("delimiter", "d", "-", "Delimiter for flattened keys")
	Cmd.Flags().BoolP("in-place", "i", false, "Overwrite input files with converted output")
	Cmd.Flags().StringArray("outputs", nil, "Multiple outputs as format:path pairs (repeatable, supports {group} template)")
	Cmd.Flags().String("split-by", "topLevel", "Split strategy: topLevel (default), type, or path[N]")
}

func run(cmd *cobra.Command, args []string) error {
	output, _ := cmd.Flags().GetString("output")
	formatFlag, _ := cmd.Flags().GetString("format")
	flatten, _ := cmd.Flags().GetBool("flatten")
	delimiter, _ := cmd.Flags().GetString("delimiter")
	inPlace, _ := cmd.Flags().GetBool("in-place")
	schemaFlag, _ := cmd.Flags().GetString("schema")
	outputsFlag, _ := cmd.Flags().GetStringArray("outputs")
	splitByFlag, _ := cmd.Flags().GetString("split-by")

	// Parse format
	format, err := convertlib.ParseFormat(formatFlag)
	if err != nil {
		return err
	}

	// Parse CLI outputs flag into OutputSpecs
	var cliOutputs []config.OutputSpec
	for _, spec := range outputsFlag {
		formatPart, pathPart, found := strings.Cut(spec, ":")
		if !found {
			return fmt.Errorf("invalid output spec %q: expected format:path", spec)
		}
		cliOutputs = append(cliOutputs, config.OutputSpec{
			Format:  formatPart,
			Path:    pathPart,
			SplitBy: splitByFlag, // Apply global split-by to all CLI outputs
		})
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
	if len(cliOutputs) > 0 && output != "" {
		return fmt.Errorf("--outputs and --output are mutually exclusive")
	}
	if len(cliOutputs) > 0 && inPlace {
		return fmt.Errorf("--outputs and --in-place are mutually exclusive")
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

	// Determine outputs: CLI flag takes precedence over config
	outputs := cliOutputs
	if len(outputs) == 0 && len(cfg.Outputs) > 0 && output == "" {
		// Use config outputs only if no single output is specified
		outputs = cfg.Outputs
	}

	// Multi-output mode
	if len(outputs) > 0 {
		return runMultiOutput(filesystem, jsonParser, cfg, resolvedFiles, targetSchema, outputs)
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
	var failures int
	for _, rf := range resolvedFiles {
		data, err := filesystem.ReadFile(rf.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", rf.Specifier, err)
			failures++
			continue
		}

		detectedVersion, err := schema.DetectVersion(data, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error detecting schema for %s: %v\n", rf.Specifier, err)
			failures++
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
			failures++
			continue
		}

		if err := resolver.ResolveAliases(tokens, detectedVersion); err != nil {
			fmt.Fprintf(os.Stderr, "Resolution error in %s: %v\n", rf.Specifier, err)
			failures++
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
			failures++
			continue
		}

		if err := filesystem.WriteFile(rf.Path, jsonBytes, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", rf.Specifier, err)
			failures++
			continue
		}
	}

	if failures > 0 {
		return fmt.Errorf("failed to convert %d file(s)", failures)
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
	// Parse all files and resolve aliases
	allTokens, detectedVersion, err := parseAndResolveTokens(filesystem, jsonParser, cfg, resolvedFiles)
	if err != nil {
		return err
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

// pathIndexPattern matches path[N] split-by values.
var pathIndexPattern = regexp.MustCompile(`^path\[(\d+)\]$`)

func runMultiOutput(
	filesystem fs.FileSystem,
	jsonParser *parser.JSONParser,
	cfg *config.Config,
	resolvedFiles []*specifier.ResolvedFile,
	targetSchema schema.Version,
	outputs []config.OutputSpec,
) error {
	// Parse all files and resolve aliases
	allTokens, detectedVersion, err := parseAndResolveTokens(filesystem, jsonParser, cfg, resolvedFiles)
	if err != nil {
		return err
	}

	// Determine output schema
	outputSchema := targetSchema
	if outputSchema == schema.Unknown {
		outputSchema = detectedVersion
	}

	// Get global prefix
	prefix := viper.GetString("prefix")
	if prefix == "" {
		prefix = cfg.Prefix
	}

	// Phase 3: Generate each output
	var failures int
	for _, out := range outputs {
		format, err := convertlib.ParseFormat(out.Format)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing format for %s: %v\n", out.Path, err)
			failures++
			continue
		}

		// Use output-specific prefix if set, otherwise global
		outPrefix := out.Prefix
		if outPrefix == "" {
			outPrefix = prefix
		}

		// Use output-specific delimiter if set
		delimiter := out.Delimiter
		if delimiter == "" {
			delimiter = "-"
		}

		// Check if this is a split output (path contains {group})
		if strings.Contains(out.Path, "{group}") {
			if err := generateSplitOutput(filesystem, allTokens, out, format, outPrefix, delimiter, detectedVersion, outputSchema); err != nil {
				fmt.Fprintf(os.Stderr, "Error generating split output %s: %v\n", out.Path, err)
				failures++
			}
			continue
		}

		// Regular single-file output
		opts := convertlib.Options{
			InputSchema:  detectedVersion,
			OutputSchema: outputSchema,
			Flatten:      out.Flatten,
			Delimiter:    delimiter,
			Format:       format,
			Prefix:       outPrefix,
		}

		outputBytes, err := convertlib.FormatTokens(allTokens, format, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting %s: %v\n", out.Path, err)
			failures++
			continue
		}

		// Append newline for proper file formatting (if not already present)
		if len(outputBytes) > 0 && outputBytes[len(outputBytes)-1] != '\n' {
			outputBytes = append(outputBytes, '\n')
		}

		// Ensure parent directory exists
		if err := ensureDir(out.Path); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory for %s: %v\n", out.Path, err)
			failures++
			continue
		}

		if err := filesystem.WriteFile(out.Path, outputBytes, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to %s: %v\n", out.Path, err)
			failures++
			continue
		}

		fmt.Fprintf(os.Stderr, "Wrote %s\n", out.Path)
	}

	if failures > 0 {
		return fmt.Errorf("failed to generate %d output(s)", failures)
	}
	return nil
}

// generateSplitOutput generates multiple files by splitting tokens based on the splitBy strategy.
func generateSplitOutput(
	filesystem fs.FileSystem,
	allTokens []*token.Token,
	out config.OutputSpec,
	format convertlib.Format,
	prefix string,
	delimiter string,
	inputSchema schema.Version,
	outputSchema schema.Version,
) error {
	// Group tokens by split key
	groups := groupTokens(allTokens, out.SplitBy)

	var failures int
	for groupName, tokens := range groups {
		// Sanitize group name to prevent path traversal
		safeName := sanitizeGroupName(groupName)

		// Expand path template with sanitized name
		path := strings.ReplaceAll(out.Path, "{group}", safeName)

		opts := convertlib.Options{
			InputSchema:  inputSchema,
			OutputSchema: outputSchema,
			Flatten:      out.Flatten,
			Delimiter:    delimiter,
			Format:       format,
			Prefix:       prefix,
		}

		outputBytes, err := convertlib.FormatTokens(tokens, format, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting %s: %v\n", path, err)
			failures++
			continue
		}

		// Append newline for proper file formatting (if not already present)
		if len(outputBytes) > 0 && outputBytes[len(outputBytes)-1] != '\n' {
			outputBytes = append(outputBytes, '\n')
		}

		// Ensure parent directory exists
		if err := ensureDir(path); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory for %s: %v\n", path, err)
			failures++
			continue
		}

		if err := filesystem.WriteFile(path, outputBytes, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to %s: %v\n", path, err)
			failures++
			continue
		}

		fmt.Fprintf(os.Stderr, "Wrote %s\n", path)
	}

	if failures > 0 {
		return fmt.Errorf("failed to generate %d split file(s)", failures)
	}
	return nil
}

// groupTokens groups tokens by the specified split strategy.
func groupTokens(tokens []*token.Token, splitBy string) map[string][]*token.Token {
	groups := make(map[string][]*token.Token)

	for _, tok := range tokens {
		key := getSplitKey(tok, splitBy)
		groups[key] = append(groups[key], tok)
	}

	return groups
}

// getSplitKey returns the split key for a token based on the split strategy.
func getSplitKey(tok *token.Token, splitBy string) string {
	switch {
	case splitBy == "" || splitBy == "topLevel":
		// Default: first path segment
		if len(tok.Path) > 0 {
			return tok.Path[0]
		}
		return "other"

	case splitBy == "type":
		// Group by token type
		if tok.Type != "" {
			return tok.Type
		}
		return "other"

	default:
		// Check for path[N] pattern
		if matches := pathIndexPattern.FindStringSubmatch(splitBy); len(matches) == 2 {
			idx, err := strconv.Atoi(matches[1])
			if err == nil && idx >= 0 && idx < len(tok.Path) {
				return tok.Path[idx]
			}
		}
		// Fallback to first path segment
		if len(tok.Path) > 0 {
			return tok.Path[0]
		}
		return "other"
	}
}

// sanitizeGroupName sanitizes a group name for use in file paths.
// It prevents path traversal attacks by replacing unsafe characters.
func sanitizeGroupName(name string) string {
	// Replace path separators and parent directory references
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "..", "_")

	// Filter to safe characters: alphanumerics, dot, dash, underscore
	var sb strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '.',
			r == '-',
			r == '_':
			sb.WriteRune(r)
		default:
			sb.WriteRune('_')
		}
	}
	return sb.String()
}

// ensureDir creates the parent directory for a file path if it doesn't exist.
func ensureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0755)
}

// parseAndResolveTokens parses all files and resolves aliases.
func parseAndResolveTokens(
	filesystem fs.FileSystem,
	jsonParser *parser.JSONParser,
	cfg *config.Config,
	resolvedFiles []*specifier.ResolvedFile,
) ([]*token.Token, schema.Version, error) {
	var allTokens []*token.Token
	var detectedVersion schema.Version

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

	if detectedVersion == schema.Unknown {
		detectedVersion = schema.Draft
	}
	if err := resolver.ResolveAliases(allTokens, detectedVersion); err != nil {
		return nil, schema.Unknown, fmt.Errorf("error resolving aliases: %w", err)
	}

	return allTokens, detectedVersion, nil
}
