/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package mcp

import (
	"context"

	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/fs"
	"bennypowers.dev/asimonim/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server implements an MCP server for design tokens.
type Server struct {
	fs     fs.FileSystem
	cfg    *config.Config
	cwd    string
	server *mcp.Server
}

// NewServer creates a new design tokens MCP server.
func NewServer(filesystem fs.FileSystem, cfg *config.Config, cwd string) *Server {
	s := &Server{
		fs:  filesystem,
		cfg: cfg,
		cwd: cwd,
		server: mcp.NewServer(&mcp.Implementation{
			Name:    "asimonim",
			Version: version.Get(),
		}, nil),
	}

	s.setupTools()
	s.setupResources()

	return s
}

// Run starts the MCP server with stdio transport.
func (s *Server) Run(ctx context.Context) error {
	return s.server.Run(ctx, &mcp.StdioTransport{})
}
