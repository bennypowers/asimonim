/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package mcp

import (
	"testing"

	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/testutil"
)

func newTestServer(t *testing.T, fixtureDir string) *Server {
	t.Helper()
	mfs := testutil.NewFixtureFS(t, fixtureDir, "/test")
	cfg := &config.Config{
		Files: []config.FileSpec{{Path: "/test/tokens.json"}},
	}
	return NewServer(mfs, cfg, "/test")
}

func TestNewServer(t *testing.T) {
	s := newTestServer(t, "fixtures/draft/simple")
	if s == nil {
		t.Fatal("expected non-nil server")
	}
	if s.server == nil {
		t.Fatal("expected non-nil MCP server")
	}
}
