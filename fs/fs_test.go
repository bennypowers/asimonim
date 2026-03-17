/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package fs_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"bennypowers.dev/asimonim/fs"
)

func TestNewOSFileSystem(t *testing.T) {
	osfs := fs.NewOSFileSystem()
	if osfs == nil {
		t.Fatal("NewOSFileSystem returned nil")
	}
}

func TestOSFileSystem_WriteAndReadFile(t *testing.T) {
	osfs := fs.NewOSFileSystem()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	content := []byte("hello world")
	if err := osfs.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	got, err := osfs.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if string(got) != "hello world" {
		t.Errorf("ReadFile = %q, want %q", got, "hello world")
	}
}

func TestOSFileSystem_MkdirAll(t *testing.T) {
	osfs := fs.NewOSFileSystem()
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")

	if err := osfs.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}

	info, err := os.Stat(nested)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestOSFileSystem_Stat(t *testing.T) {
	osfs := fs.NewOSFileSystem()
	dir := t.TempDir()
	path := filepath.Join(dir, "stat-test.txt")
	if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
		t.Fatalf("setup WriteFile error: %v", err)
	}

	info, err := osfs.Stat(path)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if info.Name() != "stat-test.txt" {
		t.Errorf("Name = %q, want %q", info.Name(), "stat-test.txt")
	}
}

func TestOSFileSystem_Exists(t *testing.T) {
	osfs := fs.NewOSFileSystem()
	dir := t.TempDir()
	path := filepath.Join(dir, "exists-test.txt")

	if osfs.Exists(path) {
		t.Error("Exists returned true for nonexistent file")
	}

	if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
		t.Fatalf("setup WriteFile error: %v", err)
	}

	if !osfs.Exists(path) {
		t.Error("Exists returned false for existing file")
	}
}

func TestOSFileSystem_ReadDir(t *testing.T) {
	osfs := fs.NewOSFileSystem()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644); err != nil {
		t.Fatalf("setup WriteFile error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644); err != nil {
		t.Fatalf("setup WriteFile error: %v", err)
	}

	entries, err := osfs.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	names := []string{entries[0].Name(), entries[1].Name()}
	sort.Strings(names)
	if names[0] != "a.txt" || names[1] != "b.txt" {
		t.Errorf("expected [a.txt, b.txt], got %v", names)
	}
}

func TestOSFileSystem_Remove(t *testing.T) {
	osfs := fs.NewOSFileSystem()
	dir := t.TempDir()
	path := filepath.Join(dir, "remove-test.txt")
	if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
		t.Fatalf("setup WriteFile error: %v", err)
	}

	if err := osfs.Remove(path); err != nil {
		t.Fatalf("Remove error: %v", err)
	}

	if osfs.Exists(path) {
		t.Error("file still exists after Remove")
	}
}

func TestOSFileSystem_TempDir(t *testing.T) {
	osfs := fs.NewOSFileSystem()
	td := osfs.TempDir()
	if td == "" {
		t.Fatal("TempDir returned empty string")
	}
	info, err := os.Stat(td)
	if err != nil {
		t.Fatalf("TempDir path does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("TempDir path is not a directory: %s", td)
	}
}

func TestOSFileSystem_Open(t *testing.T) {
	osfs := fs.NewOSFileSystem()
	dir := t.TempDir()
	path := filepath.Join(dir, "open-test.txt")
	if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
		t.Fatalf("setup WriteFile error: %v", err)
	}

	f, err := osfs.Open(path)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	defer f.Close()

	buf := make([]byte, 12)
	n, err := f.Read(buf)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if string(buf[:n]) != "test content" {
		t.Errorf("Read = %q, want %q", string(buf[:n]), "test content")
	}
}
