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

	// listRoots resolves project roots for requests without a session.
	// Injectable for tests.
	listRoots func(ctx context.Context) ([]*mcp.Root, error)

	// cfgProvided records whether cfg was passed to the constructor. An
	// explicit config is shared by every session; otherwise each session
	// resolves and caches its own config from MCP roots.
	cfgProvided bool

	mu      sync.Mutex
	rootDir string         // resolved lazily from roots, then cached
	cfg     *config.Config // explicit config, or lazily resolved without a session

	// sessions holds lazily resolved config state per MCP session.
	sessions map[*mcp.ServerSession]*sessionConfig
}

// sessionConfig holds lazily resolved config state for one MCP session.
type sessionConfig struct {
	listRoots func(ctx context.Context) ([]*mcp.Root, error)
	rootDir   string
	cfg       *config.Config
}

// NewServer creates a new design tokens MCP server.
// If cfg is nil, config is resolved lazily from MCP roots on first use.
func NewServer(filesystem fs.FileSystem, cfg *config.Config, cwd string) *Server {
	s := &Server{
		fs:          filesystem,
		cfg:         cfg,
		cfgProvided: cfg != nil,
		cwd:         cwd,
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
// Config resolution stays lazy and session-aware. Each MCP session resolves
// and caches its own roots and config, so an embedding server can serve
// clients with different workspaces. An explicit config is shared by all
// sessions. Requests without a session share a single lazy cache.
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
		fs:          filesystem,
		cfg:         cfg,
		cfgProvided: cfg != nil,
		cwd:         cwd,
	}
	s.Register(server)
	return s
}

// configForRequest resolves config for the session, caching per session.
func (s *Server) configForRequest(ctx context.Context, session *mcp.ServerSession) (*config.Config, string) {
	if session == nil {
		return s.resolveConfig(ctx)
	}
	return s.resolveSessionConfig(ctx, session)
}

// resolveConfig returns the config for a request without a session, resolving
// it lazily from MCP roots if needed. Uses double-check locking to avoid
// holding the mutex during the listRoots RPC.
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

// resolveSessionConfig returns config resolved for one MCP session. Each
// session caches its own roots and config so an embedding server serving
// multiple clients does not leak one client's workspace into another.
func (s *Server) resolveSessionConfig(ctx context.Context, session *mcp.ServerSession) (*config.Config, string) {
	s.mu.Lock()
	if s.cfgProvided {
		cfg := s.cfg
		s.mu.Unlock()
		return cfg, s.cwd
	}
	state := s.sessions[session]
	if state != nil && state.cfg != nil {
		cfg, cwd := state.cfg, configCwd(state.rootDir, s.cwd)
		s.mu.Unlock()
		return cfg, cwd
	}
	if state == nil {
		if s.sessions == nil {
			s.sessions = make(map[*mcp.ServerSession]*sessionConfig)
		}
		state = &sessionConfig{
			listRoots: func(ctx context.Context) ([]*mcp.Root, error) {
				result, err := session.ListRoots(ctx, nil)
				if err != nil || result == nil {
					return nil, err
				}
				return result.Roots, nil
			},
		}
		s.sessions[session] = state
		go s.forgetSession(session)
	}
	listRoots := state.listRoots
	s.mu.Unlock()

	rootDir := s.resolveRootDir(ctx, listRoots)

	s.mu.Lock()
	defer s.mu.Unlock()
	if state.cfg != nil {
		return state.cfg, configCwd(state.rootDir, s.cwd)
	}
	state.rootDir = rootDir
	state.cfg = config.LoadFullOrDefault(s.fs, rootDir)
	return state.cfg, configCwd(state.rootDir, s.cwd)
}

// forgetSession drops a session's cached config once its connection closes.
func (s *Server) forgetSession(session *mcp.ServerSession) {
	_ = session.Wait()
	s.mu.Lock()
	delete(s.sessions, session)
	s.mu.Unlock()
}

// configCwd returns rootDir, falling back to cwd when no root was resolved.
func configCwd(rootDir, cwd string) string {
	if rootDir != "" {
		return rootDir
	}
	return cwd
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
