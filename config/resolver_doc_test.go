/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package config

import (
	"testing"

	"bennypowers.dev/asimonim/internal/mapfs"
	"bennypowers.dev/asimonim/specifier"
	"bennypowers.dev/asimonim/testutil"
)

func TestExtractResolverSourcePaths(t *testing.T) {
	t.Run("extracts inline sources", func(t *testing.T) {
		mfs := mapfs.New()
		mfs.AddFile("/project/resolver.json", `{
			"version": "2025.10",
			"resolutionOrder": [{
				"type": "set",
				"name": "base",
				"sources": [
					{"$ref": "./palette.json"},
					{"$ref": "./colors.json"}
				]
			}]
		}`, 0644)

		paths, err := extractResolverSourcePaths(mfs, "/project/resolver.json")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(paths) != 2 {
			t.Fatalf("expected 2 paths, got %d", len(paths))
		}
		if paths[0] != "/project/palette.json" {
			t.Errorf("expected /project/palette.json, got %s", paths[0])
		}
		if paths[1] != "/project/colors.json" {
			t.Errorf("expected /project/colors.json, got %s", paths[1])
		}
	})

	t.Run("extracts named set references", func(t *testing.T) {
		mfs := mapfs.New()
		mfs.AddFile("/project/resolver.json", `{
			"version": "2025.10",
			"sets": {
				"base": {
					"sources": [{"$ref": "./palette.json"}]
				}
			},
			"resolutionOrder": [
				{"$ref": "#/sets/base"}
			]
		}`, 0644)

		paths, err := extractResolverSourcePaths(mfs, "/project/resolver.json")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(paths) != 1 {
			t.Fatalf("expected 1 path, got %d", len(paths))
		}
		if paths[0] != "/project/palette.json" {
			t.Errorf("expected /project/palette.json, got %s", paths[0])
		}
	})

	t.Run("deduplicates paths", func(t *testing.T) {
		mfs := mapfs.New()
		mfs.AddFile("/project/resolver.json", `{
			"version": "2025.10",
			"resolutionOrder": [
				{"sources": [{"$ref": "./palette.json"}]},
				{"sources": [{"$ref": "./palette.json"}]}
			]
		}`, 0644)

		paths, err := extractResolverSourcePaths(mfs, "/project/resolver.json")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(paths) != 1 {
			t.Fatalf("expected 1 path (deduped), got %d", len(paths))
		}
	})

	t.Run("ignores JSON pointer refs in sources", func(t *testing.T) {
		mfs := mapfs.New()
		mfs.AddFile("/project/resolver.json", `{
			"version": "2025.10",
			"resolutionOrder": [{
				"sources": [
					{"$ref": "./palette.json"},
					{"$ref": "#/some/internal"}
				]
			}]
		}`, 0644)

		paths, err := extractResolverSourcePaths(mfs, "/project/resolver.json")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(paths) != 1 {
			t.Fatalf("expected 1 path (no pointer refs), got %d", len(paths))
		}
	})
}

func TestResolveResolverSources(t *testing.T) {
	t.Run("loads source files from resolver document in config", func(t *testing.T) {
		mfs := testutil.NewFixtureFS(t, "fixtures/config/resolver-sources", "/project")
		specResolver, err := specifier.NewDefaultResolver(mfs, "/project")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg, err := Load(mfs, "/project")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg == nil {
			t.Fatal("expected config, got nil")
		}

		if len(cfg.Resolvers) != 1 {
			t.Fatalf("expected 1 resolver, got %d", len(cfg.Resolvers))
		}

		sources, err := cfg.ResolveResolverSources(specResolver, mfs, "/project")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(sources) != 2 {
			t.Fatalf("expected 2 source files, got %d", len(sources))
		}

		// Check that both source files are returned
		foundPalette := false
		foundColors := false
		for _, s := range sources {
			if s.Path == "/project/src/design-tokens/palette.json" {
				foundPalette = true
			}
			if s.Path == "/project/src/design-tokens/colors.json" {
				foundColors = true
			}
		}
		if !foundPalette {
			t.Error("expected to find palette.json in resolved sources")
		}
		if !foundColors {
			t.Error("expected to find colors.json in resolved sources")
		}
	})
}
