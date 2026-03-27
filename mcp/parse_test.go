/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package mcp

import (
	"testing"

	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/specifier"
	"bennypowers.dev/asimonim/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWorkspaceTokens(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
	cfg := &config.Config{
		Files: []config.FileSpec{{Path: "/test/tokens.json"}},
	}

	t.Run("parses all tokens", func(t *testing.T) {
		result, err := parseWorkspaceTokens(mfs, cfg, nil, "/test")
		require.NoError(t, err)
		// color.primary, color.secondary, spacing.small, spacing.medium, spacing.large
		assert.Len(t, result.AllTokens, 5)
		assert.Equal(t, schema.Draft, result.Version)
	})

	t.Run("groups by source", func(t *testing.T) {
		result, err := parseWorkspaceTokens(mfs, cfg, nil, "/test")
		require.NoError(t, err)
		assert.Len(t, result.Sources, 1)
		assert.Equal(t, "/test/tokens.json", result.Sources[0].Source)
		assert.Len(t, result.Sources[0].Tokens, 5)
	})

	t.Run("explicit files", func(t *testing.T) {
		result, err := parseWorkspaceTokens(mfs, cfg, []string{"/test/tokens.json"}, "/test")
		require.NoError(t, err)
		assert.Len(t, result.AllTokens, 5)
	})

	t.Run("resolves aliases", func(t *testing.T) {
		result, err := parseWorkspaceTokens(mfs, cfg, nil, "/test")
		require.NoError(t, err)
		// color.secondary references color.primary
		for _, tok := range result.AllTokens {
			if tok.Name == "color-secondary" {
				assert.True(t, tok.IsResolved)
				assert.NotEmpty(t, tok.ResolutionChain)
				return
			}
		}
		t.Fatal("color-secondary not found")
	})

	t.Run("no files error", func(t *testing.T) {
		emptyCfg := &config.Config{}
		_, err := parseWorkspaceTokens(mfs, emptyCfg, nil, "/test")
		assert.Error(t, err)
	})
}

func TestSourceLabel(t *testing.T) {
	tests := []struct {
		specifier string
		expected  string
	}{
		{"tokens.json", "tokens.json"},
		{"npm:@rhds/tokens/json/global.json", "@rhds/tokens"},
		{"npm:my-tokens/tokens.json", "my-tokens"},
		{"jsr:@design/tokens/tokens.json", "@design/tokens"},
		{"/path/to/tokens.json", "/path/to/tokens.json"},
	}

	for _, tt := range tests {
		t.Run(tt.specifier, func(t *testing.T) {
			rf := &specifier.ResolvedFile{Specifier: tt.specifier, Path: "/fake"}
			assert.Equal(t, tt.expected, sourceLabel(rf))
		})
	}
}

func TestParseWorkspaceTokens_WithSchema(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
	cfg := &config.Config{
		Files:  []config.FileSpec{{Path: "/test/tokens.json"}},
		Schema: "draft",
	}

	result, err := parseWorkspaceTokens(mfs, cfg, nil, "/test")
	require.NoError(t, err)
	assert.Equal(t, schema.Draft, result.Version)
	assert.Len(t, result.AllTokens, 5)
}

func TestParseWorkspaceTokens_ExplicitFilesOverrideConfig(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
	// Config points to a nonexistent file, but explicit files should be used
	cfg := &config.Config{
		Files: []config.FileSpec{{Path: "/test/nonexistent.json"}},
	}

	result, err := parseWorkspaceTokens(mfs, cfg, []string{"/test/tokens.json"}, "/test")
	require.NoError(t, err)
	assert.Len(t, result.AllTokens, 5)
}

func TestParseWorkspaceTokens_BadFile(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
	mfs.AddFile("/test/bad.json", `not json`, 0644)
	cfg := &config.Config{
		Files: []config.FileSpec{{Path: "/test/bad.json"}},
	}

	_, err := parseWorkspaceTokens(mfs, cfg, nil, "/test")
	assert.Error(t, err)
}

func TestParseWorkspaceTokens_ExplicitFileNotFound(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
	cfg := &config.Config{}

	_, err := parseWorkspaceTokens(mfs, cfg, []string{"/test/nonexistent.json"}, "/test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error reading")
}

func TestParseWorkspaceTokens_MultipleSourceLabels(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
	// Copy the fixture to a second path to test multiple sources
	data, err := mfs.ReadFile("/test/tokens.json")
	require.NoError(t, err)
	mfs.AddFile("/test/other.json", string(data), 0644)

	cfg := &config.Config{
		Files: []config.FileSpec{
			{Path: "/test/tokens.json"},
			{Path: "/test/other.json"},
		},
	}

	result, err := parseWorkspaceTokens(mfs, cfg, nil, "/test")
	require.NoError(t, err)
	assert.Len(t, result.Sources, 2)
	assert.Equal(t, "/test/tokens.json", result.Sources[0].Source)
	assert.Equal(t, "/test/other.json", result.Sources[1].Source)
}

func TestFilterTokens(t *testing.T) {
	tokens := testutil.ParseFixtureTokens(t, "fixtures/draft/simple", schema.Draft)

	t.Run("no filter", func(t *testing.T) {
		result := filterTokens(tokens, "", "")
		assert.Len(t, result, 5)
	})

	t.Run("filter by type", func(t *testing.T) {
		result := filterTokens(tokens, "color", "")
		assert.Len(t, result, 2)
	})

	t.Run("filter by group", func(t *testing.T) {
		result := filterTokens(tokens, "", "spacing")
		assert.Len(t, result, 3)
	})

	t.Run("filter by both", func(t *testing.T) {
		result := filterTokens(tokens, "dimension", "spacing")
		assert.Len(t, result, 3)
	})

	t.Run("no matches", func(t *testing.T) {
		result := filterTokens(tokens, "nonexistent", "")
		assert.Empty(t, result)
	})
}
