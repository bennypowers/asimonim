//go:build cgo

/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package lsp provides the lsp command for asimonim.
// It requires CGO for tree-sitter parser support.
package lsp

import (
	"os"

	"github.com/spf13/cobra"

	"bennypowers.dev/asimonim/cmd"
	"bennypowers.dev/asimonim/internal/version"
	"bennypowers.dev/asimonim/lsp"
)

func init() {
	cmd.RootCmd.AddCommand(&cobra.Command{
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
	})
}
