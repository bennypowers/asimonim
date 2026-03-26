/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package mcp provides the mcp command for asimonim.
package mcp

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/fs"
	"bennypowers.dev/asimonim/internal/logger"
	mcpserver "bennypowers.dev/asimonim/mcp"
)

// NewCmd creates a fresh mcp command.
func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Launch MCP server for design tokens",
		Long: `Launch a Model Context Protocol (MCP) server that provides design token
context for AI systems.

The server discovers tokens from:
- Local token files specified in .config/design-tokens.yaml
- npm/jsr dependencies with designTokens field or export condition
- Resolver documents referenced in config

Tools provided:
- validate_tokens: Validate token files for correctness
- search_tokens: Search tokens by name, value, or type
- convert_tokens: Convert tokens to CSS, SCSS, JS, Swift, Android, etc.

Resources provided:
- asimonim://tokens: List available token sources
- asimonim://tokens/{source}: All tokens from a source
- asimonim://token/{source}/{path}: Individual token detail
- asimonim://config: Workspace configuration`,
		RunE: run,
	}
}

func run(cmd *cobra.Command, _ []string) error {
	// Silence logger to prevent stdout contamination.
	// MCP over stdio requires stdout to contain only JSON-RPC messages.
	logger.SetOutput(io.Discard)

	filesystem := fs.NewOSFileSystem()
	cfg := config.LoadOrDefault(filesystem, ".")

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	server := mcpserver.NewServer(filesystem, cfg, cwd)
	return server.Run(cmd.Context())
}
