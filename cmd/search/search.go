/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package search provides the search command for asimonim.
package search

import (
	"fmt"
	"maps"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"bennypowers.dev/asimonim/cmd/render"
	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/fs"
	"bennypowers.dev/asimonim/parser"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/specifier"
	"bennypowers.dev/asimonim/token"
)

// Cmd is the search cobra command.
var Cmd = &cobra.Command{
	Use:   "search <query> [files...]",
	Short: "Search tokens by name, value, or type",
	Long:  `Search design tokens by name, value, or type with optional regex support.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  run,
}

func init() {
	Cmd.Flags().Bool("name", false, "Search names only")
	Cmd.Flags().Bool("value", false, "Search values only")
	Cmd.Flags().String("type", "", "Filter by token type")
	Cmd.Flags().Bool("regex", false, "Query is a regex")
	Cmd.Flags().String("format", "table", "Output format: table, names, markdown")
	Cmd.Flags().String("group", "", "Filter by group/path prefix (e.g., color.brand)")
	Cmd.Flags().Bool("deprecated", false, "Show only deprecated tokens")
	Cmd.Flags().Bool("no-deprecated", false, "Hide deprecated tokens")
	Cmd.Flags().Bool("toc", false, "Include table of contents (markdown only)")
	Cmd.Flags().Int("toc-depth", 3, "Maximum TOC depth (1-6)")
	Cmd.Flags().Bool("links", false, "Add anchor links to tokens (markdown only)")
}

func run(cmd *cobra.Command, args []string) error {
	query := args[0]
	files := args[1:]

	nameOnly, _ := cmd.Flags().GetBool("name")
	valueOnly, _ := cmd.Flags().GetBool("value")
	typeFilter, _ := cmd.Flags().GetString("type")
	useRegex, _ := cmd.Flags().GetBool("regex")
	format, _ := cmd.Flags().GetString("format")
	schemaFlag, _ := cmd.Flags().GetString("schema")
	groupFilter, _ := cmd.Flags().GetString("group")
	onlyDeprecated, _ := cmd.Flags().GetBool("deprecated")
	hideDeprecated, _ := cmd.Flags().GetBool("no-deprecated")
	includeTOC, _ := cmd.Flags().GetBool("toc")
	tocDepth, _ := cmd.Flags().GetInt("toc-depth")
	showLinks, _ := cmd.Flags().GetBool("links")

	if onlyDeprecated && hideDeprecated {
		return fmt.Errorf("cannot use --deprecated and --no-deprecated together")
	}

	if tocDepth < 1 || tocDepth > 6 {
		return fmt.Errorf("toc-depth must be between 1 and 6, got %d", tocDepth)
	}

	var pattern *regexp.Regexp
	var err error
	if useRegex {
		pattern, err = regexp.Compile(query)
		if err != nil {
			return fmt.Errorf("invalid regex: %w", err)
		}
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

	// Use config files if no files provided
	var resolvedFiles []*specifier.ResolvedFile
	if len(files) == 0 {
		var err error
		resolvedFiles, err = cfg.ResolveFiles(specResolver, filesystem, ".")
		if err != nil {
			return fmt.Errorf("error resolving config files: %w", err)
		}
	} else {
		for _, file := range files {
			rf, err := specResolver.Resolve(file)
			if err != nil {
				return fmt.Errorf("error resolving %s: %w", file, err)
			}
			resolvedFiles = append(resolvedFiles, rf)
		}
	}

	if len(resolvedFiles) == 0 {
		return fmt.Errorf("no files specified and no files found in config")
	}

	var schemaVersion schema.Version
	if schemaFlag != "" {
		schemaVersion, err = schema.FromString(schemaFlag)
		if err != nil {
			return fmt.Errorf("invalid schema version: %s", schemaFlag)
		}
	} else if cfg.SchemaVersion() != schema.Unknown {
		schemaVersion = cfg.SchemaVersion()
	}

	var matches []*token.Token
	var allGroupMeta = make(map[string]render.GroupMeta)

	for _, rf := range resolvedFiles {
		data, err := filesystem.ReadFile(rf.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", rf.Specifier, err)
			continue
		}

		// Extract group metadata for markdown rendering
		if format == "markdown" || format == "md" {
			if groupMeta, err := render.ExtractGroupMeta(data); err == nil {
				maps.Copy(allGroupMeta, groupMeta)
			}
		}

		version := schemaVersion
		if version == schema.Unknown {
			version, err = schema.DetectVersion(data, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error detecting schema for %s: %v\n", rf.Specifier, err)
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
			continue
		}

		for _, tok := range tokens {
			matched := false
			if nameOnly {
				matched = matchString(tok.Name, query, pattern)
			} else if valueOnly {
				matched = matchString(tok.Value, query, pattern)
			} else {
				matched = matchString(tok.Name, query, pattern) ||
					matchString(tok.Value, query, pattern) ||
					matchString(tok.Type, query, pattern) ||
					matchString(tok.Description, query, pattern)
			}

			if matched {
				matches = append(matches, tok)
			}
		}
	}

	// Apply filters
	matches = filterTokens(matches, typeFilter, groupFilter, onlyDeprecated, hideDeprecated)

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Name < matches[j].Name
	})

	// Compute display rows
	rows := render.ComputeRows(matches, false)

	switch format {
	case "names":
		return render.Names(rows)
	case "markdown", "md":
		opts := render.MarkdownOptions{
			GroupMeta:  allGroupMeta,
			IncludeTOC: includeTOC,
			TOCDepth:   tocDepth,
			ShowLinks:  showLinks,
		}
		return render.MarkdownWithOptions(rows, opts)
	default:
		return render.Table(rows)
	}
}

func filterTokens(tokens []*token.Token, typeFilter, groupFilter string, onlyDeprecated, hideDeprecated bool) []*token.Token {
	result := tokens

	if typeFilter != "" {
		filtered := make([]*token.Token, 0, len(result))
		for _, tok := range result {
			if tok.Type == typeFilter {
				filtered = append(filtered, tok)
			}
		}
		result = filtered
	}

	if groupFilter != "" {
		filtered := make([]*token.Token, 0, len(result))
		for _, tok := range result {
			if strings.HasPrefix(tok.DotPath(), groupFilter) {
				filtered = append(filtered, tok)
			}
		}
		result = filtered
	}

	if onlyDeprecated {
		filtered := make([]*token.Token, 0, len(result))
		for _, tok := range result {
			if tok.Deprecated {
				filtered = append(filtered, tok)
			}
		}
		result = filtered
	} else if hideDeprecated {
		filtered := make([]*token.Token, 0, len(result))
		for _, tok := range result {
			if !tok.Deprecated {
				filtered = append(filtered, tok)
			}
		}
		result = filtered
	}

	return result
}

func matchString(s, query string, pattern *regexp.Regexp) bool {
	if pattern != nil {
		return pattern.MatchString(s)
	}
	return strings.Contains(strings.ToLower(s), strings.ToLower(query))
}
