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
var RootCmd = &cobra.Command{
	Use:   "asimonim",
	Short: "Parse and work with design tokens definitions",
	Long:  `asimonim parses and validates design token files, defined by the Design Tokens Community Group specification.`,
}

// Execute runs the root command.
func Execute() error {
	return RootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringP("schema", "s", "", "Force schema version (draft, v2025.10)")
	RootCmd.PersistentFlags().StringP("prefix", "p", "", "Prefix for output variable names")

	_ = viper.BindPFlag("schema", RootCmd.PersistentFlags().Lookup("schema"))
	_ = viper.BindPFlag("prefix", RootCmd.PersistentFlags().Lookup("prefix"))

	RootCmd.AddCommand(convert.Cmd)
	RootCmd.AddCommand(list.Cmd)
	RootCmd.AddCommand(search.Cmd)
	RootCmd.AddCommand(validate.Cmd)
	RootCmd.AddCommand(version.Cmd)
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
