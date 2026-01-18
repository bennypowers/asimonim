/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package fs provides filesystem abstractions for asimonim.
package fs

import (
	"io/fs"
	"os"
)

// FileSystem provides an abstraction over filesystem operations.
// This interface is congruent with bennypowers.dev/cem/internal/platform.FileSystem
// and bennypowers.dev/mappa/fs.FileSystem to enable duck typing compatibility.
type FileSystem interface {
	// File operations
	WriteFile(name string, data []byte, perm fs.FileMode) error
	ReadFile(name string) ([]byte, error)
	Remove(name string) error

	// Directory operations
	MkdirAll(path string, perm fs.FileMode) error
	ReadDir(name string) ([]fs.DirEntry, error)
	TempDir() string

	// File system queries
	Stat(name string) (fs.FileInfo, error)
	Exists(path string) bool

	// fs.FS compatibility - allows use with fs.WalkDir
	Open(name string) (fs.File, error)
}

// OSFileSystem implements FileSystem using the standard os package.
type OSFileSystem struct{}

// NewOSFileSystem creates a new filesystem that uses the standard os package.
func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

// WriteFile writes data to a file with the given permissions.
func (f *OSFileSystem) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}

// ReadFile reads the entire contents of a file.
func (f *OSFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// Remove deletes the named file or empty directory.
func (f *OSFileSystem) Remove(name string) error {
	return os.Remove(name)
}

// MkdirAll creates a directory path and all parents that do not exist.
func (f *OSFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

// TempDir returns the default directory for temporary files.
func (f *OSFileSystem) TempDir() string {
	return os.TempDir()
}

// Stat returns file information for the named file.
func (f *OSFileSystem) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// Exists returns true if the path exists.
func (f *OSFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ReadDir reads the named directory and returns its entries.
func (f *OSFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

// Open opens the named file for reading.
func (f *OSFileSystem) Open(name string) (fs.File, error) {
	return os.Open(name)
}
