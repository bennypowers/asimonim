/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package list provides the list command for asimonim.
package list

import (
	"context"
	"fmt"
	"maps"
	"os"
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

// Cmd is the list cobra command.
var Cmd = NewCmd()

// NewCmd creates a fresh list command with its own flags.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [files...]",
		Short: "List tokens from design token files",
		Long:  `List all tokens from design token files with optional filtering and formatting.`,
		Args:  cobra.ArbitraryArgs,
		RunE:  run,
	}
	cmd.Flags().String("type", "", "Filter by token type")
	cmd.Flags().Bool("resolved", false, "Show resolved values")
	cmd.Flags().Bool("css", false, "Output as CSS custom properties")
	cmd.Flags().String("format", "table", "Output format: table, css, markdown")
	cmd.Flags().String("group", "", "Filter by group/path prefix (e.g., color.brand)")
	cmd.Flags().Bool("deprecated", false, "Show only deprecated tokens")
	cmd.Flags().Bool("no-deprecated", false, "Hide deprecated tokens")
	cmd.Flags().Bool("toc", false, "Include table of contents (markdown only)")
	cmd.Flags().Int("toc-depth", 3, "Maximum TOC depth (1-6)")
	cmd.Flags().Bool("links", false, "Add anchor links to tokens (markdown only)")
	return cmd
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

	if tocDepth < 1 || tocDepth > 6 {
		return fmt.Errorf("toc-depth must be between 1 and 6, got %d", tocDepth)
	}

	if onlyDeprecated && hideDeprecated {
		return fmt.Errorf("cannot use --deprecated and --no-deprecated together")
	}

	if css {
		format = "css"
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

	result, err := load.LoadAll(context.Background(), cfg, args, load.Options{
		Root:          cwd,
		SchemaVersion: schemaVersion,
	})
	if err != nil {
		return err
	}

	allTokens := result.All

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

	allTokens = filterTokens(allTokens, typeFilter, groupFilter, onlyDeprecated, hideDeprecated)

	sort.Slice(allTokens, func(i, j int) bool {
		return allTokens[i].Name < allTokens[j].Name
	})

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
