/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package list provides the list command for asimonim.
package list

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

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
	Args:  cobra.MinimumNArgs(1),
	RunE:  run,
}

func init() {
	Cmd.Flags().String("type", "", "Filter by token type")
	Cmd.Flags().Bool("resolved", false, "Show resolved values")
	Cmd.Flags().Bool("css", false, "Output as CSS custom properties")
	Cmd.Flags().String("format", "table", "Output format: table, json, css")
}

func run(cmd *cobra.Command, args []string) error {
	typeFilter, _ := cmd.Flags().GetString("type")
	resolved, _ := cmd.Flags().GetBool("resolved")
	css, _ := cmd.Flags().GetBool("css")
	format, _ := cmd.Flags().GetString("format")
	schemaFlag, _ := cmd.Flags().GetString("schema")

	if css {
		format = "css"
	}

	filesystem := fs.NewOSFileSystem()
	jsonParser := parser.NewJSONParser()

	var schemaVersion schema.Version
	if schemaFlag != "" {
		var err error
		schemaVersion, err = schema.FromString(schemaFlag)
		if err != nil {
			return fmt.Errorf("invalid schema version: %s", schemaFlag)
		}
	}

	var allTokens []*token.Token

	for _, file := range args {
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

		opts := parser.Options{
			SchemaVersion: version,
			SkipPositions: true, // CLI doesn't need LSP position tracking
		}
		tokens, err := jsonParser.ParseFile(filesystem, file, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", file, err)
			continue
		}

		if resolved {
			if err := resolver.ResolveAliases(tokens, version); err != nil {
				fmt.Fprintf(os.Stderr, "Error resolving %s: %v\n", file, err)
			}
		}

		allTokens = append(allTokens, tokens...)
	}

	if typeFilter != "" {
		filtered := make([]*token.Token, 0)
		for _, tok := range allTokens {
			if tok.Type == typeFilter {
				filtered = append(filtered, tok)
			}
		}
		allTokens = filtered
	}

	sort.Slice(allTokens, func(i, j int) bool {
		return allTokens[i].Name < allTokens[j].Name
	})

	switch format {
	case "json":
		return outputJSON(allTokens, resolved)
	case "css":
		return outputCSS(allTokens, resolved)
	default:
		return outputTable(allTokens, resolved)
	}
}

func outputTable(tokens []*token.Token, resolved bool) error {
	for _, tok := range tokens {
		value := tok.Value
		if resolved && tok.ResolvedValue != nil {
			value = fmt.Sprintf("%v", tok.ResolvedValue)
		}
		typeStr := tok.Type
		if typeStr == "" {
			typeStr = "-"
		}
		fmt.Printf("%-40s %-12s %s\n", tok.Name, typeStr, value)
	}
	return nil
}

func outputJSON(tokens []*token.Token, resolved bool) error {
	type tokenOutput struct {
		Name        string `json:"name"`
		Value       string `json:"value"`
		Type        string `json:"type,omitempty"`
		Description string `json:"description,omitempty"`
	}

	output := make([]tokenOutput, 0, len(tokens))
	for _, tok := range tokens {
		value := tok.Value
		if resolved && tok.ResolvedValue != nil {
			value = fmt.Sprintf("%v", tok.ResolvedValue)
		}
		output = append(output, tokenOutput{
			Name:        tok.Name,
			Value:       value,
			Type:        tok.Type,
			Description: tok.Description,
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func outputCSS(tokens []*token.Token, resolved bool) error {
	fmt.Println(":root {")
	for _, tok := range tokens {
		value := tok.Value
		if resolved && tok.ResolvedValue != nil {
			value = fmt.Sprintf("%v", tok.ResolvedValue)
		}
		cssName := tok.CSSVariableName()
		if strings.HasPrefix(value, "{") && strings.Contains(value, ":") {
			continue
		}
		fmt.Printf("  %s: %s;\n", cssName, value)
	}
	fmt.Println("}")
	return nil
}
