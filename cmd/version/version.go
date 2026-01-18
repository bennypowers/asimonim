/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package version provides the version command for asimonim.
package version

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"bennypowers.dev/asimonim/internal/version"
)

// Cmd is the version cobra command that prints version and build information.
var Cmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print version information for asimonim.`,
	RunE:  run,
}

func init() {
	Cmd.Flags().StringP("format", "f", "text", "Output format (text, json)")
}

func run(cmd *cobra.Command, args []string) error {
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return fmt.Errorf("error reading format flag: %w", err)
	}
	switch format {
	case "json":
		buildInfo := version.Info()
		out, err := json.MarshalIndent(buildInfo, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling version info: %w", err)
		}
		fmt.Println(string(out))
	default:
		fmt.Printf("asimonim %s\n", version.Get())
	}
	return nil
}
