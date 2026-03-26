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

func TestHandleValidate(t *testing.T) {
	s := newTestServer(t, "fixtures/draft/simple")

	t.Run("valid tokens", func(t *testing.T) {
		result, _, err := s.handleValidate(context.Background(), nil, validateInput{})
		require.NoError(t, err)
		assert.False(t, result.IsError)
		text := resultText(t, result)
		assert.Contains(t, text, "All files valid.")
		// 5 tokens: color.primary, color.secondary, spacing.small, spacing.medium, spacing.large
		assert.Contains(t, text, "5 tokens")
	})

	t.Run("explicit files", func(t *testing.T) {
		result, _, err := s.handleValidate(context.Background(), nil, validateInput{
			Files: []string{"/test/tokens.json"},
		})
		require.NoError(t, err)
		assert.False(t, result.IsError)
		text := resultText(t, result)
		assert.Contains(t, text, "All files valid.")
	})
}

func TestHandleSearch(t *testing.T) {
	s := newTestServer(t, "fixtures/draft/simple")

	t.Run("search by name", func(t *testing.T) {
		result, _, err := s.handleSearch(context.Background(), nil, searchInput{
			Query:    "primary",
			NameOnly: true,
		})
		require.NoError(t, err)
		assert.False(t, result.IsError)

		var tokens []tokenSummary
		text := resultText(t, result)
		require.NoError(t, json.Unmarshal([]byte(text), &tokens))
		// color.primary: #FF6B35
		assert.Len(t, tokens, 1)
		assert.Equal(t, "color-primary", tokens[0].Name)
		assert.Equal(t, "color", tokens[0].Type)
	})

	t.Run("search by type filter", func(t *testing.T) {
		result, _, err := s.handleSearch(context.Background(), nil, searchInput{
			Query: "spacing",
			Type:  "dimension",
		})
		require.NoError(t, err)

		var tokens []tokenSummary
		text := resultText(t, result)
		require.NoError(t, json.Unmarshal([]byte(text), &tokens))
		// spacing.small, spacing.medium, spacing.large
		assert.Len(t, tokens, 3)
		for _, tok := range tokens {
			assert.Equal(t, "dimension", tok.Type)
		}
	})

	t.Run("search by group filter", func(t *testing.T) {
		result, _, err := s.handleSearch(context.Background(), nil, searchInput{
			Query: "color",
			Group: "color",
		})
		require.NoError(t, err)

		var tokens []tokenSummary
		text := resultText(t, result)
		require.NoError(t, json.Unmarshal([]byte(text), &tokens))
		// color.primary, color.secondary
		assert.Len(t, tokens, 2)
	})

	t.Run("regex search", func(t *testing.T) {
		result, _, err := s.handleSearch(context.Background(), nil, searchInput{
			Query: "^spacing",
			Regex: true,
		})
		require.NoError(t, err)

		var tokens []tokenSummary
		text := resultText(t, result)
		require.NoError(t, json.Unmarshal([]byte(text), &tokens))
		assert.Len(t, tokens, 3)
	})

	t.Run("empty query", func(t *testing.T) {
		result, _, err := s.handleSearch(context.Background(), nil, searchInput{})
		require.NoError(t, err)
		assert.True(t, result.IsError)
	})

	t.Run("name only", func(t *testing.T) {
		result, _, err := s.handleSearch(context.Background(), nil, searchInput{
			Query:    "primary",
			NameOnly: true,
		})
		require.NoError(t, err)

		var tokens []tokenSummary
		text := resultText(t, result)
		require.NoError(t, json.Unmarshal([]byte(text), &tokens))
		assert.Len(t, tokens, 1)
	})

	t.Run("value only", func(t *testing.T) {
		result, _, err := s.handleSearch(context.Background(), nil, searchInput{
			Query:     "#FF6B35",
			ValueOnly: true,
		})
		require.NoError(t, err)

		var tokens []tokenSummary
		text := resultText(t, result)
		require.NoError(t, json.Unmarshal([]byte(text), &tokens))
		// color.primary has this value, color.secondary resolves to it
		assert.GreaterOrEqual(t, len(tokens), 1)
	})
}

func TestHandleConvert(t *testing.T) {
	s := newTestServer(t, "fixtures/draft/simple")

	t.Run("convert to css", func(t *testing.T) {
		result, _, err := s.handleConvert(context.Background(), nil, convertInput{
			Format: "css",
		})
		require.NoError(t, err)
		assert.False(t, result.IsError)
		text := resultText(t, result)
		assert.Contains(t, text, "--color-primary")
		assert.Contains(t, text, "--spacing-small")
	})

	t.Run("convert to scss", func(t *testing.T) {
		result, _, err := s.handleConvert(context.Background(), nil, convertInput{
			Format: "scss",
		})
		require.NoError(t, err)
		assert.False(t, result.IsError)
		text := resultText(t, result)
		assert.Contains(t, text, "$color-primary")
	})

	t.Run("convert with prefix", func(t *testing.T) {
		result, _, err := s.handleConvert(context.Background(), nil, convertInput{
			Format: "css",
			Prefix: "my",
		})
		require.NoError(t, err)
		assert.False(t, result.IsError)
		text := resultText(t, result)
		assert.Contains(t, text, "--my-color-primary")
	})

	t.Run("convert with type filter", func(t *testing.T) {
		result, _, err := s.handleConvert(context.Background(), nil, convertInput{
			Format: "css",
			Type:   "color",
		})
		require.NoError(t, err)
		assert.False(t, result.IsError)
		text := resultText(t, result)
		assert.Contains(t, text, "--color-primary")
		assert.NotContains(t, text, "--spacing")
	})

	t.Run("convert with group filter", func(t *testing.T) {
		result, _, err := s.handleConvert(context.Background(), nil, convertInput{
			Format: "css",
			Group:  "spacing",
		})
		require.NoError(t, err)
		assert.False(t, result.IsError)
		text := resultText(t, result)
		assert.Contains(t, text, "--spacing-small")
		assert.NotContains(t, text, "--color")
	})

	t.Run("empty format", func(t *testing.T) {
		result, _, err := s.handleConvert(context.Background(), nil, convertInput{})
		require.NoError(t, err)
		assert.True(t, result.IsError)
	})

	t.Run("invalid format", func(t *testing.T) {
		result, _, err := s.handleConvert(context.Background(), nil, convertInput{
			Format: "invalid",
		})
		require.NoError(t, err)
		assert.True(t, result.IsError)
	})
}

func TestHandleValidate_Strict(t *testing.T) {
	// Create a fixture with a deprecated token
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
	// Add a file with a deprecated token
	mfs.AddFile("/test/deprecated.json", `{
		"old-color": {
			"$type": "color",
			"$value": "#000",
			"$deprecated": true
		}
	}`, 0644)
	cfg := &config.Config{
		Files: []config.FileSpec{{Path: "/test/deprecated.json"}},
	}
	s := NewServer(mfs, cfg, "/test")

	t.Run("strict fails on deprecation", func(t *testing.T) {
		result, _, err := s.handleValidate(context.Background(), nil, validateInput{
			Strict: true,
		})
		require.NoError(t, err)
		assert.True(t, result.IsError)
		text := resultText(t, result)
		assert.Contains(t, text, "strict mode")
	})

	t.Run("non-strict passes with deprecation", func(t *testing.T) {
		result, _, err := s.handleValidate(context.Background(), nil, validateInput{})
		require.NoError(t, err)
		assert.False(t, result.IsError)
		text := resultText(t, result)
		assert.Contains(t, text, "deprecated")
		assert.Contains(t, text, "All files valid.")
	})
}

func TestHandleValidate_CircularReference(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
	mfs.AddFile("/test/circular.json", `{
		"a": {
			"$type": "color",
			"$value": "{b}"
		},
		"b": {
			"$type": "color",
			"$value": "{a}"
		}
	}`, 0644)
	cfg := &config.Config{
		Files: []config.FileSpec{{Path: "/test/circular.json"}},
	}
	s := NewServer(mfs, cfg, "/test")

	result, _, err := s.handleValidate(context.Background(), nil, validateInput{})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	text := resultText(t, result)
	// Circular reference caught during alias resolution in parseWorkspaceTokens
	assert.Contains(t, text, "circular reference")
}

func TestHandleValidate_InvalidFile(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
	cfg := &config.Config{
		Files: []config.FileSpec{{Path: "/test/nonexistent.json"}},
	}
	s := NewServer(mfs, cfg, "/test")

	result, _, err := s.handleValidate(context.Background(), nil, validateInput{})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestHandleSearch_InvalidRegex(t *testing.T) {
	s := newTestServer(t, "fixtures/draft/simple")
	result, _, err := s.handleSearch(context.Background(), nil, searchInput{
		Query: "[invalid",
		Regex: true,
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	text := resultText(t, result)
	assert.Contains(t, text, "Invalid regex")
}

func TestHandleConvert_InvalidFile(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
	cfg := &config.Config{
		Files: []config.FileSpec{{Path: "/test/nonexistent.json"}},
	}
	s := NewServer(mfs, cfg, "/test")

	result, _, err := s.handleConvert(context.Background(), nil, convertInput{
		Format: "css",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestHandleSearch_InvalidFile(t *testing.T) {
	mfs := testutil.NewFixtureFS(t, "fixtures/draft/simple", "/test")
	cfg := &config.Config{
		Files: []config.FileSpec{{Path: "/test/nonexistent.json"}},
	}
	s := NewServer(mfs, cfg, "/test")

	result, _, err := s.handleSearch(context.Background(), nil, searchInput{
		Query: "test",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

// resultText extracts the text from the first content item of a CallToolResult.
func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	require.NotNil(t, result)
	require.NotEmpty(t, result.Content)
	data, err := json.Marshal(result.Content[0])
	require.NoError(t, err)
	var content struct {
		Text string `json:"text"`
	}
	require.NoError(t, json.Unmarshal(data, &content))
	return content.Text
}

