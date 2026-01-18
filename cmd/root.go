/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package cmd provides CLI commands for asimonim.
package cmd

import (
	"github.com/spf13/cobra"

	"bennypowers.dev/asimonim/cmd/list"
	"bennypowers.dev/asimonim/cmd/search"
	"bennypowers.dev/asimonim/cmd/validate"
	"bennypowers.dev/asimonim/cmd/version"
)

var rootCmd = &cobra.Command{
	Use:   "asimonim",
	Short: "Parse and work with design tokens definitions",
	Long:  `asimonim parses and validates design token files, defined by the Design Tokens Community Group specification.`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringP("schema", "s", "", "Force schema version (draft, v2025_10)")

	rootCmd.AddCommand(list.Cmd)
	rootCmd.AddCommand(search.Cmd)
	rootCmd.AddCommand(validate.Cmd)
	rootCmd.AddCommand(version.Cmd)
}
