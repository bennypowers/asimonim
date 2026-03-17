package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizePath(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()
	workspaceRoot := tmpDir

	// Create a mock node_modules structure
	nodeModulesDir := filepath.Join(tmpDir, "node_modules")
	require.NoError(t, os.MkdirAll(nodeModulesDir, 0o755))

	// Create a mock package with tokens and package.json with exports
	mockPkgDir := filepath.Join(nodeModulesDir, "@design-system", "tokens")
	require.NoError(t, os.MkdirAll(mockPkgDir, 0o755))
	tokensFile := filepath.Join(mockPkgDir, "tokens.json")
	require.NoError(t, os.WriteFile(tokensFile, []byte(`{"color": {}}`), 0o644))

	// Create package.json with exports
	packageJSON := `{
		"name": "@design-system/tokens",
		"version": "1.0.0",
		"exports": {
			".": "./tokens.json",
			"./tokens": "./tokens.json",
			"./dist/*": "./dist/*.json"
		}
	}`
	require.NoError(t, os.WriteFile(filepath.Join(mockPkgDir, "package.json"), []byte(packageJSON), 0o644))

	// Create dist directory for pattern matching
	distDir := filepath.Join(mockPkgDir, "dist")
	require.NoError(t, os.MkdirAll(distDir, 0o755))
	colorsFile := filepath.Join(distDir, "colors.json")
	require.NoError(t, os.WriteFile(colorsFile, []byte(`{"primary": "#ff0000"}`), 0o644))

	tests := []struct {
		name          string
		path          string
		workspaceRoot string
		expected      string
		wantErr       bool
		skipOnCI      bool // Skip tests that require HOME env var
	}{
		{
			name:          "absolute path unchanged",
			path:          "/absolute/path/to/tokens.json",
			workspaceRoot: workspaceRoot,
			expected:      "/absolute/path/to/tokens.json",
		},
		{
			name:          "relative path resolved",
			path:          "./relative/tokens.json",
			workspaceRoot: workspaceRoot,
			expected:      filepath.Join(workspaceRoot, "relative", "tokens.json"),
		},
		{
			name:          "relative path without dot",
			path:          "relative/tokens.json",
			workspaceRoot: workspaceRoot,
			expected:      filepath.Join(workspaceRoot, "relative", "tokens.json"),
		},
		{
			name:          "home directory expansion",
			path:          "~/my-tokens.json",
			workspaceRoot: workspaceRoot,
			expected:      filepath.Join(os.Getenv("HOME"), "my-tokens.json"),
			skipOnCI:      os.Getenv("HOME") == "",
		},
		{
			name:          "npm: scoped package with direct path",
			path:          "npm:@design-system/tokens/tokens.json",
			workspaceRoot: workspaceRoot,
			expected:      tokensFile,
		},
		{
			name:          "npm: package main entry (uses exports)",
			path:          "npm:@design-system/tokens",
			workspaceRoot: workspaceRoot,
			expected:      tokensFile, // Resolves via exports "." field
		},
		{
			name:          "npm: package with export path",
			path:          "npm:@design-system/tokens/tokens",
			workspaceRoot: workspaceRoot,
			expected:      tokensFile, // Resolves via exports "./tokens" field
		},
		{
			name:          "npm: package with wildcard export",
			path:          "npm:@design-system/tokens/dist/colors",
			workspaceRoot: workspaceRoot,
			expected:      colorsFile, // Resolves via exports "./dist/*" pattern
		},
		{
			name:          "npm: unscoped package",
			path:          "npm:design-tokens/tokens.json",
			workspaceRoot: workspaceRoot,
			wantErr:       true, // Package doesn't exist in our test setup
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnCI {
				t.Skip("Skipping test that requires HOME environment variable")
			}

			got, err := normalizePath(tt.path, tt.workspaceRoot)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestNormalizePathErrors(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name          string
		path          string
		workspaceRoot string
		errContains   string
		wantErr       bool
	}{
		{
			name:          "npm: package not found",
			path:          "npm:nonexistent-package/tokens.json",
			workspaceRoot: tmpDir,
			wantErr:       true,
			errContains:   "not found",
		},
		{
			name:          "npm: invalid package name",
			path:          "npm:",
			workspaceRoot: tmpDir,
			wantErr:       true,
			errContains:   "invalid npm package",
		},
		{
			name:          "npm: empty package name",
			path:          "npm:/tokens.json",
			workspaceRoot: tmpDir,
			wantErr:       true,
			errContains:   "invalid npm package",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizePath(tt.path, tt.workspaceRoot)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Empty(t, got)
				return
			}

			require.NoError(t, err)
		})
	}
}

// TestResolveNpmPath_PathTraversal tests security fixes for path traversal vulnerabilities
func TestResolveNpmPath_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceRoot := tmpDir

	// Create a minimal node_modules structure
	nodeModulesDir := filepath.Join(tmpDir, "node_modules")
	require.NoError(t, os.MkdirAll(nodeModulesDir, 0o755))

	// Create a legitimate package for comparison tests
	legitimatePkgDir := filepath.Join(nodeModulesDir, "legitimate-package")
	require.NoError(t, os.MkdirAll(legitimatePkgDir, 0o755))
	legitimateFile := filepath.Join(legitimatePkgDir, "tokens.json")
	require.NoError(t, os.WriteFile(legitimateFile, []byte(`{}`), 0o644))

	tests := []struct {
		name        string
		npmPath     string
		shouldError bool
		errContains string
	}{
		{
			name:        "valid unscoped package",
			npmPath:     "legitimate-package/tokens.json",
			shouldError: false,
		},
		{
			name:        "path traversal in package name - dotdot",
			npmPath:     "../../etc/passwd",
			shouldError: true,
			errContains: "path traversal not allowed",
		},
		{
			name:        "path traversal in package name - single dotdot",
			npmPath:     "../sensitive-file.txt",
			shouldError: true,
			errContains: "path traversal not allowed",
		},
		{
			name:        "path traversal with scoped package format",
			npmPath:     "@../../../etc/passwd",
			shouldError: true,
			errContains: "path traversal not allowed",
		},
		{
			name:        "path traversal in scope name",
			npmPath:     "@../../etc/passwd",
			shouldError: true,
			errContains: "path traversal not allowed",
		},
		{
			name:        "dots in legitimate package name (valid)",
			npmPath:     "my.package.name/tokens.json",
			shouldError: true, // Will fail because package doesn't exist, but NOT because of traversal
			errContains: "not found",
		},
		{
			name:        "dotdot in subpath (not package name)",
			npmPath:     "legitimate-package/../../../etc/passwd",
			shouldError: true,
			errContains: "not found", // This tests that subpath traversal is blocked by file system checks
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveNpmPath(tt.npmPath, workspaceRoot)

			if tt.shouldError {
				require.Error(t, err, "Expected error for npm path: %s", tt.npmPath)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains,
						"Error message should contain '%s' for npm path: %s", tt.errContains, tt.npmPath)
				}
				assert.Empty(t, got, "Should return empty path on error")
				return
			}

			require.NoError(t, err, "Should not error for valid npm path: %s", tt.npmPath)
			assert.NotEmpty(t, got, "Should return non-empty path for valid npm path")
		})
	}
}

// TestResolvePackageEntry tests resolvePackageEntry with various package.json configurations
func TestResolvePackageEntry(t *testing.T) {
	t.Run("resolves via exports '.' entry", func(t *testing.T) {
		tmpDir := t.TempDir()
		pkgDir := filepath.Join(tmpDir, "pkg")
		require.NoError(t, os.MkdirAll(pkgDir, 0o755))

		// Create the target file
		targetFile := filepath.Join(pkgDir, "dist", "index.js")
		require.NoError(t, os.MkdirAll(filepath.Dir(targetFile), 0o755))
		require.NoError(t, os.WriteFile(targetFile, []byte("export default {}"), 0o644))

		// package.json with exports "." pointing to dist/index.js
		pkgJSON := `{"exports": {".": "./dist/index.js"}}`
		require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(pkgJSON), 0o644))

		got, err := resolvePackageEntry(pkgDir, ".")
		require.NoError(t, err)
		assert.Equal(t, targetFile, got)
	})

	t.Run("falls back to main field when exports does not match", func(t *testing.T) {
		tmpDir := t.TempDir()
		pkgDir := filepath.Join(tmpDir, "pkg")
		require.NoError(t, os.MkdirAll(pkgDir, 0o755))

		// Create the main entry file
		mainFile := filepath.Join(pkgDir, "lib", "main.js")
		require.NoError(t, os.MkdirAll(filepath.Dir(mainFile), 0o755))
		require.NoError(t, os.WriteFile(mainFile, []byte("module.exports = {}"), 0o644))

		// package.json with main field only (no exports)
		pkgJSON := `{"main": "lib/main.js"}`
		require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(pkgJSON), 0o644))

		got, err := resolvePackageEntry(pkgDir, ".")
		require.NoError(t, err)
		assert.Equal(t, mainFile, got)
	})

	t.Run("falls back to index.js when no main field", func(t *testing.T) {
		tmpDir := t.TempDir()
		pkgDir := filepath.Join(tmpDir, "pkg")
		require.NoError(t, os.MkdirAll(pkgDir, 0o755))

		// Create index.js
		indexFile := filepath.Join(pkgDir, "index.js")
		require.NoError(t, os.WriteFile(indexFile, []byte("module.exports = {}"), 0o644))

		// package.json with no exports or main
		pkgJSON := `{"name": "test-pkg"}`
		require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(pkgJSON), 0o644))

		got, err := resolvePackageEntry(pkgDir, ".")
		require.NoError(t, err)
		assert.Equal(t, indexFile, got)
	})

	t.Run("returns error when no entry point found", func(t *testing.T) {
		tmpDir := t.TempDir()
		pkgDir := filepath.Join(tmpDir, "pkg")
		require.NoError(t, os.MkdirAll(pkgDir, 0o755))

		// package.json with no exports, no main, and no index.js
		pkgJSON := `{"name": "empty-pkg"}`
		require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(pkgJSON), 0o644))

		_, err := resolvePackageEntry(pkgDir, ".")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no entry point found")
	})

	t.Run("returns error when main file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		pkgDir := filepath.Join(tmpDir, "pkg")
		require.NoError(t, os.MkdirAll(pkgDir, 0o755))

		// package.json with main pointing to nonexistent file, no index.js
		pkgJSON := `{"main": "nonexistent.js"}`
		require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(pkgJSON), 0o644))

		_, err := resolvePackageEntry(pkgDir, ".")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no entry point found")
	})

	t.Run("returns error when package.json is missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		pkgDir := filepath.Join(tmpDir, "pkg")
		require.NoError(t, os.MkdirAll(pkgDir, 0o755))
		// No package.json created

		_, err := resolvePackageEntry(pkgDir, ".")
		require.Error(t, err)
	})
}

// TestResolveExports tests resolveExports with various export configurations
func TestResolveExports(t *testing.T) {
	t.Run("string exports resolves for '.' entry", func(t *testing.T) {
		tmpDir := t.TempDir()
		targetFile := filepath.Join(tmpDir, "dist", "index.js")
		require.NoError(t, os.MkdirAll(filepath.Dir(targetFile), 0o755))
		require.NoError(t, os.WriteFile(targetFile, []byte(""), 0o644))

		got, err := resolveExports(tmpDir, "./dist/index.js", ".")
		require.NoError(t, err)
		assert.Equal(t, targetFile, got)
	})

	t.Run("string exports resolves for './' entry", func(t *testing.T) {
		tmpDir := t.TempDir()
		targetFile := filepath.Join(tmpDir, "dist", "index.js")
		require.NoError(t, os.MkdirAll(filepath.Dir(targetFile), 0o755))
		require.NoError(t, os.WriteFile(targetFile, []byte(""), 0o644))

		got, err := resolveExports(tmpDir, "./dist/index.js", "./")
		require.NoError(t, err)
		assert.Equal(t, targetFile, got)
	})

	t.Run("string exports does not match non-root path", func(t *testing.T) {
		tmpDir := t.TempDir()

		_, err := resolveExports(tmpDir, "./dist/index.js", "./tokens")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match")
	})

	t.Run("string exports fails when target file missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		// No file created at the export target

		_, err := resolveExports(tmpDir, "./nonexistent.js", ".")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match")
	})

	t.Run("map exports with exact match", func(t *testing.T) {
		tmpDir := t.TempDir()
		targetFile := filepath.Join(tmpDir, "dist", "tokens.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(targetFile), 0o755))
		require.NoError(t, os.WriteFile(targetFile, []byte("{}"), 0o644))

		exports := map[string]any{
			"./tokens": "./dist/tokens.json",
		}
		got, err := resolveExports(tmpDir, exports, "./tokens")
		require.NoError(t, err)
		assert.Equal(t, targetFile, got)
	})

	t.Run("map exports adds './' prefix for matching", func(t *testing.T) {
		tmpDir := t.TempDir()
		targetFile := filepath.Join(tmpDir, "dist", "tokens.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(targetFile), 0o755))
		require.NoError(t, os.WriteFile(targetFile, []byte("{}"), 0o644))

		exports := map[string]any{
			"./tokens": "./dist/tokens.json",
		}
		// Request without "./" prefix -- should try prefixed match
		got, err := resolveExports(tmpDir, exports, "tokens")
		require.NoError(t, err)
		assert.Equal(t, targetFile, got)
	})

	t.Run("map exports with wildcard pattern", func(t *testing.T) {
		tmpDir := t.TempDir()
		targetFile := filepath.Join(tmpDir, "dist", "colors.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(targetFile), 0o755))
		require.NoError(t, os.WriteFile(targetFile, []byte("{}"), 0o644))

		exports := map[string]any{
			"./*": "./dist/*.json",
		}
		got, err := resolveExports(tmpDir, exports, "./colors")
		require.NoError(t, err)
		assert.Equal(t, targetFile, got)
	})

	t.Run("map exports returns error when no match found", func(t *testing.T) {
		tmpDir := t.TempDir()

		exports := map[string]any{
			"./tokens": "./dist/tokens.json",
		}
		_, err := resolveExports(tmpDir, exports, "./nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no export found")
	})

	t.Run("unsupported exports type returns error", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Pass an integer as exports (unsupported)
		_, err := resolveExports(tmpDir, 42, ".")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported exports type")
	})
}

// TestResolveExportTarget tests resolveExportTarget with various target types
func TestResolveExportTarget(t *testing.T) {
	t.Run("string target resolves to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		targetFile := filepath.Join(tmpDir, "dist", "index.js")
		require.NoError(t, os.MkdirAll(filepath.Dir(targetFile), 0o755))
		require.NoError(t, os.WriteFile(targetFile, []byte(""), 0o644))

		got, err := resolveExportTarget(tmpDir, "./dist/index.js")
		require.NoError(t, err)
		assert.Equal(t, targetFile, got)
	})

	t.Run("string target returns error when file missing", func(t *testing.T) {
		tmpDir := t.TempDir()

		_, err := resolveExportTarget(tmpDir, "./nonexistent.js")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "export target not found")
	})

	t.Run("conditional exports resolves 'default' first", func(t *testing.T) {
		tmpDir := t.TempDir()
		defaultFile := filepath.Join(tmpDir, "dist", "default.js")
		require.NoError(t, os.MkdirAll(filepath.Dir(defaultFile), 0o755))
		require.NoError(t, os.WriteFile(defaultFile, []byte(""), 0o644))

		target := map[string]any{
			"import":  "./dist/index.mjs",
			"require": "./dist/index.cjs",
			"default": "./dist/default.js",
		}
		got, err := resolveExportTarget(tmpDir, target)
		require.NoError(t, err)
		assert.Equal(t, defaultFile, got)
	})

	t.Run("conditional exports falls back to 'require'", func(t *testing.T) {
		tmpDir := t.TempDir()
		requireFile := filepath.Join(tmpDir, "dist", "index.cjs")
		require.NoError(t, os.MkdirAll(filepath.Dir(requireFile), 0o755))
		require.NoError(t, os.WriteFile(requireFile, []byte(""), 0o644))

		target := map[string]any{
			"import":  "./dist/index.mjs",
			"require": "./dist/index.cjs",
		}
		got, err := resolveExportTarget(tmpDir, target)
		require.NoError(t, err)
		assert.Equal(t, requireFile, got)
	})

	t.Run("conditional exports falls back to 'import'", func(t *testing.T) {
		tmpDir := t.TempDir()
		importFile := filepath.Join(tmpDir, "dist", "index.mjs")
		require.NoError(t, os.MkdirAll(filepath.Dir(importFile), 0o755))
		require.NoError(t, os.WriteFile(importFile, []byte(""), 0o644))

		target := map[string]any{
			"import": "./dist/index.mjs",
		}
		got, err := resolveExportTarget(tmpDir, target)
		require.NoError(t, err)
		assert.Equal(t, importFile, got)
	})

	t.Run("conditional exports returns error when no suitable export", func(t *testing.T) {
		tmpDir := t.TempDir()

		target := map[string]any{
			"node": "./dist/node.js",
		}
		_, err := resolveExportTarget(tmpDir, target)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no suitable conditional export found")
	})

	t.Run("unsupported target type returns error", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Pass an integer (unsupported)
		_, err := resolveExportTarget(tmpDir, 42)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported export target type")
	})

	t.Run("nested conditional exports resolves recursively", func(t *testing.T) {
		tmpDir := t.TempDir()
		targetFile := filepath.Join(tmpDir, "dist", "tokens.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(targetFile), 0o755))
		require.NoError(t, os.WriteFile(targetFile, []byte("{}"), 0o644))

		// Nested: "default" points to another map with "default"
		target := map[string]any{
			"default": map[string]any{
				"default": "./dist/tokens.json",
			},
		}
		got, err := resolveExportTarget(tmpDir, target)
		require.NoError(t, err)
		assert.Equal(t, targetFile, got)
	})
}

// TestMatchExportPattern tests matchExportPattern edge cases
func TestMatchExportPattern(t *testing.T) {
	tests := []struct {
		name         string
		pattern      string
		requested    string
		wantMatch    bool
		wantSubst    string
	}{
		{
			name:      "no wildcard in pattern",
			pattern:   "./tokens",
			requested: "./tokens",
			wantMatch: false,
			wantSubst: "",
		},
		{
			name:      "multiple wildcards not supported",
			pattern:   "./*/*.json",
			requested: "./a/b.json",
			wantMatch: false,
			wantSubst: "",
		},
		{
			name:      "simple wildcard match",
			pattern:   "./*",
			requested: "./colors",
			wantMatch: true,
			wantSubst: "colors",
		},
		{
			name:      "wildcard with suffix",
			pattern:   "./*.json",
			requested: "./colors.json",
			wantMatch: true,
			wantSubst: "colors",
		},
		{
			name:      "wildcard with prefix and suffix",
			pattern:   "./dist/*.json",
			requested: "./dist/tokens.json",
			wantMatch: true,
			wantSubst: "tokens",
		},
		{
			name:      "no match when prefix differs",
			pattern:   "./dist/*",
			requested: "./lib/tokens",
			wantMatch: false,
			wantSubst: "",
		},
		{
			name:      "no match when suffix differs",
			pattern:   "./*.json",
			requested: "./tokens.yaml",
			wantMatch: false,
			wantSubst: "",
		},
		{
			name:      "empty substitution",
			pattern:   "./*",
			requested: "./",
			wantMatch: true,
			wantSubst: "",
		},
		{
			name:      "prefix longer than requested path",
			pattern:   "./very/long/prefix/*",
			requested: "./short",
			wantMatch: false,
			wantSubst: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, subst := matchExportPattern(tt.pattern, tt.requested)
			assert.Equal(t, tt.wantMatch, matched, "matched")
			assert.Equal(t, tt.wantSubst, subst, "substitution")
		})
	}
}

// TestExpandPattern tests expandPattern with various input types
func TestExpandPattern(t *testing.T) {
	t.Run("string pattern expands wildcard", func(t *testing.T) {
		result := expandPattern("./dist/*.json", "colors")
		assert.Equal(t, "./dist/colors.json", result)
	})

	t.Run("string pattern with no wildcard returns as-is", func(t *testing.T) {
		result := expandPattern("./dist/tokens.json", "colors")
		assert.Equal(t, "./dist/tokens.json", result)
	})

	t.Run("non-string pattern returns unchanged", func(t *testing.T) {
		// e.g., a map target should pass through unexpanded
		input := map[string]any{"default": "./dist/*.json"}
		result := expandPattern(input, "colors")
		assert.Equal(t, input, result)
	})

	t.Run("nil pattern returns nil", func(t *testing.T) {
		result := expandPattern(nil, "colors")
		assert.Nil(t, result)
	})

	t.Run("integer pattern returns unchanged", func(t *testing.T) {
		result := expandPattern(42, "colors")
		assert.Equal(t, 42, result)
	})
}

// TestResolvePackageSubpath tests resolvePackageSubpath
func TestResolvePackageSubpath(t *testing.T) {
	t.Run("resolves via exports field", func(t *testing.T) {
		tmpDir := t.TempDir()
		pkgDir := filepath.Join(tmpDir, "pkg")
		require.NoError(t, os.MkdirAll(pkgDir, 0o755))

		targetFile := filepath.Join(pkgDir, "dist", "tokens.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(targetFile), 0o755))
		require.NoError(t, os.WriteFile(targetFile, []byte("{}"), 0o644))

		pkgJSON := `{"exports": {"./tokens": "./dist/tokens.json"}}`
		require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(pkgJSON), 0o644))

		got, err := resolvePackageSubpath(pkgDir, "tokens")
		require.NoError(t, err)
		assert.Equal(t, targetFile, got)
	})

	t.Run("falls back to direct file when exports does not match", func(t *testing.T) {
		tmpDir := t.TempDir()
		pkgDir := filepath.Join(tmpDir, "pkg")
		require.NoError(t, os.MkdirAll(pkgDir, 0o755))

		// Create direct file
		directFile := filepath.Join(pkgDir, "tokens.json")
		require.NoError(t, os.WriteFile(directFile, []byte("{}"), 0o644))

		// package.json with exports that don't match our subpath
		pkgJSON := `{"exports": {"./other": "./other.json"}}`
		require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(pkgJSON), 0o644))

		got, err := resolvePackageSubpath(pkgDir, "tokens.json")
		require.NoError(t, err)
		assert.Equal(t, directFile, got)
	})

	t.Run("falls back to direct file when no package.json", func(t *testing.T) {
		tmpDir := t.TempDir()
		pkgDir := filepath.Join(tmpDir, "pkg")
		require.NoError(t, os.MkdirAll(pkgDir, 0o755))

		directFile := filepath.Join(pkgDir, "tokens.json")
		require.NoError(t, os.WriteFile(directFile, []byte("{}"), 0o644))

		// No package.json
		got, err := resolvePackageSubpath(pkgDir, "tokens.json")
		require.NoError(t, err)
		assert.Equal(t, directFile, got)
	})

	t.Run("returns error when no package.json and file not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		pkgDir := filepath.Join(tmpDir, "pkg")
		require.NoError(t, os.MkdirAll(pkgDir, 0o755))

		// No package.json, no file
		_, err := resolvePackageSubpath(pkgDir, "nonexistent.json")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
	})

	t.Run("returns error when exports and direct file both fail", func(t *testing.T) {
		tmpDir := t.TempDir()
		pkgDir := filepath.Join(tmpDir, "pkg")
		require.NoError(t, os.MkdirAll(pkgDir, 0o755))

		pkgJSON := `{"exports": {"./other": "./other.json"}}`
		require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(pkgJSON), 0o644))

		_, err := resolvePackageSubpath(pkgDir, "nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
	})
}

// TestReadPackageJSON tests readPackageJSON error handling
func TestReadPackageJSON(t *testing.T) {
	t.Run("parses valid package.json", func(t *testing.T) {
		tmpDir := t.TempDir()
		pkgJSON := `{"main": "index.js", "exports": "./dist/index.js"}`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(pkgJSON), 0o644))

		pkg, err := readPackageJSON(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, "index.js", pkg.Main)
		assert.Equal(t, "./dist/index.js", pkg.Exports)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{invalid"), 0o644))

		_, err := readPackageJSON(tmpDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse package.json")
	})

	t.Run("returns error for missing package.json", func(t *testing.T) {
		tmpDir := t.TempDir()

		_, err := readPackageJSON(tmpDir)
		require.Error(t, err)
	})
}

// TestResolveNpmPath_UnscopedPackage tests unscoped package resolution
func TestResolveNpmPath_UnscopedPackage(t *testing.T) {
	tmpDir := t.TempDir()
	nodeModulesDir := filepath.Join(tmpDir, "node_modules")
	require.NoError(t, os.MkdirAll(nodeModulesDir, 0o755))

	// Create unscoped package with exports
	pkgDir := filepath.Join(nodeModulesDir, "my-tokens")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))

	tokensFile := filepath.Join(pkgDir, "tokens.json")
	require.NoError(t, os.WriteFile(tokensFile, []byte("{}"), 0o644))

	pkgJSON := `{"exports": {".": "./tokens.json", "./colors": "./tokens.json"}}`
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(pkgJSON), 0o644))

	t.Run("resolves main entry for unscoped package", func(t *testing.T) {
		got, err := resolveNpmPath("my-tokens", tmpDir)
		require.NoError(t, err)
		assert.Equal(t, tokensFile, got)
	})

	t.Run("resolves subpath for unscoped package", func(t *testing.T) {
		got, err := resolveNpmPath("my-tokens/colors", tmpDir)
		require.NoError(t, err)
		assert.Equal(t, tokensFile, got)
	})

	t.Run("resolves direct file for unscoped package", func(t *testing.T) {
		got, err := resolveNpmPath("my-tokens/tokens.json", tmpDir)
		require.NoError(t, err)
		assert.Equal(t, tokensFile, got)
	})
}

// TestResolveNpmPath_ScopedPackageNoSubpath tests scoped package with only one segment
func TestResolveNpmPath_ScopedPackageEdgeCases(t *testing.T) {
	t.Run("rejects scoped package with only scope", func(t *testing.T) {
		tmpDir := t.TempDir()
		_, err := resolveNpmPath("@scope", tmpDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "scoped packages require @scope/package format")
	})
}

// TestResolveNpmPath_BoundaryValidation tests that npm: paths are restricted to node_modules
func TestResolveNpmPath_BoundaryValidation(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceRoot := tmpDir

	// Create node_modules
	nodeModulesDir := filepath.Join(tmpDir, "node_modules")
	require.NoError(t, os.MkdirAll(nodeModulesDir, 0o755))

	tests := []struct {
		name        string
		npmPath     string
		shouldError bool
		errContains string
	}{
		{
			name:        "absolute path disguised as package name",
			npmPath:     "../../../../../../../etc/passwd",
			shouldError: true,
			errContains: "path traversal not allowed",
		},
		{
			name:        "package name with multiple dotdot sequences",
			npmPath:     "../../../../../../etc/shadow",
			shouldError: true,
			errContains: "path traversal not allowed",
		},
		{
			name:        "scoped package with dotdot in scope",
			npmPath:     "@../evil/package/file.json",
			shouldError: true,
			errContains: "path traversal not allowed",
		},
		{
			name:        "scoped package with dotdot in package name",
			npmPath:     "@scope/../evil/file.json",
			shouldError: true,
			errContains: "path traversal not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveNpmPath(tt.npmPath, workspaceRoot)

			if tt.shouldError {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Empty(t, got)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
