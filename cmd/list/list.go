/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package list provides the list command for asimonim.
package list

import (
	"fmt"
	"maps"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"bennypowers.dev/asimonim/cmd/render"
	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/fs"
	"bennypowers.dev/asimonim/parser"
	"bennypowers.dev/asimonim/resolver"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/token"
)

// Cmd is the list cobra command.
var Cmd = &cobra.Command{
	Use:   "list [files...]",
	Short: "List tokens from design token files",
	Long:  `List all tokens from design token files with optional filtering and formatting.`,
	Args:  cobra.ArbitraryArgs,
	RunE:  run,
}

func init() {
	Cmd.Flags().String("type", "", "Filter by token type")
	Cmd.Flags().Bool("resolved", false, "Show resolved values")
	Cmd.Flags().Bool("css", false, "Output as CSS custom properties")
	Cmd.Flags().String("format", "table", "Output format: table, css, markdown")
	Cmd.Flags().String("group", "", "Filter by group/path prefix (e.g., color.brand)")
	Cmd.Flags().Bool("deprecated", false, "Show only deprecated tokens")
	Cmd.Flags().Bool("no-deprecated", false, "Hide deprecated tokens")
	Cmd.Flags().Bool("toc", false, "Include table of contents (markdown only)")
	Cmd.Flags().Int("toc-depth", 3, "Maximum TOC depth (1-6)")
	Cmd.Flags().Bool("links", false, "Add anchor links to tokens (markdown only)")
}

func run(cmd *cobra.Command, args []string) error {
	typeFilter, _ := cmd.Flags().GetString("type")
	resolved, _ := cmd.Flags().GetBool("resolved")
	css, _ := cmd.Flags().GetBool("css")
	format, _ := cmd.Flags().GetString("format")
	schemaFlag, _ := cmd.Flags().GetString("schema")
	groupFilter, _ := cmd.Flags().GetString("group")
	onlyDeprecated, _ := cmd.Flags().GetBool("deprecated")
	hideDeprecated, _ := cmd.Flags().GetBool("no-deprecated")
	includeTOC, _ := cmd.Flags().GetBool("toc")
	tocDepth, _ := cmd.Flags().GetInt("toc-depth")
	showLinks, _ := cmd.Flags().GetBool("links")

	if css {
		format = "css"
	}

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

	var allTokens []*token.Token
	var detectedVersion schema.Version
	var allGroupMeta = make(map[string]render.GroupMeta)

	// Phase 1: Parse all files
	for _, file := range files {
		data, err := filesystem.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", file, err)
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
				fmt.Fprintf(os.Stderr, "Error detecting schema for %s: %v\n", file, err)
				continue
			}
		}
		if detectedVersion == schema.Unknown {
			detectedVersion = version
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
			continue
		}

		allTokens = append(allTokens, tokens...)
	}

	// Phase 2: Resolve aliases across all tokens (enables cross-file references)
	if detectedVersion == schema.Unknown {
		detectedVersion = schema.Draft
	}
	_ = resolver.ResolveAliases(allTokens, detectedVersion)

	// Apply filters
	allTokens = filterTokens(allTokens, typeFilter, groupFilter, onlyDeprecated, hideDeprecated)

	sort.Slice(allTokens, func(i, j int) bool {
		return allTokens[i].Name < allTokens[j].Name
	})

	// Compute display rows once
	rows := render.ComputeRows(allTokens, resolved)

	switch format {
	case "css":
		return render.CSS(rows)
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
