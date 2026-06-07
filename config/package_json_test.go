/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package config

import (
	"testing"

	"bennypowers.dev/asimonim/internal/mapfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromPackageJSON_AsimonimKey(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"name": "my-project",
		"asimonim": {
			"prefix": "ds",
			"tokensFiles": ["./tokens.json"],
			"groupMarkers": ["_", "@"]
		}
	}`, 0644)

	cfg, err := LoadFromPackageJSON(mfs, "/project")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "ds", cfg.Prefix)
	require.Len(t, cfg.Files, 1)
	assert.Equal(t, "./tokens.json", cfg.Files[0].Path)
	assert.Equal(t, []string{"_", "@"}, cfg.GroupMarkers)
}

func TestLoadFromPackageJSON_DesignTokensLanguageServerKey(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"name": "my-project",
		"designTokensLanguageServer": {
			"prefix": "rh",
			"tokensFiles": ["npm:@rhds/tokens/json/rhds.tokens.json"]
		}
	}`, 0644)

	cfg, err := LoadFromPackageJSON(mfs, "/project")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "rh", cfg.Prefix)
	require.Len(t, cfg.Files, 1)
	assert.Equal(t, "npm:@rhds/tokens/json/rhds.tokens.json", cfg.Files[0].Path)
}

func TestLoadFromPackageJSON_DashKey(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"name": "my-project",
		"design-tokens-language-server": {
			"prefix": "pf",
			"tokensFiles": ["./pf-tokens.json"]
		}
	}`, 0644)

	cfg, err := LoadFromPackageJSON(mfs, "/project")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "pf", cfg.Prefix)
	require.Len(t, cfg.Files, 1)
	assert.Equal(t, "./pf-tokens.json", cfg.Files[0].Path)
}

func TestLoadFromPackageJSON_AsimonimKeyTakesPrecedence(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"name": "my-project",
		"asimonim": {
			"prefix": "preferred"
		},
		"designTokensLanguageServer": {
			"prefix": "legacy"
		}
	}`, 0644)

	cfg, err := LoadFromPackageJSON(mfs, "/project")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "preferred", cfg.Prefix)
}

func TestLoadFromPackageJSON_TokensFilesAsString(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"asimonim": {
			"tokensFiles": "./single-file.json"
		}
	}`, 0644)

	cfg, err := LoadFromPackageJSON(mfs, "/project")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Len(t, cfg.Files, 1)
	assert.Equal(t, "./single-file.json", cfg.Files[0].Path)
}

func TestLoadFromPackageJSON_TokensFilesArray(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"asimonim": {
			"tokensFiles": ["./a.json", "npm:@scope/pkg/tokens.json", "./b.json"]
		}
	}`, 0644)

	cfg, err := LoadFromPackageJSON(mfs, "/project")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Len(t, cfg.Files, 3)
	assert.Equal(t, "./a.json", cfg.Files[0].Path)
	assert.Equal(t, "npm:@scope/pkg/tokens.json", cfg.Files[1].Path)
	assert.Equal(t, "./b.json", cfg.Files[2].Path)
}

func TestLoadFromPackageJSON_Resolvers(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"asimonim": {
			"resolvers": ["./tokens.resolver.json", "npm:@acme/tokens/resolver.json"]
		}
	}`, 0644)

	cfg, err := LoadFromPackageJSON(mfs, "/project")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Len(t, cfg.Resolvers, 2)
	assert.Equal(t, "./tokens.resolver.json", cfg.Resolvers[0])
	assert.Equal(t, "npm:@acme/tokens/resolver.json", cfg.Resolvers[1])
}

func TestLoadFromPackageJSON_CDN(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"asimonim": {
			"cdn": "jsdelivr"
		}
	}`, 0644)

	cfg, err := LoadFromPackageJSON(mfs, "/project")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "jsdelivr", cfg.CDN)
}

func TestLoadFromPackageJSON_NoPackageJSON(t *testing.T) {
	mfs := mapfs.New()

	cfg, err := LoadFromPackageJSON(mfs, "/project")
	require.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestLoadFromPackageJSON_NoConfigKey(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"name": "my-project",
		"version": "1.0.0"
	}`, 0644)

	cfg, err := LoadFromPackageJSON(mfs, "/project")
	require.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestLoadFromPackageJSON_EmptyRootDir(t *testing.T) {
	mfs := mapfs.New()

	cfg, err := LoadFromPackageJSON(mfs, "")
	require.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestLoadFromPackageJSON_InvalidJSON(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{not valid`, 0644)

	_, err := LoadFromPackageJSON(mfs, "/project")
	assert.Error(t, err)
}

func TestLoadFromPackageJSON_ConfigNotObject(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"asimonim": "not-an-object"
	}`, 0644)

	_, err := LoadFromPackageJSON(mfs, "/project")
	assert.Error(t, err)
}

func TestLoadFromPackageJSON_FilesFallback(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"asimonim": {
			"files": ["./fallback-tokens.json"]
		}
	}`, 0644)

	cfg, err := LoadFromPackageJSON(mfs, "/project")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Len(t, cfg.Files, 1)
	assert.Equal(t, "./fallback-tokens.json", cfg.Files[0].Path)
}

func TestLoadFromPackageJSON_TokensFilesTakesPrecedenceOverFiles(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"asimonim": {
			"tokensFiles": ["./preferred.json"],
			"files": ["./fallback.json"]
		}
	}`, 0644)

	cfg, err := LoadFromPackageJSON(mfs, "/project")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Len(t, cfg.Files, 1)
	assert.Equal(t, "./preferred.json", cfg.Files[0].Path)
}

func TestLoadFromPackageJSON_TokensFilesEmptyArray(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"asimonim": {
			"tokensFiles": []
		}
	}`, 0644)

	cfg, err := LoadFromPackageJSON(mfs, "/project")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Empty array should be non-nil (explicit "no files"), not nil (unset)
	assert.NotNil(t, cfg.Files)
	assert.Empty(t, cfg.Files)
}

func TestLoadFromPackageJSON_TokensFilesMixedArray(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"asimonim": {
			"tokensFiles": [
				"./simple.json",
				{"path": "./with-prefix.json", "prefix": "custom"}
			]
		}
	}`, 0644)

	cfg, err := LoadFromPackageJSON(mfs, "/project")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Len(t, cfg.Files, 2)
	assert.Equal(t, "./simple.json", cfg.Files[0].Path)
	assert.Equal(t, "", cfg.Files[0].Prefix)
	assert.Equal(t, "./with-prefix.json", cfg.Files[1].Path)
	assert.Equal(t, "custom", cfg.Files[1].Prefix)
}

func TestLoadFromPackageJSON_TokensFilesInvalidEntry(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"asimonim": {
			"tokensFiles": ["valid.json", 42]
		}
	}`, 0644)

	_, err := LoadFromPackageJSON(mfs, "/project")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid tokensFiles entry")
}

func TestLoadFromPackageJSON_Schema(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"asimonim": {
			"schema": "draft"
		}
	}`, 0644)

	cfg, err := LoadFromPackageJSON(mfs, "/project")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "draft", cfg.Schema)
}

func TestLoadFromPackageJSON_TokensFilesEmptyStringPath(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"asimonim": {
			"tokensFiles": ""
		}
	}`, 0644)

	_, err := LoadFromPackageJSON(mfs, "/project")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing path")
}

func TestLoadFromPackageJSON_TokensFilesObjectMissingPath(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"asimonim": {
			"tokensFiles": [{"prefix": "custom"}]
		}
	}`, 0644)

	_, err := LoadFromPackageJSON(mfs, "/project")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing path")
}

func TestLoadFromPackageJSON_PrefixWrongType(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"asimonim": {
			"prefix": 42
		}
	}`, 0644)

	_, err := LoadFromPackageJSON(mfs, "/project")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prefix")
}

func TestLoadFromPackageJSON_CDNWrongType(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"asimonim": {
			"cdn": true
		}
	}`, 0644)

	_, err := LoadFromPackageJSON(mfs, "/project")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cdn")
}

func TestLoadFromPackageJSON_SchemaWrongType(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		"asimonim": {
			"schema": ["draft"]
		}
	}`, 0644)

	_, err := LoadFromPackageJSON(mfs, "/project")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schema")
}

func TestLoadFromPackageJSON_JSONC(t *testing.T) {
	mfs := mapfs.New()
	mfs.AddFile("/project/package.json", `{
		// comment
		"asimonim": {
			"prefix": "ds"
		}
	}`, 0644)

	cfg, err := LoadFromPackageJSON(mfs, "/project")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "ds", cfg.Prefix)
}
