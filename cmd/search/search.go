/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package search provides the search command for asimonim.
package search

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/fs"
	"bennypowers.dev/asimonim/parser"
	"bennypowers.dev/asimonim/schema"
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
	Cmd.Flags().String("format", "table", "Output format: table, json, names")
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

	// Load config from .config/design-tokens.{yaml,json}
	cfg := config.LoadOrDefault(filesystem, ".")

	// Use config files if no files provided
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
		schemaVersion, err = schema.FromString(schemaFlag)
		if err != nil {
			return fmt.Errorf("invalid schema version: %s", schemaFlag)
		}
	} else if cfg.SchemaVersion() != schema.Unknown {
		schemaVersion = cfg.SchemaVersion()
	}

	var matches []*token.Token

	for _, file := range files {
		data, err := filesystem.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", file, err)
			continue
		}

		version := schemaVersion
		if version == schema.Unknown {
			version, err = schema.DetectVersion(data, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error detecting schema for %s: %v\n", file, err)
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
			continue
		}

		for _, tok := range tokens {
			if typeFilter != "" && tok.Type != typeFilter {
				continue
			}

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

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Name < matches[j].Name
	})

	switch format {
	case "json":
		return outputJSON(matches)
	case "names":
		return outputNames(matches)
	default:
		return outputTable(matches)
	}
}

func matchString(s, query string, pattern *regexp.Regexp) bool {
	if pattern != nil {
		return pattern.MatchString(s)
	}
	return strings.Contains(strings.ToLower(s), strings.ToLower(query))
}

func outputTable(tokens []*token.Token) error {
	if len(tokens) == 0 {
		return nil
	}

	// Calculate column widths
	nameWidth := 4
	typeWidth := 4
	for _, tok := range tokens {
		name := tok.CSSVariableName()
		if len(name) > nameWidth {
			nameWidth = len(name)
		}
		if len(tok.Type) > typeWidth {
			typeWidth = len(tok.Type)
		}
	}

	for _, tok := range tokens {
		typeStr := tok.Type
		if typeStr == "" {
			typeStr = "-"
		}
		fmt.Printf("%-*s  %-*s  %s\n", nameWidth, tok.CSSVariableName(), typeWidth, typeStr, tok.Value)
	}
	return nil
}

func outputJSON(tokens []*token.Token) error {
	type tokenOutput struct {
		Name        string `json:"name"`
		Value       string `json:"value"`
		Type        string `json:"type,omitempty"`
		Description string `json:"description,omitempty"`
		FilePath    string `json:"file,omitempty"`
	}

	output := make([]tokenOutput, 0, len(tokens))
	for _, tok := range tokens {
		output = append(output, tokenOutput{
			Name:        tok.CSSVariableName(),
			Value:       tok.Value,
			Type:        tok.Type,
			Description: tok.Description,
			FilePath:    tok.FilePath,
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func outputNames(tokens []*token.Token) error {
	for _, tok := range tokens {
		fmt.Println(tok.CSSVariableName())
	}
	return nil
}
