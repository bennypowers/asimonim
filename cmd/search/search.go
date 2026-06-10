/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package search provides the search command for asimonim.
package search

import (
	"context"
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
	"bennypowers.dev/asimonim/load"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/token"
)

// Cmd is the search cobra command.
var Cmd = NewCmd()

// NewCmd creates a fresh search command with its own flags.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query> [files...]",
		Short: "Search tokens by name, value, or type",
		Long:  `Search design tokens by name, value, or type with optional regex support.`,
		Args:  cobra.MinimumNArgs(1),
		RunE:  run,
	}
	cmd.Flags().Bool("name", false, "Search names only")
	cmd.Flags().Bool("value", false, "Search values only")
	cmd.Flags().String("type", "", "Filter by token type")
	cmd.Flags().Bool("regex", false, "Query is a regex")
	cmd.Flags().String("format", "table", "Output format: table, names, markdown")
	cmd.Flags().String("group", "", "Filter by group/path prefix (e.g., color.brand)")
	cmd.Flags().Bool("deprecated", false, "Show only deprecated tokens")
	cmd.Flags().Bool("no-deprecated", false, "Hide deprecated tokens")
	cmd.Flags().Bool("toc", false, "Include table of contents (markdown only)")
	cmd.Flags().Int("toc-depth", 3, "Maximum TOC depth (1-6)")
	cmd.Flags().Bool("links", false, "Add anchor links to tokens (markdown only)")
	return cmd
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

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	cfg := config.LoadOrDefault(filesystem, ".")

	var schemaVersion schema.Version
	if schemaFlag != "" {
		schemaVersion, err = schema.FromString(schemaFlag)
		if err != nil {
			return fmt.Errorf("invalid schema version: %s", schemaFlag)
		}
	}

	result, err := load.LoadAll(context.Background(), cfg, files, load.Options{
		Root:          cwd,
		SchemaVersion: schemaVersion,
	})
	if err != nil {
		return err
	}

	// Extract group metadata for markdown rendering
	var allGroupMeta = make(map[string]render.GroupMeta)
	if format == "markdown" || format == "md" {
		for _, src := range result.Sources {
			data, readErr := filesystem.ReadFile(src.Path)
			if readErr != nil {
				continue
			}
			if groupMeta, metaErr := render.ExtractGroupMeta(data); metaErr == nil {
				maps.Copy(allGroupMeta, groupMeta)
			}
		}
	}

	// Search across all loaded tokens
	var matches []*token.Token
	for _, tok := range result.All {
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

	matches = filterTokens(matches, typeFilter, groupFilter, onlyDeprecated, hideDeprecated)

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Name < matches[j].Name
	})

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
