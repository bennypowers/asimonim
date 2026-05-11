/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package mcp

import (
	"context"
	"net/url"
	"sync"

	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/fs"
	"bennypowers.dev/asimonim/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server implements an MCP server for design tokens.
type Server struct {
	fs     fs.FileSystem
	cwd    string
	server *mcp.Server

	// listRoots resolves project roots from the MCP client.
	// Set automatically from session on first tool call; injectable for tests.
	listRoots func(ctx context.Context) ([]*mcp.Root, error)

	mu      sync.Mutex
	rootDir string         // resolved lazily from roots, then cached
	cfg     *config.Config // loaded lazily from rootDir, then cached
}

// NewServer creates a new design tokens MCP server.
// If cfg is nil, config is resolved lazily from MCP roots on first use.
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

// configForRequest resolves config from the MCP session, caching the result.
// Captures the session's ListRoots on first call, then resolves config lazily.
func (s *Server) configForRequest(ctx context.Context, session *mcp.ServerSession) (*config.Config, string) {
	if session != nil {
		s.ensureListRoots(session)
	}
	return s.resolveConfig(ctx)
}

// resolveConfig returns the config, resolving it lazily from MCP roots if needed.
// Uses double-check locking to avoid holding the mutex during the listRoots RPC.
func (s *Server) resolveConfig(ctx context.Context) (*config.Config, string) {
	s.mu.Lock()
	if s.cfg != nil {
		cfg, cwd := s.cfg, s.resolvedCwd()
		s.mu.Unlock()
		return cfg, cwd
	}
	listRoots := s.listRoots
	s.mu.Unlock()

	rootDir := s.resolveRootDir(ctx, listRoots)

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cfg != nil {
		return s.cfg, s.resolvedCwd()
	}
	s.rootDir = rootDir
	s.cfg = config.LoadOrDefault(s.fs, rootDir)
	return s.cfg, s.resolvedCwd()
}

// resolveRootDir determines the project root directory.
// Tries MCP roots first (primary root), falls back to cwd.
func (s *Server) resolveRootDir(ctx context.Context, listRoots func(context.Context) ([]*mcp.Root, error)) string {
	if listRoots == nil {
		return s.cwd
	}

	roots, err := listRoots(ctx)
	if err != nil || len(roots) == 0 {
		return s.cwd
	}

	path, err := fileURIToPath(roots[0].URI)
	if err != nil {
		return s.cwd
	}

	return path
}

// resolvedCwd returns the resolved project root, or cwd as fallback.
// Must be called under s.mu.
func (s *Server) resolvedCwd() string {
	if s.rootDir != "" {
		return s.rootDir
	}
	return s.cwd
}

// ensureListRoots captures the session's ListRoots for lazy config resolution.
// No-op if listRoots is already set (e.g., by tests).
func (s *Server) ensureListRoots(session *mcp.ServerSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listRoots != nil || session == nil {
		return
	}
	s.listRoots = func(ctx context.Context) ([]*mcp.Root, error) {
		result, err := session.ListRoots(ctx, nil)
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, nil
		}
		return result.Roots, nil
	}
}

// fileURIToPath converts a file:// URI to a filesystem path.
// Handles Windows drive letters (file:///C:/path → C:/path)
// and percent-encoded characters (file:///my%20dir → /my dir).
func fileURIToPath(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	if u.Scheme != "file" {
		return "", &url.Error{Op: "parse", URL: uri, Err: url.InvalidHostError(u.Scheme)}
	}
	path := u.Path
	if path == "" {
		return "", &url.Error{Op: "parse", URL: uri, Err: url.InvalidHostError("empty path")}
	}
	// Handle Windows drive letters: /C:/path → C:/path
	if len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}
	return path, nil
}

// Run starts the MCP server with stdio transport.
func (s *Server) Run(ctx context.Context) error {
	return s.server.Run(ctx, &mcp.StdioTransport{})
}
