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

	"bennypowers.dev/asimonim/internal/version"
	"bennypowers.dev/asimonim/lsp"
)

// Cmd is the lsp cobra command.
var Cmd = &cobra.Command{
	Use:   "lsp",
	Short: "Start the Design Tokens Language Server",
	Long:  `Start the Design Tokens Language Server using stdio transport for communication with editors.`,
	RunE: func(cmd *cobra.Command, args []string) error {
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
