/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package parser

import (
	"testing"
)

func TestIsLikelyJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{
			name:     "starts with brace",
			input:    []byte(`{"key": "value"}`),
			expected: true,
		},
		{
			name:     "brace with leading whitespace",
			input:    []byte("  \t\n{\"key\": \"value\"}"),
			expected: true,
		},
		{
			name:     "brace with UTF-8 BOM",
			input:    []byte{0xEF, 0xBB, 0xBF, '{'},
			expected: true,
		},
		{
			name:     "YAML content",
			input:    []byte("color:\n  $value: '#fff'\n"),
			expected: false,
		},
		{
			name:     "starts with letter",
			input:    []byte("key: value"),
			expected: false,
		},
		{
			name:     "starts with dash (YAML list)",
			input:    []byte("- item1\n- item2"),
			expected: false,
		},
		{
			name:     "empty input",
			input:    []byte{},
			expected: false,
		},
		{
			name:     "only whitespace",
			input:    []byte("   \t\n  "),
			expected: false,
		},
		{
			name:     "starts with array bracket",
			input:    []byte(`["a", "b"]`),
			expected: false,
		},
		{
			name:     "starts with hash (YAML comment)",
			input:    []byte("# comment\nkey: value"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLikelyJSON(tt.input)
			if got != tt.expected {
				t.Errorf("isLikelyJSON(%q) = %v, want %v", string(tt.input), got, tt.expected)
			}
		})
	}
}

func TestNormalizeMap(t *testing.T) {
	t.Run("map[string]any passes through", func(t *testing.T) {
		input := map[string]any{
			"color": "#fff",
			"size":  42,
		}
		result := normalizeMap(input)
		m, ok := result.(map[string]any)
		if !ok {
			t.Fatal("expected map[string]any")
		}
		if m["color"] != "#fff" {
			t.Errorf("expected color=#fff, got %v", m["color"])
		}
		if m["size"] != 42 {
			t.Errorf("expected size=42, got %v", m["size"])
		}
	})

	t.Run("map[any]any converts to map[string]any", func(t *testing.T) {
		// YAML with numeric keys produces map[any]any
		input := map[any]any{
			10:      "ten",
			"hello": "world",
		}
		result := normalizeMap(input)
		m, ok := result.(map[string]any)
		if !ok {
			t.Fatal("expected map[string]any")
		}
		if m["10"] != "ten" {
			t.Errorf("expected 10=ten, got %v", m["10"])
		}
		if m["hello"] != "world" {
			t.Errorf("expected hello=world, got %v", m["hello"])
		}
	})

	t.Run("nested map[any]any converts recursively", func(t *testing.T) {
		input := map[any]any{
			"outer": map[any]any{
				100: "hundred",
			},
		}
		result := normalizeMap(input)
		m, ok := result.(map[string]any)
		if !ok {
			t.Fatal("expected map[string]any")
		}
		inner, ok := m["outer"].(map[string]any)
		if !ok {
			t.Fatal("expected inner map[string]any")
		}
		if inner["100"] != "hundred" {
			t.Errorf("expected 100=hundred, got %v", inner["100"])
		}
	})

	t.Run("slices are recursively normalized", func(t *testing.T) {
		input := []any{
			map[any]any{
				1: "one",
			},
			"plain string",
		}
		result := normalizeMap(input)
		arr, ok := result.([]any)
		if !ok {
			t.Fatal("expected []any")
		}
		if len(arr) != 2 {
			t.Fatalf("expected 2 elements, got %d", len(arr))
		}
		m, ok := arr[0].(map[string]any)
		if !ok {
			t.Fatal("expected first element to be map[string]any")
		}
		if m["1"] != "one" {
			t.Errorf("expected 1=one, got %v", m["1"])
		}
		if arr[1] != "plain string" {
			t.Errorf("expected 'plain string', got %v", arr[1])
		}
	})

	t.Run("primitive values pass through", func(t *testing.T) {
		if normalizeMap("hello") != "hello" {
			t.Error("string should pass through")
		}
		if normalizeMap(42) != 42 {
			t.Error("int should pass through")
		}
		if normalizeMap(3.14) != 3.14 {
			t.Error("float should pass through")
		}
		if normalizeMap(true) != true {
			t.Error("bool should pass through")
		}
		if normalizeMap(nil) != nil {
			t.Error("nil should pass through")
		}
	})

	t.Run("nested map[string]any with child map[any]any", func(t *testing.T) {
		input := map[string]any{
			"tokens": map[any]any{
				"color": map[any]any{
					"$value": "#fff",
				},
			},
		}
		result := normalizeMap(input)
		m := result.(map[string]any)
		tokens := m["tokens"].(map[string]any)
		color := tokens["color"].(map[string]any)
		if color["$value"] != "#fff" {
			t.Errorf("expected $value=#fff, got %v", color["$value"])
		}
	})
}
