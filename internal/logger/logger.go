/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

// Package logger provides a configurable logger that can be silenced for LSP/MCP integrations.
package logger

import (
	"io"
	"log"
	"os"
)

var (
	// Default logs to stderr. Set to io.Discard for silent mode (LSP, MCP).
	output io.Writer = os.Stderr
	logger *log.Logger
)

func init() {
	logger = log.New(output, "", 0)
}

// SetOutput configures the logger output destination.
// Use io.Discard to silence all logging.
func SetOutput(w io.Writer) {
	output = w
	logger = log.New(output, "", 0)
}

// Warn logs a warning message.
func Warn(format string, args ...any) {
	logger.Printf("warning: "+format, args...)
}

// Info logs an informational message.
func Info(format string, args ...any) {
	logger.Printf(format, args...)
}

// Debug logs a debug message. Currently same as Info.
func Debug(format string, args ...any) {
	logger.Printf(format, args...)
}
