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
	"strings"

	"bennypowers.dev/asimonim/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Selection filters a set of items by exact name.
//
// An empty Include list selects every item. Exclude is applied after Include
// and wins on overlap. Matching is exact and case-sensitive. Empty strings
// are rejected.
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

// WithConflictPolicy sets how Register resolves conflicts with items already
// registered on the target server. Repeated calls use the last value.
func WithConflictPolicy(p ConflictPolicy) RegisterOption {
	return func(c *registerConfig) {
		c.conflict = p
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

// inspectServer lists the features already registered on server.
//
// The SDK exposes no way to query a server's registered features directly.
// Conflict detection therefore uses a short-lived in-memory client, which
// observes the same surface a real client would without installing handlers.
func inspectServer(server *mcp.Server) (serverSurface, error) {
	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		return serverSurface{}, fmt.Errorf("connect: %w", err)
	}
	defer func() { _ = serverSession.Close() }()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "asimonim-register-probe",
		Version: version.Get(),
	}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		return serverSurface{}, fmt.Errorf("connect client: %w", err)
	}
	defer func() { _ = clientSession.Close() }()

	surface := serverSurface{
		tools:     make(map[string]struct{}),
		resources: make(map[string]struct{}),
		templates: make(map[string]struct{}),
	}

	for cursor := ""; ; {
		result, err := clientSession.ListTools(ctx, &mcp.ListToolsParams{Cursor: cursor})
		if err != nil {
			return serverSurface{}, fmt.Errorf("list tools: %w", err)
		}
		for _, tool := range result.Tools {
			surface.tools[tool.Name] = struct{}{}
		}
		if result.NextCursor == "" {
			break
		}
		cursor = result.NextCursor
	}

	for cursor := ""; ; {
		result, err := clientSession.ListResources(ctx, &mcp.ListResourcesParams{Cursor: cursor})
		if err != nil {
			return serverSurface{}, fmt.Errorf("list resources: %w", err)
		}
		for _, resource := range result.Resources {
			surface.resources[resource.URI] = struct{}{}
		}
		if result.NextCursor == "" {
			break
		}
		cursor = result.NextCursor
	}

	for cursor := ""; ; {
		result, err := clientSession.ListResourceTemplates(ctx, &mcp.ListResourceTemplatesParams{Cursor: cursor})
		if err != nil {
			return serverSurface{}, fmt.Errorf("list resource templates: %w", err)
		}
		for _, template := range result.ResourceTemplates {
			surface.templates[template.URITemplate] = struct{}{}
		}
		if result.NextCursor == "" {
			break
		}
		cursor = result.NextCursor
	}

	return surface, nil
}

// filterConflicts removes features that already exist on the server when the
// policy is ConflictSkip, and records every conflict for ConflictError.
func filterConflicts(kind string, features []feature, existing map[string]struct{}, policy ConflictPolicy, conflicts *[]string) []feature {
	if len(existing) == 0 {
		return features
	}
	kept := make([]feature, 0, len(features))
	for _, f := range features {
		if _, ok := existing[f.id]; ok {
			*conflicts = append(*conflicts, fmt.Sprintf("%s %q", kind, f.id))
			if policy == ConflictSkip {
				continue
			}
		}
		kept = append(kept, f)
	}
	return kept
}

// formatConflicts returns an error describing every conflicting feature.
func formatConflicts(conflicts []string) error {
	return fmt.Errorf("mcp: registration conflict: %s", strings.Join(conflicts, ", "))
}
