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
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"bennypowers.dev/asimonim/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	// probeClientName identifies the short-lived client used to inspect a
	// target server during conflict detection.
	probeClientName = "asimonim-register-probe"
)

// probeTimeout bounds the probe's RPCs so a blocked host handler or middleware
// cannot stall Register indefinitely. It is a variable so tests can shorten it.
var probeTimeout = 5 * time.Second

// Selection filters a set of items by exact name.
//
// An empty Include list selects every item. Exclude is applied after Include
// and wins on overlap. Matching is exact and case-sensitive. Empty strings
// are rejected. An Exclude entry that matches nothing is accepted and
// ignored, so a deny list stays valid across versions.
type Selection struct {
	Include []string
	Exclude []string
}

// RegisterOption configures [Server.Register].
type RegisterOption func(*registerConfig)

// registerConfig is the accumulated state of the options passed to Register.
type registerConfig struct {
	tools     Selection
	resources Selection
	templates Selection
	conflict  ConflictPolicy
}

// WithTools filters the tools Register mounts by exact tool name.
//
// Repeated calls accumulate: their Include and Exclude lists are unioned.
// Exclude wins over Include. An Include entry that names no tool is an error.
func WithTools(sel Selection) RegisterOption {
	sel = cloneSelection(sel)
	return func(c *registerConfig) {
		c.tools = mergeSelections(c.tools, sel)
	}
}

// WithResources filters the resources Register mounts by exact resource URI.
//
// Repeated calls accumulate: their Include and Exclude lists are unioned.
// Exclude wins over Include. An Include entry that names no resource is an
// error.
func WithResources(sel Selection) RegisterOption {
	sel = cloneSelection(sel)
	return func(c *registerConfig) {
		c.resources = mergeSelections(c.resources, sel)
	}
}

// WithTemplates filters the resource templates Register mounts by exact URI
// template string.
//
// Repeated calls accumulate: their Include and Exclude lists are unioned.
// Exclude wins over Include. An Include entry that names no template is an
// error.
func WithTemplates(sel Selection) RegisterOption {
	sel = cloneSelection(sel)
	return func(c *registerConfig) {
		c.templates = mergeSelections(c.templates, sel)
	}
}

// ConflictPolicy controls what Register does when an item it would mount
// already exists on the target server.
type ConflictPolicy int

const (
	// ConflictReplace overwrites the existing entry. This is the default and
	// the behavior before options existed.
	ConflictReplace ConflictPolicy = iota
	// ConflictError fails Register before anything is registered. The target
	// server is left unchanged.
	ConflictError
	// ConflictSkip keeps the existing entry and skips the conflicting one.
	ConflictSkip
)

// RegistrationConflict reports the features that already exist on the target
// server when Register is called with [ConflictError]. It is returned as an
// error and can be inspected with [errors.As].
type RegistrationConflict struct {
	// Tools lists conflicting tool names.
	Tools []string
	// Resources lists conflicting resource URIs.
	Resources []string
	// Templates lists conflicting URI template strings.
	Templates []string
}

// Error implements the error interface.
func (c *RegistrationConflict) Error() string {
	parts := make([]string, 0, len(c.Tools)+len(c.Resources)+len(c.Templates))
	for _, id := range c.Tools {
		parts = append(parts, fmt.Sprintf("tool %q", id))
	}
	for _, id := range c.Resources {
		parts = append(parts, fmt.Sprintf("resource %q", id))
	}
	for _, id := range c.Templates {
		parts = append(parts, fmt.Sprintf("template %q", id))
	}
	return "mcp: registration conflict: " + strings.Join(parts, ", ")
}

func (c *RegistrationConflict) empty() bool {
	return len(c.Tools) == 0 && len(c.Resources) == 0 && len(c.Templates) == 0
}

// add records one conflicting feature.
func (c *RegistrationConflict) add(kind, id string) {
	switch kind {
	case "tool":
		c.Tools = append(c.Tools, id)
	case "resource":
		c.Resources = append(c.Resources, id)
	case "template":
		c.Templates = append(c.Templates, id)
	}
}

// WithConflictPolicy sets how Register resolves conflicts with items already
// registered on the target server. Repeated calls use the last value.
//
// ConflictError and ConflictSkip inspect the target server through a
// short-lived in-memory client session, so host session hooks, middleware,
// and connection logging observe that probe session. ConflictReplace, the
// default, does not inspect the server.
func WithConflictPolicy(p ConflictPolicy) RegisterOption {
	return func(c *registerConfig) {
		c.conflict = p
	}
}

// cloneSelection copies both lists so later mutation of the caller's Selection
// cannot change an option that was already constructed.
func cloneSelection(sel Selection) Selection {
	return Selection{
		Include: slices.Clone(sel.Include),
		Exclude: slices.Clone(sel.Exclude),
	}
}

// mergeSelections returns the union of a and b. The result owns its slices, so
// later changes to either input do not affect the accumulated options.
func mergeSelections(a, b Selection) Selection {
	if len(b.Include) > 0 {
		a.Include = append(a.Include[:len(a.Include):len(a.Include)], b.Include...)
	}
	if len(b.Exclude) > 0 {
		a.Exclude = append(a.Exclude[:len(a.Exclude):len(a.Exclude)], b.Exclude...)
	}
	return a
}

func newRegisterConfig(opts []RegisterOption) *registerConfig {
	cfg := new(registerConfig)
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}
	return cfg
}

// feature is one static item asimonim can mount on an MCP server.
type feature struct {
	id  string
	add func(*mcp.Server)
}

// selectFeatures validates sel against all and returns the features it
// selects. Validation covers every entry before filtering so an invalid
// selection never partially registers.
func selectFeatures(kind string, all []feature, sel Selection) ([]feature, error) {
	known := make(map[string]struct{}, len(all))
	for _, f := range all {
		known[f.id] = struct{}{}
	}
	for _, id := range sel.Include {
		if id == "" {
			return nil, fmt.Errorf("mcp: %s selection: empty entry in Include", kind)
		}
		if _, ok := known[id]; !ok {
			return nil, fmt.Errorf("mcp: %s selection: unknown Include entry %q", kind, id)
		}
	}
	for _, id := range sel.Exclude {
		if id == "" {
			return nil, fmt.Errorf("mcp: %s selection: empty entry in Exclude", kind)
		}
	}

	include := make(map[string]struct{}, len(sel.Include))
	for _, id := range sel.Include {
		include[id] = struct{}{}
	}
	exclude := make(map[string]struct{}, len(sel.Exclude))
	for _, id := range sel.Exclude {
		exclude[id] = struct{}{}
	}

	selected := make([]feature, 0, len(all))
	for _, f := range all {
		if len(include) > 0 {
			if _, ok := include[f.id]; !ok {
				continue
			}
		}
		if _, ok := exclude[f.id]; ok {
			continue
		}
		selected = append(selected, f)
	}
	return selected, nil
}

// serverSurface is the set of feature identifiers already registered on a
// target server.
type serverSurface struct {
	tools     map[string]struct{}
	resources map[string]struct{}
	templates map[string]struct{}
}

// surfaceNeeds selects which feature categories conflict detection inspects.
type surfaceNeeds struct {
	tools     bool
	resources bool
	templates bool
}

func (n surfaceNeeds) empty() bool {
	return !n.tools && !n.resources && !n.templates
}

// inspectServer lists the features already registered on server.
//
// The SDK exposes no way to query a server's registered features directly.
// Conflict detection therefore uses a short-lived in-memory client, which
// observes the same surface a real client would without installing handlers.
// The probe performs a normal MCP initialize handshake, so host session hooks
// and middleware run for it. Only the requested categories are listed.
func inspectServer(server *mcp.Server, need surfaceNeeds) (serverSurface, error) {
	surface := serverSurface{
		tools:     make(map[string]struct{}),
		resources: make(map[string]struct{}),
		templates: make(map[string]struct{}),
	}
	if need.empty() {
		return surface, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		return serverSurface{}, fmt.Errorf("connect: %w", err)
	}
	defer closeProbeSessions(serverSession)

	client := mcp.NewClient(&mcp.Implementation{
		Name:    probeClientName,
		Version: version.Get(),
	}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		return serverSurface{}, fmt.Errorf("connect client: %w", err)
	}
	defer closeProbeSessions(clientSession)

	if need.tools {
		tools, err := listFeatureIDs(func(cursor string) (string, []string, error) {
			result, err := clientSession.ListTools(ctx, &mcp.ListToolsParams{Cursor: cursor})
			if err != nil {
				return "", nil, fmt.Errorf("list tools: %w", err)
			}
			ids := make([]string, len(result.Tools))
			for i, tool := range result.Tools {
				ids[i] = tool.Name
			}
			return result.NextCursor, ids, nil
		})
		if err != nil {
			return serverSurface{}, err
		}
		surface.tools = tools
	}

	if need.resources {
		resources, err := listFeatureIDs(func(cursor string) (string, []string, error) {
			result, err := clientSession.ListResources(ctx, &mcp.ListResourcesParams{Cursor: cursor})
			if err != nil {
				return "", nil, fmt.Errorf("list resources: %w", err)
			}
			ids := make([]string, len(result.Resources))
			for i, resource := range result.Resources {
				ids[i] = resource.URI
			}
			return result.NextCursor, ids, nil
		})
		if err != nil {
			return serverSurface{}, err
		}
		surface.resources = resources
	}

	if need.templates {
		templates, err := listFeatureIDs(func(cursor string) (string, []string, error) {
			result, err := clientSession.ListResourceTemplates(ctx, &mcp.ListResourceTemplatesParams{Cursor: cursor})
			if err != nil {
				return "", nil, fmt.Errorf("list resource templates: %w", err)
			}
			ids := make([]string, len(result.ResourceTemplates))
			for i, template := range result.ResourceTemplates {
				ids[i] = template.URITemplate
			}
			return result.NextCursor, ids, nil
		})
		if err != nil {
			return serverSurface{}, err
		}
		surface.templates = templates
	}

	return surface, nil
}

// closeProbeSessions closes probe sessions in the background. Close waits for
// in-flight handler goroutines, so a host handler or middleware that ignores
// context cancellation could otherwise block Register after the probe deadline.
func closeProbeSessions(sessions ...io.Closer) {
	for _, session := range sessions {
		go func() { _ = session.Close() }()
	}
}

// listFeatureIDs collects every identifier returned by fetch, following
// pagination cursors until the server reports no next page.
func listFeatureIDs(fetch func(cursor string) (next string, ids []string, err error)) (map[string]struct{}, error) {
	all := make(map[string]struct{})
	for cursor := ""; ; {
		next, ids, err := fetch(cursor)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			all[id] = struct{}{}
		}
		if next == "" {
			return all, nil
		}
		cursor = next
	}
}

// filterConflicts removes features that already exist on the server when the
// policy is ConflictSkip, and records every conflict for ConflictError.
func filterConflicts(kind string, features []feature, existing map[string]struct{}, policy ConflictPolicy, conflict *RegistrationConflict) []feature {
	if len(existing) == 0 {
		return features
	}
	kept := make([]feature, 0, len(features))
	for _, f := range features {
		if _, ok := existing[f.id]; ok {
			conflict.add(kind, f.id)
			if policy == ConflictSkip {
				continue
			}
		}
		kept = append(kept, f)
	}
	return kept
}
