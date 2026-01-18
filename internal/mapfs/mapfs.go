/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package mapfs provides an in-memory filesystem implementation for testing.
package mapfs

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
	"sync"
	"testing/fstest"
	"time"
)

// MapFileSystem implements FileSystem using an in-memory fstest.MapFS.
// This is useful for testing without touching the real filesystem.
type MapFileSystem struct {
	mu      sync.RWMutex
	mapFS   fstest.MapFS
	tempDir string
	modTime time.Time
}

// New creates a new in-memory filesystem for testing.
func New() *MapFileSystem {
	return &MapFileSystem{
		mapFS:   make(fstest.MapFS),
		tempDir: "/tmp",
		modTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

// AddFile adds a file to the in-memory filesystem.
func (mfs *MapFileSystem) AddFile(p string, content string, mode fs.FileMode) {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()

	p = mfs.cleanPath(p)
	mfs.mapFS[p] = &fstest.MapFile{
		Data:    []byte(content),
		Mode:    mode,
		ModTime: mfs.modTime,
	}
}

// AddDir adds a directory to the in-memory filesystem.
func (mfs *MapFileSystem) AddDir(p string, mode fs.FileMode) {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()

	p = mfs.cleanPath(p)
	keepFile := p + "/.keep"
	mfs.mapFS[keepFile] = &fstest.MapFile{
		Data:    []byte(""),
		Mode:    mode.Perm(),
		ModTime: mfs.modTime,
	}
}

// WriteFile implements FileSystem.
func (mfs *MapFileSystem) WriteFile(name string, data []byte, perm fs.FileMode) error {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()

	name = mfs.cleanPath(name)

	if err := mfs.ensureParentDirLocked(name); err != nil {
		return err
	}

	mfs.mapFS[name] = &fstest.MapFile{
		Data:    append([]byte(nil), data...),
		Mode:    perm,
		ModTime: mfs.modTime,
	}

	return nil
}

// ReadFile implements FileSystem.
func (mfs *MapFileSystem) ReadFile(name string) ([]byte, error) {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()

	return fs.ReadFile(mfs.mapFS, mfs.cleanPath(name))
}

// Remove implements FileSystem.
func (mfs *MapFileSystem) Remove(name string) error {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()

	name = mfs.cleanPath(name)

	if _, exists := mfs.mapFS[name]; !exists {
		return &fs.PathError{Op: "remove", Path: name, Err: fs.ErrNotExist}
	}

	delete(mfs.mapFS, name)
	return nil
}

// MkdirAll implements FileSystem.
func (mfs *MapFileSystem) MkdirAll(p string, perm fs.FileMode) error {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()

	p = mfs.cleanPath(p)
	keepFile := p + "/.keep"

	if file, exists := mfs.mapFS[p]; exists && !file.Mode.IsDir() {
		return &fs.PathError{Op: "mkdir", Path: p, Err: fmt.Errorf("not a directory")}
	}

	mfs.mapFS[keepFile] = &fstest.MapFile{
		Data:    []byte(""),
		Mode:    perm.Perm(),
		ModTime: mfs.modTime,
	}

	return nil
}

// TempDir implements FileSystem.
func (mfs *MapFileSystem) TempDir() string {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()
	return mfs.tempDir
}

// SetTempDir sets the temp directory path.
func (mfs *MapFileSystem) SetTempDir(dir string) {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()
	mfs.tempDir = dir
}

// Stat implements FileSystem.
func (mfs *MapFileSystem) Stat(name string) (fs.FileInfo, error) {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()

	return fs.Stat(mfs.mapFS, mfs.cleanPath(name))
}

// Exists implements FileSystem.
func (mfs *MapFileSystem) Exists(p string) bool {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()

	p = mfs.cleanPath(p)

	if _, exists := mfs.mapFS[p]; exists {
		return true
	}

	prefix := p + "/"
	for filePath := range mfs.mapFS {
		if strings.HasPrefix(filePath, prefix) {
			return true
		}
	}

	return false
}

// ReadDir implements FileSystem.
func (mfs *MapFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()

	return fs.ReadDir(mfs.mapFS, mfs.cleanPath(name))
}

// Open implements FileSystem.
func (mfs *MapFileSystem) Open(name string) (fs.File, error) {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()

	return mfs.mapFS.Open(mfs.cleanPath(name))
}

// ListFiles returns all files in the MapFS for debugging.
func (mfs *MapFileSystem) ListFiles() map[string]string {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()

	result := make(map[string]string)
	for p, file := range mfs.mapFS {
		// Directories are stored as .keep files
		if strings.HasSuffix(p, "/.keep") || p == ".keep" {
			dirPath := path.Dir(p)
			if dirPath == "." {
				dirPath = "/"
			}
			result[dirPath] = "directory"
		} else {
			result[p] = fmt.Sprintf("file (%d bytes)", len(file.Data))
		}
	}
	return result
}

func (mfs *MapFileSystem) cleanPath(p string) string {
	cleaned := path.Clean(p)
	if !path.IsAbs(cleaned) {
		cleaned = "/" + cleaned
	}
	return strings.TrimPrefix(cleaned, "/")
}

func (mfs *MapFileSystem) ensureParentDirLocked(filePath string) error {
	dir := path.Dir(filePath)
	if dir == "." || dir == "/" || dir == "" {
		return nil
	}

	if file, exists := mfs.mapFS[dir]; exists && !file.Mode.IsDir() {
		return &fs.PathError{Op: "open", Path: filePath, Err: fmt.Errorf("not a directory")}
	}

	return nil
}
