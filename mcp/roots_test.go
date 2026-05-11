/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"bennypowers.dev/asimonim/internal/mapfs"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// projectFS creates a filesystem with a config and token file at /project,
// simulating a project root with .config/design-tokens.yaml.
func projectFS() *mapfs.MapFileSystem {
	mfs := mapfs.New()
	mfs.AddFile("/project/.config/design-tokens.yaml", "files:\n  - /project/tokens.json\n", 0644)
	mfs.AddFile("/project/tokens.json", `{
		"color": {
			"primary": {
				"$type": "color",
				"$value": "#FF0000"
			}
		}
	}`, 0644)
	return mfs
}

// cwdFS creates a filesystem with config at /mycwd for fallback tests.
func cwdFS() *mapfs.MapFileSystem {
	mfs := mapfs.New()
	mfs.AddFile("/mycwd/.config/design-tokens.yaml", "files:\n  - /mycwd/tokens.json\n", 0644)
	mfs.AddFile("/mycwd/tokens.json", `{
		"spacing": {
			"small": {
				"$type": "dimension",
				"$value": "4px"
			}
		}
	}`, 0644)
	return mfs
}

func TestSearchDiscoversConfigFromRoots(t *testing.T) {
	mfs := projectFS()
	s := NewServer(mfs, nil, "/elsewhere")
	s.listRoots = func(_ context.Context) ([]*mcp.Root, error) {
		return []*mcp.Root{{URI: "file:///project"}}, nil
	}

	result, _, err := s.handleSearch(context.Background(), nil, searchInput{
		Query: "primary",
	})
	require.NoError(t, err)
	assert.False(t, result.IsError, "expected search to succeed via roots discovery, got: %s", resultText(t, result))

	var tokens []tokenSummary
	require.NoError(t, json.Unmarshal([]byte(resultText(t, result)), &tokens))
	assert.Len(t, tokens, 1)
	assert.Equal(t, "color-primary", tokens[0].Name)
}

func TestValidateDiscoversConfigFromRoots(t *testing.T) {
	mfs := projectFS()
	s := NewServer(mfs, nil, "/elsewhere")
	s.listRoots = func(_ context.Context) ([]*mcp.Root, error) {
		return []*mcp.Root{{URI: "file:///project"}}, nil
	}

	result, _, err := s.handleValidate(context.Background(), nil, validateInput{})
	require.NoError(t, err)
	assert.False(t, result.IsError, "expected validate to succeed via roots discovery, got: %s", resultText(t, result))
	assert.Contains(t, resultText(t, result), "1 tokens")
}

func TestConvertDiscoversConfigFromRoots(t *testing.T) {
	mfs := projectFS()
	s := NewServer(mfs, nil, "/elsewhere")
	s.listRoots = func(_ context.Context) ([]*mcp.Root, error) {
		return []*mcp.Root{{URI: "file:///project"}}, nil
	}

	result, _, err := s.handleConvert(context.Background(), nil, convertInput{
		Format: "css",
	})
	require.NoError(t, err)
	assert.False(t, result.IsError, "expected convert to succeed via roots discovery, got: %s", resultText(t, result))
	assert.Contains(t, resultText(t, result), "--color-primary")
}

func TestRootsFallbackToCwd(t *testing.T) {
	mfs := cwdFS()
	s := NewServer(mfs, nil, "/mycwd")

	// listRoots returns empty (client doesn't support roots)
	s.listRoots = func(_ context.Context) ([]*mcp.Root, error) {
		return nil, nil
	}

	result, _, err := s.handleSearch(context.Background(), nil, searchInput{
		Query: "small",
	})
	require.NoError(t, err)
	assert.False(t, result.IsError, "expected fallback to cwd, got: %s", resultText(t, result))
}

func TestRootsUsesFirstRootOnly(t *testing.T) {
	mfs := projectFS()
	mfs.AddFile("/other/.config/design-tokens.yaml", "files:\n  - /other/tokens.json\n", 0644)
	mfs.AddFile("/other/tokens.json", `{
		"other": {
			"token": {
				"$type": "color",
				"$value": "#000"
			}
		}
	}`, 0644)

	s := NewServer(mfs, nil, "/elsewhere")
	s.listRoots = func(_ context.Context) ([]*mcp.Root, error) {
		return []*mcp.Root{
			{URI: "file:///project"},
			{URI: "file:///other"},
		}, nil
	}

	result, _, err := s.handleSearch(context.Background(), nil, searchInput{
		Query: "primary",
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)

	var tokens []tokenSummary
	require.NoError(t, json.Unmarshal([]byte(resultText(t, result)), &tokens))
	// color.primary from first root (/project), not "other.token" from /other
	assert.Len(t, tokens, 1)
	assert.Equal(t, "color-primary", tokens[0].Name)
}

func TestConfigCachedAfterFirstResolution(t *testing.T) {
	mfs := projectFS()
	s := NewServer(mfs, nil, "/elsewhere")

	calls := 0
	s.listRoots = func(_ context.Context) ([]*mcp.Root, error) {
		calls++
		return []*mcp.Root{{URI: "file:///project"}}, nil
	}

	result1, _, err := s.handleSearch(context.Background(), nil, searchInput{Query: "primary"})
	require.NoError(t, err)
	assert.False(t, result1.IsError)

	result2, _, err := s.handleSearch(context.Background(), nil, searchInput{Query: "primary"})
	require.NoError(t, err)
	assert.False(t, result2.IsError)

	assert.Equal(t, 1, calls, "listRoots should be called only once (cached)")
}

func TestRootsErrorFallbackToCwd(t *testing.T) {
	mfs := cwdFS()
	s := NewServer(mfs, nil, "/mycwd")

	// listRoots returns error (client disconnected, permission denied, etc.)
	s.listRoots = func(_ context.Context) ([]*mcp.Root, error) {
		return nil, fmt.Errorf("session error: client disconnected")
	}

	result, _, err := s.handleSearch(context.Background(), nil, searchInput{
		Query: "small",
	})
	require.NoError(t, err)
	assert.False(t, result.IsError, "expected fallback to cwd when listRoots errors, got: %s", resultText(t, result))
}

func TestRootsInvalidURIFallbackToCwd(t *testing.T) {
	mfs := cwdFS()

	tests := []struct {
		name string
		uri  string
	}{
		{"non-file scheme", "http://example.com/project"},
		{"empty path", "file://"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(mfs, nil, "/mycwd")
			s.listRoots = func(_ context.Context) ([]*mcp.Root, error) {
				return []*mcp.Root{{URI: tt.uri}}, nil
			}

			result, _, err := s.handleSearch(context.Background(), nil, searchInput{Query: "small"})
			require.NoError(t, err)
			assert.False(t, result.IsError, "should fall back to cwd for URI %q, got: %s", tt.uri, resultText(t, result))
		})
	}
}

func TestFileURIToPath(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		want    string
		wantErr bool
	}{
		{"unix path", "file:///home/user/project", "/home/user/project", false},
		{"windows drive letter", "file:///C:/Users/project", "C:/Users/project", false},
		{"UNC path", "file://server/share/project", "//server/share/project", false},
		{"percent-encoded space", "file:///my%20project", "/my project", false},
		{"non-file scheme", "http://example.com", "", true},
		{"empty path", "file://", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fileURIToPath(tt.uri)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCrossHandlerCaching(t *testing.T) {
	mfs := projectFS()
	s := NewServer(mfs, nil, "/elsewhere")

	var calls int32
	s.listRoots = func(_ context.Context) ([]*mcp.Root, error) {
		atomic.AddInt32(&calls, 1)
		return []*mcp.Root{{URI: "file:///project"}}, nil
	}

	// Call different handlers in sequence
	r1, _, err := s.handleValidate(context.Background(), nil, validateInput{})
	require.NoError(t, err)
	assert.False(t, r1.IsError)

	r2, _, err := s.handleConvert(context.Background(), nil, convertInput{Format: "css"})
	require.NoError(t, err)
	assert.False(t, r2.IsError)

	r3, _, err := s.handleSearch(context.Background(), nil, searchInput{Query: "primary"})
	require.NoError(t, err)
	assert.False(t, r3.IsError)

	assert.Equal(t, int32(1), atomic.LoadInt32(&calls), "listRoots should be called once across different handlers")
}

func TestConcurrentResolveConfig(t *testing.T) {
	mfs := projectFS()
	s := NewServer(mfs, nil, "/elsewhere")

	s.listRoots = func(_ context.Context) ([]*mcp.Root, error) {
		// Simulate slow RPC
		time.Sleep(10 * time.Millisecond)
		return []*mcp.Root{{URI: "file:///project"}}, nil
	}

	var wg sync.WaitGroup
	errs := make([]error, 20)
	results := make([]*mcp.CallToolResult, 20)

	for i := range 20 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			r, _, err := s.handleSearch(context.Background(), nil, searchInput{Query: "primary"})
			errs[idx] = err
			results[idx] = r
		}(i)
	}

	wg.Wait()

	for i := range 20 {
		require.NoError(t, errs[i], "goroutine %d", i)
		assert.False(t, results[i].IsError, "goroutine %d", i)
	}
}

func TestResolveConfigDoesNotHoldMutexDuringListRoots(t *testing.T) {
	mfs := projectFS()
	s := NewServer(mfs, nil, "/elsewhere")

	firstCallStarted := make(chan struct{})
	unblock := make(chan struct{})
	var callCount atomic.Int32

	s.listRoots = func(_ context.Context) ([]*mcp.Root, error) {
		n := callCount.Add(1)
		if n == 1 {
			close(firstCallStarted)
			<-unblock
		}
		return []*mcp.Root{{URI: "file:///project"}}, nil
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine 1: enters resolveConfig, calls listRoots, blocks on unblock channel
	go func() {
		defer wg.Done()
		s.resolveConfig(context.Background())
	}()

	// Wait for first listRoots call to start (proving mutex was released)
	<-firstCallStarted

	// Goroutine 2: if mutex released during RPC, this enters listRoots concurrently
	go func() {
		defer wg.Done()
		s.resolveConfig(context.Background())
	}()

	// Give goroutine 2 time to enter listRoots
	time.Sleep(10 * time.Millisecond)

	// Second call should have entered listRoots while first is blocked.
	// If mutex were held during RPC, callCount would still be 1.
	assert.GreaterOrEqual(t, callCount.Load(), int32(2),
		"second resolveConfig should enter listRoots while first is blocked (mutex not held during RPC)")

	close(unblock)
	wg.Wait()
}

func TestRootsNilListRootsResult(t *testing.T) {
	mfs := cwdFS()
	s := NewServer(mfs, nil, "/mycwd")

	// Simulate SDK returning nil result without error
	s.listRoots = func(_ context.Context) ([]*mcp.Root, error) {
		return nil, nil
	}

	result, _, err := s.handleSearch(context.Background(), nil, searchInput{Query: "small"})
	require.NoError(t, err)
	assert.False(t, result.IsError, "nil listRoots result should fall back to cwd")
}

func TestRootsPercentEncodedURI(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/my project/.config/design-tokens.yaml", "files:\n  - /my project/tokens.json\n", 0644)
	mfs.AddFile("/my project/tokens.json", `{
		"encoded": {
			"token": {
				"$type": "color",
				"$value": "#123456"
			}
		}
	}`, 0644)

	s := NewServer(mfs, nil, "/elsewhere")
	s.listRoots = func(_ context.Context) ([]*mcp.Root, error) {
		return []*mcp.Root{{URI: "file:///my%20project"}}, nil
	}

	result, _, err := s.handleSearch(context.Background(), nil, searchInput{Query: "token"})
	require.NoError(t, err)
	assert.False(t, result.IsError, "percent-encoded URI should decode to correct path, got: %s", resultText(t, result))
}

func TestRootsNilListRootsFuncFallbackToCwd(t *testing.T) {
	mfs := cwdFS()
	// listRoots field is nil (no session captured yet, no test injection)
	s := NewServer(mfs, nil, "/mycwd")

	result, _, err := s.handleSearch(context.Background(), nil, searchInput{Query: "small"})
	require.NoError(t, err)
	assert.False(t, result.IsError, "nil listRoots func should fall back to cwd")
}
