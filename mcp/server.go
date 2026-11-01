/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

package mcp

import (
	"context"
	"errors"
	"net/url"
	"sync"

	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/fs"
	"bennypowers.dev/asimonim/version"
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

	s.Register(s.server)

	return s
}

// Register adds asimonim's tools and resources to server.
//
// Use this to mount asimonim's token tools and resources on an existing MCP
// server. The caller owns server and its transport: Register only wires up
// handlers and does not start the server. Register panics if server is nil.
//
// Config resolution stays lazy and session-aware, but the resolved roots and
// config are cached on the Server. The first request that needs config
// resolves roots through its session, and later requests reuse that result.
// To honor different client roots, use a separate Server and registration per
// session.
func (s *Server) Register(server *mcp.Server) {
	if server == nil {
		panic("mcp: Register called with nil server")
	}
	s.setupTools(server)
	s.setupResources(server)
}

// Register creates a Server and registers its tools and resources on server.
//
// The returned Server holds asimonim's handler state (filesystem, config, and
// resolved roots). It has no transport of its own, so its Run method returns
// an error; the caller owns server and runs it with the transport of their
// choice. See [Server.Register] for details.
func Register(server *mcp.Server, filesystem fs.FileSystem, cfg *config.Config, cwd string) *Server {
	s := &Server{
		fs:  filesystem,
		cfg: cfg,
		cwd: cwd,
	}
	s.Register(server)
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
	s.cfg = config.LoadFullOrDefault(s.fs, rootDir)
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
// Handles:
//   - Unix paths: file:///home/user → /home/user
//   - Windows drive letters: file:///C:/path → C:/path
//   - UNC paths: file://server/share/path → //server/share/path
//   - Percent-encoded characters: file:///my%20dir → /my dir
func fileURIToPath(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	if u.Scheme != "file" {
		return "", &url.Error{Op: "parse", URL: uri, Err: url.InvalidHostError(u.Scheme)}
	}
	path := u.Path
	if u.Host != "" {
		// UNC-style: file://server/share → //server/share
		path = "//" + u.Host + path
	}
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
//
// Run is only available on servers created by NewServer. A Server created by
// Register has no transport of its own because the caller owns the embedding
// server and transport, so Run returns an error.
func (s *Server) Run(ctx context.Context) error {
	if s.server == nil {
		return errors.New("mcp: Run is only available on servers created by NewServer")
	}
	return s.server.Run(ctx, &mcp.StdioTransport{})
}
