/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

//lint:file-ignore SA1019 roots/list is deprecated by SEP-2577 but still supported

package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"

	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/fs"
	"bennypowers.dev/asimonim/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerMu serializes asimonim registrations so filtering, conflict
// detection, and handler installation form one critical section. The SDK has
// no transaction spanning those calls, so this is the only way to keep two
// Register calls on the same server from interleaving. Register is a
// setup-time operation, so a single process-wide lock is fine.
var registerMu sync.Mutex

// Workspace provenance values reported in result _meta under
// [workspaceMetaKey].
const (
	// sourceExplicitConfig means the caller passed an explicit config to
	// NewServer or Register.
	sourceExplicitConfig = "explicit-config"
	// sourceHost means the caller supplied a workspace with [WithWorkspace].
	sourceHost = "host"
	// sourceRoots means the workspace came from the client's MCP roots.
	sourceRoots = "roots"
	// sourceCwd means the workspace fell back to the server's cwd.
	sourceCwd = "cwd"
)

const (
	// workspaceMetaKey is the result _meta key carrying workspace
	// provenance: {"root": string, "source": string, "warning"?: string}.
	workspaceMetaKey = "asimonim/workspace"

	// inputRequestRoots is the server-assigned ID for a roots/list input
	// request in a multi round-trip request.
	inputRequestRoots = "asimonim/roots"

	// requestStateRoots is the opaque state echoed by a client retrying a
	// roots input request. It lets the server distinguish a genuine first
	// call from a retry that lost its input responses. It is not a security
	// boundary: it only chooses between asking for roots and falling back to
	// cwd, and the client supplies the roots either way. It shares a value
	// with inputRequestRoots for readability, but the two are independent.
	requestStateRoots = "asimonim/roots"

	// protocolMRTR is the first protocol revision that forbids
	// server-initiated roots/list requests and requires multi round-trip
	// input requests instead (SEP-2322). Versions are compared
	// lexicographically, as the SDK does.
	protocolMRTR = "2026-07-28"
)

// workspace is the resolved workspace for a request together with the
// provenance surfaced to the caller.
type workspace struct {
	cfg     *config.Config
	rootDir string
	source  string
	warning string
}

// meta returns the JSON object attached to results under [workspaceMetaKey].
func (w workspace) meta() map[string]any {
	m := map[string]any{
		"root":   w.rootDir,
		"source": w.source,
	}
	if w.warning != "" {
		m["warning"] = w.warning
	}
	return m
}

// Server implements an MCP server for design tokens.
type Server struct {
	fs     fs.FileSystem
	cwd    string
	server *mcp.Server

	// listRoots resolves project roots for requests without a session.
	// Injectable for tests.
	listRoots func(ctx context.Context) ([]*mcp.Root, error) //nolint:staticcheck // roots/list is deprecated by SEP-2577 but still supported

	// cfgProvided records whether cfg was passed to the constructor. An
	// explicit config is shared by every session; otherwise each session
	// resolves and caches its own workspace.
	cfgProvided bool

	// cfg is the explicit config, nil when cfgProvided is false.
	cfg *config.Config

	// hostWorkspace is the directory supplied through [WithWorkspace]. It is
	// shared by every session and never performs a roots round trip.
	hostWorkspace string

	mu        sync.Mutex
	shared    workspaceState                         // requests without a session
	hostState workspaceState                         // lazily loaded hostWorkspace
	sessions  map[*mcp.ServerSession]*workspaceState // per-session cache
}

// workspaceState caches one resolved workspace.
type workspaceState struct {
	resolved bool
	ws       workspace
}

// NewServer creates a new design tokens MCP server.
// If cfg is nil, config is resolved lazily from MCP roots on first use. Clients
// on protocol revision 2026-07-28 resolve roots through multi round-trip
// requests (SEP-2322).
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

	// NewServer uses the same registration path as an embedder. Default
	// options cannot fail.
	if err := s.Register(s.server); err != nil {
		panic(fmt.Sprintf("mcp: NewServer: %v", err))
	}

	return s
}

// Register adds asimonim's tools and resources to server.
//
// Use this to mount asimonim's token tools and resources on an existing MCP
// server. The caller owns server and its transport: Register does not start
// the caller's transport and does not take ownership of server. Register
// returns an error for a nil server or an invalid option instead of panicking.
// ConflictError and ConflictSkip briefly attach an in-memory probe session to
// server when there are selected features to inspect; see
// [WithConflictPolicy].
//
// Options filter the mounted surface with [WithTools], [WithResources], and
// [WithTemplates], choose how name conflicts are handled with
// [WithConflictPolicy], and supply a host workspace with [WithWorkspace].
// With no options Register mounts the full surface and replaces existing
// entries, which is the behavior before options existed. Filters are exact,
// case-sensitive matches; Exclude wins over Include. All options are validated
// before any server interaction, and conflicts are detected before any handler
// is installed, so an error installs no asimonim handler. ConflictError and
// ConflictSkip do open a short-lived in-memory session to inspect the server;
// host session hooks can observe it. See [WithConflictPolicy].
//
// Register is a setup-time operation. Call it before the target server starts
// serving requests, and do not add features to the target server concurrently
// from other goroutines. Asimonim Register calls are serialized, but the SDK
// offers no transaction covering other writers, so conflict detection is best
// effort against them; a concurrent writer can still change the surface
// between inspection and installation. Do not call Register from a host
// session hook or middleware that runs during the conflict probe.
//
// Workspace resolution is lazy and session-aware, with this precedence:
//
//  1. an explicit config passed to NewServer or Register;
//  2. a workspace supplied with [WithWorkspace];
//  3. the session's cached resolution;
//  4. the client's MCP roots, directly on revisions before 2026-07-28 and
//     through multi round-trip requests on 2026-07-28 and later;
//  5. the server's cwd.
//
// Each session caches its own roots and config, so an embedding server can
// serve clients with different workspaces. Every completed tool and resource
// result carries provenance in _meta under "asimonim/workspace". When roots
// cannot be read, the result falls back to cwd with a warning in that object
// and a slog warning on stderr.
func (s *Server) Register(server *mcp.Server, opts ...RegisterOption) error {
	if server == nil {
		return errors.New("mcp: Register called with nil server")
	}

	registerMu.Lock()
	defer registerMu.Unlock()

	cfg := newRegisterConfig(opts)
	if cfg.workspaceSet {
		if s.cfgProvided {
			return errors.New("mcp: WithWorkspace cannot be combined with an explicit config")
		}
		if strings.TrimSpace(cfg.workspace) == "" {
			return errors.New("mcp: WithWorkspace requires a non-empty directory")
		}
	}
	switch cfg.conflict {
	case ConflictReplace, ConflictError, ConflictSkip:
	default:
		return fmt.Errorf("mcp: unknown conflict policy %d", cfg.conflict)
	}

	tools, err := selectFeatures("tool", s.toolFeatures(), cfg.tools)
	if err != nil {
		return err
	}
	resources, err := selectFeatures("resource", s.resourceFeatures(), cfg.resources)
	if err != nil {
		return err
	}
	templates, err := selectFeatures("template", s.templateFeatures(), cfg.templates)
	if err != nil {
		return err
	}

	if cfg.conflict != ConflictReplace {
		surface, err := inspectServer(server, surfaceNeeds{
			tools:     len(tools) > 0,
			resources: len(resources) > 0,
			templates: len(templates) > 0,
		})
		if err != nil {
			return fmt.Errorf("mcp: inspect server before registering: %w", err)
		}
		var conflict RegistrationConflict
		tools = filterConflicts("tool", tools, surface.tools, cfg.conflict, &conflict)
		resources = filterConflicts("resource", resources, surface.resources, cfg.conflict, &conflict)
		templates = filterConflicts("template", templates, surface.templates, cfg.conflict, &conflict)
		if cfg.conflict == ConflictError && !conflict.empty() {
			return &conflict
		}
	}

	// Apply the host workspace only after every option and conflict check has
	// succeeded, so a failed Register leaves the Server's resolution state
	// untouched.
	if cfg.workspaceSet {
		s.mu.Lock()
		s.hostWorkspace = cfg.workspace
		s.hostState = workspaceState{}
		s.mu.Unlock()
	}

	for _, f := range tools {
		f.add(server)
	}
	for _, f := range resources {
		f.add(server)
	}
	for _, f := range templates {
		f.add(server)
	}
	return nil
}

// Register creates a Server and registers its tools and resources on server.
// The options accepted by Register are also accepted here.
//
// The returned Server holds asimonim's handler state (filesystem, config, and
// resolved roots). It has no transport of its own, so its Run method returns
// an error; the caller owns server and runs it with the transport of their
// choice. See [Server.Register] for details.
func Register(server *mcp.Server, filesystem fs.FileSystem, cfg *config.Config, cwd string, opts ...RegisterOption) (*Server, error) {
	s := &Server{
		fs:          filesystem,
		cfg:         cfg,
		cfgProvided: cfg != nil,
		cwd:         cwd,
	}
	if err := s.Register(server, opts...); err != nil {
		return nil, err
	}
	return s, nil
}

// configForRequest resolves config and the working directory for the session,
// caching per session.
func (s *Server) configForRequest(ctx context.Context, session *mcp.ServerSession) (*config.Config, string) {
	ws := s.workspaceForRequest(ctx, session)
	return ws.cfg, ws.rootDir
}

// workspaceForRequest resolves the workspace for a request, caching per
// session. It never issues a roots input request: callers that can return an
// input-required result use prepareWorkspace first.
func (s *Server) workspaceForRequest(ctx context.Context, session *mcp.ServerSession) workspace {
	if session == nil {
		if ws, ok := s.fixedWorkspace(); ok {
			return ws
		}
		return s.sharedWorkspace(ctx)
	}
	ws, needsInput := s.prepareWorkspace(ctx, session, nil, "")
	if needsInput {
		// A handler ran without the registration wrapper. Return a transient
		// cwd workspace without caching it, so a later wrapped call can still
		// resolve roots through MRTR. This path is defensive and intentionally
		// has no end-to-end test.
		ws = s.cwdWorkspace("roots require a multi round-trip request")
		logWorkspaceWarning(session.ID(), ws)
	}
	return ws
}

// prepareWorkspace resolves the workspace before a handler runs. It reports
// needsInput when the client must first answer a roots input request on
// protocol revision 2026-07-28 or later.
func (s *Server) prepareWorkspace(
	ctx context.Context,
	session *mcp.ServerSession,
	responses mcp.InputResponseMap,
	state string,
) (workspace, bool) {
	if session == nil {
		return s.workspaceForRequest(ctx, nil), false
	}
	if ws, ok := s.fixedWorkspace(); ok {
		return ws, false
	}
	s.mu.Lock()
	if st := s.sessions[session]; st != nil && st.resolved {
		ws := st.ws
		s.mu.Unlock()
		return ws, false
	}
	s.mu.Unlock()

	protocol, hasRoots := sessionRootsCapability(session)
	if !hasRoots {
		// Never send a roots input request to a client that did not advertise
		// the capability.
		return s.cacheSessionWorkspace(session, s.cwdWorkspace("")), false
	}
	if protocol >= protocolMRTR {
		if state != requestStateRoots && responses == nil {
			return workspace{}, true
		}
		return s.cacheSessionWorkspace(session, s.workspaceFromInputResponses(responses)), false
	}
	return s.cacheSessionWorkspace(session, s.workspaceFromSessionRoots(ctx, session)), false
}

// fixedWorkspace returns an explicit config or host workspace when one is
// configured. Neither performs a roots round trip. It releases s.mu while
// loading a host workspace so other sessions are not blocked on filesystem I/O.
func (s *Server) fixedWorkspace() (workspace, bool) {
	s.mu.Lock()
	if s.cfgProvided {
		ws := workspace{cfg: s.cfg, rootDir: s.cwd, source: sourceExplicitConfig}
		s.mu.Unlock()
		return ws, true
	}
	host := s.hostWorkspace
	if s.hostState.resolved {
		ws := s.hostState.ws
		s.mu.Unlock()
		return ws, true
	}
	if host == "" {
		s.mu.Unlock()
		return workspace{}, false
	}
	s.mu.Unlock()

	ws := workspace{
		cfg:     config.LoadFullOrDefault(s.fs, host),
		rootDir: host,
		source:  sourceHost,
	}

	s.mu.Lock()
	if s.hostState.resolved {
		ws = s.hostState.ws
	} else {
		s.hostState = workspaceState{resolved: true, ws: ws}
	}
	s.mu.Unlock()
	return ws, true
}

// sharedWorkspace resolves the workspace for a request without a session,
// using the injectable listRoots hook when present.
func (s *Server) sharedWorkspace(ctx context.Context) workspace {
	s.mu.Lock()
	if s.shared.resolved {
		ws := s.shared.ws
		s.mu.Unlock()
		return ws
	}
	listRoots := s.listRoots
	s.mu.Unlock()

	resolved := s.workspaceFromDirectRoots(ctx, listRoots)

	s.mu.Lock()
	if s.shared.resolved {
		ws := s.shared.ws
		s.mu.Unlock()
		return ws
	}
	s.shared = workspaceState{resolved: true, ws: resolved}
	s.mu.Unlock()

	logWorkspaceWarning("", resolved)
	return resolved
}

// workspaceFromSessionRoots issues a direct roots/list request. It is only
// valid on protocol revisions before 2026-07-28.
func (s *Server) workspaceFromSessionRoots(ctx context.Context, session *mcp.ServerSession) workspace {
	listRoots := func(ctx context.Context) ([]*mcp.Root, error) { //nolint:staticcheck // roots/list is deprecated by SEP-2577 but still supported
		result, err := session.ListRoots(ctx, nil) //nolint:staticcheck // roots/list is required before 2026-07-28
		if err != nil || result == nil {
			return nil, err
		}
		return result.Roots, nil
	}
	return s.workspaceFromDirectRoots(ctx, listRoots)
}

// workspaceFromDirectRoots resolves the first root returned by listRoots and
// falls back to cwd with a warning when roots cannot be used.
func (s *Server) workspaceFromDirectRoots(ctx context.Context, listRoots func(context.Context) ([]*mcp.Root, error)) workspace { //nolint:staticcheck // roots/list is deprecated by SEP-2577 but still supported
	if listRoots == nil {
		return s.cwdWorkspace("")
	}
	roots, err := listRoots(ctx)
	if err != nil {
		return s.cwdWorkspace(fmt.Sprintf("roots/list failed: %v", err))
	}
	return s.workspaceFromRoots(roots)
}

// workspaceFromInputResponses resolves the workspace from a multi round-trip
// retry. A missing or wrong-typed response falls back to cwd with a warning
// instead of asking the client again. The caller caches that fallback, which is
// deliberate: a malformed retry must not turn into a retry loop.
func (s *Server) workspaceFromInputResponses(responses mcp.InputResponseMap) workspace {
	if responses == nil {
		return s.cwdWorkspace("roots retry carried no input responses")
	}
	response, ok := responses[inputRequestRoots]
	if !ok || response == nil {
		return s.cwdWorkspace(fmt.Sprintf("roots retry did not include the %q input response", inputRequestRoots))
	}
	result, ok := response.(*mcp.ListRootsResult) //nolint:staticcheck // roots/list is deprecated by SEP-2577 but still supported
	if !ok {
		return s.cwdWorkspace(fmt.Sprintf("unexpected %q input response type %T", inputRequestRoots, response))
	}
	return s.workspaceFromRoots(result.Roots)
}

// workspaceFromRoots resolves the first client root.
func (s *Server) workspaceFromRoots(roots []*mcp.Root) workspace { //nolint:staticcheck // roots/list is deprecated by SEP-2577 but still supported
	if len(roots) == 0 {
		return s.cwdWorkspace("client advertised roots but returned an empty list")
	}
	root := roots[0]
	if root == nil {
		return s.cwdWorkspace("client returned a null root")
	}
	path, err := fileURIToPath(root.URI)
	if err != nil {
		return s.cwdWorkspace(fmt.Sprintf("invalid root URI %q: %v", root.URI, err))
	}
	return workspace{
		cfg:     config.LoadFullOrDefault(s.fs, path),
		rootDir: path,
		source:  sourceRoots,
	}
}

// cwdWorkspace returns the cwd fallback, optionally with a warning.
func (s *Server) cwdWorkspace(warning string) workspace {
	return workspace{
		cfg:     config.LoadFullOrDefault(s.fs, s.cwd),
		rootDir: s.cwd,
		source:  sourceCwd,
		warning: warning,
	}
}

// sessionRootsCapability reports the negotiated protocol revision and whether
// the client advertised the roots capability.
func sessionRootsCapability(session *mcp.ServerSession) (string, bool) {
	if session == nil {
		return "", false
	}
	params := session.InitializeParams()
	if params == nil {
		return "", false
	}
	hasRoots := params.Capabilities != nil && params.Capabilities.RootsV2 != nil //nolint:staticcheck // roots/list is deprecated by SEP-2577 but still supported
	return params.ProtocolVersion, hasRoots
}

// cacheSessionWorkspace caches ws for session, returning the existing value if
// another request resolved it first. A warning is logged once per resolution.
func (s *Server) cacheSessionWorkspace(session *mcp.ServerSession, ws workspace) workspace {
	s.mu.Lock()
	state := s.ensureSession(session)
	if state.resolved {
		cached := state.ws
		s.mu.Unlock()
		return cached
	}
	state.resolved = true
	state.ws = ws
	s.mu.Unlock()

	logWorkspaceWarning(session.ID(), ws)
	return ws
}

// ensureSession returns the cache entry for session, creating it and starting
// the cleanup goroutine on first use. Callers must hold s.mu.
func (s *Server) ensureSession(session *mcp.ServerSession) *workspaceState {
	if s.sessions == nil {
		s.sessions = make(map[*mcp.ServerSession]*workspaceState)
	}
	state := s.sessions[session]
	if state == nil {
		state = new(workspaceState)
		s.sessions[session] = state
		go s.forgetSession(session)
	}
	return state
}

// forgetSession drops a session's cached workspace once its connection closes.
func (s *Server) forgetSession(session *mcp.ServerSession) {
	_ = session.Wait()
	s.mu.Lock()
	delete(s.sessions, session)
	s.mu.Unlock()
}

// logWorkspaceWarning reports a fallback once per resolution. stdout carries
// JSON-RPC only, so slog's default stderr handler is the right destination.
// The project's internal logger is deliberately not used here because MCP mode
// silences it with io.Discard, which would hide these warnings.
func logWorkspaceWarning(sessionID string, ws workspace) {
	if ws.warning == "" {
		return
	}
	args := []any{"warning", ws.warning, "root", ws.rootDir, "source", ws.source}
	if sessionID != "" {
		args = append(args, "session", sessionID)
	}
	slog.Warn("asimonim: workspace resolution", args...)
}

// metaResult is implemented by MCP results that carry _meta.
type metaResult interface {
	GetMeta() map[string]any
	SetMeta(map[string]any)
}

// attachWorkspaceMeta records workspace provenance on a result without
// disturbing other _meta entries.
func attachWorkspaceMeta(result metaResult, ws workspace) {
	meta := result.GetMeta()
	if meta == nil {
		meta = make(map[string]any, 1)
	}
	meta[workspaceMetaKey] = ws.meta()
	result.SetMeta(meta)
}

// addServerTool registers handler through the workspace-aware wrapper so no
// tool can forget multi round-trip roots or provenance.
func addServerTool[In, Out any](s *Server, server *mcp.Server, tool *mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) {
	mcp.AddTool(server, tool, wrapServerTool(s, handler))
}

func wrapServerTool[In, Out any](s *Server, handler mcp.ToolHandlerFor[In, Out]) mcp.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input In) (*mcp.CallToolResult, Out, error) {
		var zero Out
		ws, needsInput := s.prepareWorkspace(ctx, sessionFromToolReq(req), toolInputResponses(req), toolRequestState(req))
		if needsInput {
			return inputRequiredToolResult(), zero, nil
		}
		result, output, err := handler(ctx, req, input)
		if result != nil {
			attachWorkspaceMeta(result, ws)
		}
		return result, output, err
	}
}

// addResource registers handler through the workspace-aware wrapper.
func (s *Server) addResource(server *mcp.Server, resource *mcp.Resource, handler mcp.ResourceHandler) {
	server.AddResource(resource, s.wrapResource(handler))
}

// addResourceTemplate registers handler through the workspace-aware wrapper.
func (s *Server) addResourceTemplate(server *mcp.Server, template *mcp.ResourceTemplate, handler mcp.ResourceHandler) {
	server.AddResourceTemplate(template, s.wrapResource(handler))
}

func (s *Server) wrapResource(handler mcp.ResourceHandler) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		ws, needsInput := s.prepareWorkspace(ctx, sessionFromResourceReq(req), resourceInputResponses(req), resourceRequestState(req))
		if needsInput {
			return inputRequiredResourceResult(), nil
		}
		result, err := handler(ctx, req)
		if err != nil || result == nil {
			return result, err
		}
		attachWorkspaceMeta(result, ws)
		return result, nil
	}
}

func toolInputResponses(req *mcp.CallToolRequest) mcp.InputResponseMap {
	if req == nil || req.Params == nil {
		return nil
	}
	return req.Params.InputResponses
}

func toolRequestState(req *mcp.CallToolRequest) string {
	if req == nil || req.Params == nil {
		return ""
	}
	return req.Params.RequestState
}

func resourceInputResponses(req *mcp.ReadResourceRequest) mcp.InputResponseMap {
	if req == nil || req.Params == nil {
		return nil
	}
	return req.Params.InputResponses
}

func resourceRequestState(req *mcp.ReadResourceRequest) string {
	if req == nil || req.Params == nil {
		return ""
	}
	return req.Params.RequestState
}

func inputRequiredToolResult() *mcp.CallToolResult {
	return &mcp.CallToolResult{
		InputRequests: mcp.InputRequestMap{
			inputRequestRoots: &mcp.ListRootsParams{}, //nolint:staticcheck // roots/list is deprecated by SEP-2577 but still supported
		},
		RequestState: requestStateRoots,
	}
}

func inputRequiredResourceResult() *mcp.ReadResourceResult {
	return &mcp.ReadResourceResult{
		InputRequests: mcp.InputRequestMap{
			inputRequestRoots: &mcp.ListRootsParams{}, //nolint:staticcheck // roots/list is deprecated by SEP-2577 but still supported
		},
		RequestState: requestStateRoots,
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
