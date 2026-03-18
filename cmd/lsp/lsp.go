/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package lsp provides the lsp command for asimonim.
package lsp

import (
	"os"

	"github.com/spf13/cobra"

	"bennypowers.dev/asimonim/cmd"
	"bennypowers.dev/asimonim/internal/version"
	"bennypowers.dev/asimonim/lsp"
)

// NewCmd creates a fresh lsp command.
func NewCmd() *cobra.Command {
	lspCmd := &cobra.Command{
		Use:   "lsp",
		Short: "Start the Design Tokens Language Server",
		Long:  `Start the Design Tokens Language Server using stdio transport for communication with editors.`,
		RunE: func(c *cobra.Command, args []string) error {
			server, err := lsp.NewServer(lsp.WithVersion(version.Get()))
			if err != nil {
				return err
			}
			if err := server.RunStdio(); err != nil {
				os.Exit(1)
			}
			return nil
		},
	}

	// Accept --stdio for compatibility with vscode-languageclient,
	// which appends --stdio when transport is set to stdio.
	// The flag is accepted but ignored since stdio is the only transport.
	lspCmd.Flags().Bool("stdio", false, "Use stdio transport (default, accepted for compatibility)")

	return lspCmd
}

func init() {
	cmd.RootCmd.AddCommand(NewCmd())
}
