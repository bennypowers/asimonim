/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package cmd provides CLI commands for asimonim.
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"bennypowers.dev/asimonim/cmd/convert"
	"bennypowers.dev/asimonim/cmd/list"
	"bennypowers.dev/asimonim/cmd/search"
	"bennypowers.dev/asimonim/cmd/validate"
	"bennypowers.dev/asimonim/cmd/version"
)

// RootCmd is the root cobra command, exported for subcommand registration.
var RootCmd *cobra.Command

// Execute runs the root command.
func Execute() error {
	return RootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd = NewRootCmd()
}

// NewRootCmd creates a fresh root command with all subcommands and flags.
// Each call returns an isolated command tree with no shared state.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "asimonim",
		Short: "Parse and work with design tokens definitions",
		Long:  `asimonim parses and validates design token files, defined by the Design Tokens Community Group specification.`,
	}

	rootCmd.PersistentFlags().StringP("schema", "s", "", "Force schema version (draft, v2025.10)")
	rootCmd.PersistentFlags().StringP("prefix", "p", "", "Prefix for output variable names")

	_ = viper.BindPFlag("schema", rootCmd.PersistentFlags().Lookup("schema"))
	_ = viper.BindPFlag("prefix", rootCmd.PersistentFlags().Lookup("prefix"))

	rootCmd.AddCommand(convert.NewCmd())
	rootCmd.AddCommand(list.NewCmd())
	rootCmd.AddCommand(search.NewCmd())
	rootCmd.AddCommand(validate.NewCmd())
	rootCmd.AddCommand(version.NewCmd())

	return rootCmd
}

func initConfig() {
	// Look for config in .config directory
	viper.SetConfigName("design-tokens")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".config")
	viper.AddConfigPath(".")

	// Environment variables
	viper.SetEnvPrefix("ASIMONIM")
	viper.AutomaticEnv()

	// Read config file if it exists (ignore error if not found)
	_ = viper.ReadInConfig()
}
