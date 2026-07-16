package workspace

import (
	"fmt"

	"bennypowers.dev/asimonim/lsp/internal/log"
)

// LogError logs an error message to stderr.
// The handler.go wrap functions handle error reporting directly.
func LogError(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	log.Error("%s", message)
}

// LogWarning logs a warning message to stderr.
// The handler.go wrap functions handle warning reporting directly.
func LogWarning(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	log.Warn("%s", message)
}
