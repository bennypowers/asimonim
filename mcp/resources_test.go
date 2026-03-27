/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/testutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleTokenSources(t *testing.T) {
	s := newTestServer(t, "fixtures/draft/simple")

	result, err := s.handleTokenSources(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "asimonim://tokens"},
	})
	require.NoError(t, err)
	require.Len(t, result.Contents, 1)

	var sources []sourceSummary
	require.NoError(t, json.Unmarshal([]byte(result.Contents[0].Text), &sources))
	assert.Len(t, sources, 1)
	// /test/tokens.json source with 5 tokens
	assert.Equal(t, "/test/tokens.json", sources[0].Source)
	assert.Equal(t, 5, sources[0].TokenCount)
}

func TestHandleConfig(t *testing.T) {
	s := newTestServer(t, "fixtures/draft/simple")

	result, err := s.handleConfig(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "asimonim://config"},
	})
	require.NoError(t, err)
	require.Len(t, result.Contents, 1)

	var cfg map[string]any
	require.NoError(t, json.Unmarshal([]byte(result.Contents[0].Text), &cfg))
	files, ok := cfg["files"].([]any)
	require.True(t, ok)
	assert.Len(t, files, 1)
}

func TestHandleTokensBySource(t *testing.T) {
	s := newTestServer(t, "fixtures/draft/simple")

	t.Run("existing source", func(t *testing.T) {
		result, err := s.handleTokensBySource(context.Background(), &mcp.ReadResourceRequest{
			Params: &mcp.ReadResourceParams{URI: "asimonim://tokens//test/tokens.json"},
		})
		require.NoError(t, err)
		require.Len(t, result.Contents, 1)

		var tokens []tokenSummary
		require.NoError(t, json.Unmarshal([]byte(result.Contents[0].Text), &tokens))
		assert.Len(t, tokens, 5)
	})

	t.Run("nonexistent source", func(t *testing.T) {
		_, err := s.handleTokensBySource(context.Background(), &mcp.ReadResourceRequest{
			Params: &mcp.ReadResourceParams{URI: "asimonim://tokens/nonexistent"},
		})
		assert.Error(t, err)
	})
}

func TestHandleTokenSources_InvalidConfig(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
	cfg := &config.Config{
		Files: []config.FileSpec{{Path: "/test/nonexistent.json"}},
	}
	s := NewServer(mfs, cfg, "/test")

	_, err := s.handleTokenSources(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "asimonim://tokens"},
	})
	assert.Error(t, err)
}

func TestHandleConfig_InvalidConfig(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
	cfg := &config.Config{
		Files: []config.FileSpec{{Path: "/test/nonexistent.json"}},
	}
	s := NewServer(mfs, cfg, "/test")

	// Config resource should still work even with invalid files
	// because it just reports the config, not parsed tokens
	result, err := s.handleConfig(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "asimonim://config"},
	})
	require.NoError(t, err)
	require.Len(t, result.Contents, 1)
}

func TestHandleTokensBySource_InvalidConfig(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
	cfg := &config.Config{
		Files: []config.FileSpec{{Path: "/test/nonexistent.json"}},
	}
	s := NewServer(mfs, cfg, "/test")

	_, err := s.handleTokensBySource(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "asimonim://tokens/test"},
	})
	assert.Error(t, err)
}

func TestHandleTokensBySource_InvalidURI(t *testing.T) {
	s := newTestServer(t, "fixtures/draft/simple")
	_, err := s.handleTokensBySource(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "asimonim://tokens/"},
	})
	assert.Error(t, err)
}

func TestExtractURISuffix(t *testing.T) {
	tests := []struct {
		uri      string
		prefix   string
		expected string
		ok       bool
	}{
		{"asimonim://tokens/test", "asimonim://tokens/", "test", true},
		{"asimonim://token/src/color/primary", "asimonim://token/", "src/color/primary", true},
		{"asimonim://tokens/", "asimonim://tokens/", "", false},
		{"other://tokens/test", "asimonim://tokens/", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			result, ok := extractURISuffix(tt.uri, tt.prefix)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHandleTokenDetail(t *testing.T) {
	s := newTestServer(t, "fixtures/draft/simple")

	t.Run("existing token", func(t *testing.T) {
		result, err := s.handleTokenDetail(context.Background(), &mcp.ReadResourceRequest{
			Params: &mcp.ReadResourceParams{URI: "asimonim://token//test/tokens.json/color/primary"},
		})
		require.NoError(t, err)
		require.Len(t, result.Contents, 1)

		var detail tokenDetail
		require.NoError(t, json.Unmarshal([]byte(result.Contents[0].Text), &detail))
		assert.Equal(t, "color-primary", detail.Name)
		assert.Equal(t, "color.primary", detail.Path)
		assert.Equal(t, "color", detail.Type)
		assert.Equal(t, "--color-primary", detail.CSSVariableName)
		assert.Equal(t, "Primary brand color", detail.Description)
	})

	t.Run("resolved alias token", func(t *testing.T) {
		result, err := s.handleTokenDetail(context.Background(), &mcp.ReadResourceRequest{
			Params: &mcp.ReadResourceParams{URI: "asimonim://token//test/tokens.json/color/secondary"},
		})
		require.NoError(t, err)
		require.Len(t, result.Contents, 1)

		var detail tokenDetail
		require.NoError(t, json.Unmarshal([]byte(result.Contents[0].Text), &detail))
		// color.secondary resolves from {color.primary}
		assert.Equal(t, "color-secondary", detail.Name)
		assert.True(t, detail.IsResolved)
		assert.NotEmpty(t, detail.ResolutionChain)
	})

	t.Run("nonexistent token", func(t *testing.T) {
		_, err := s.handleTokenDetail(context.Background(), &mcp.ReadResourceRequest{
			Params: &mcp.ReadResourceParams{URI: "asimonim://token//test/tokens.json/nonexistent"},
		})
		assert.Error(t, err)
	})

	t.Run("invalid URI", func(t *testing.T) {
		_, err := s.handleTokenDetail(context.Background(), &mcp.ReadResourceRequest{
			Params: &mcp.ReadResourceParams{URI: "asimonim://token/"},
		})
		assert.Error(t, err)
	})
}
