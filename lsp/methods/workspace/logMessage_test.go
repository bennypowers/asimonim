package workspace

import (
	"testing"
)

func TestLogError_NoArgs(t *testing.T) {
	// Format string with no args should not panic
	LogError("plain error message")
}

func TestLogWarning_NoArgs(t *testing.T) {
	LogWarning("plain warning message")
}

func TestLogError_FormatsMessage(t *testing.T) {
	// Verify format string works with multiple args
	LogError("error %s: code %d", "test", 42)
}

func TestLogWarning_FormatsMessage(t *testing.T) {
	LogWarning("warning %s: count %d", "test", 7)
}

func TestLogError_MultipleFormatArgs(t *testing.T) {
	LogError("error %s in %s at line %d: %v", "parse", "file.json", 42, true)
}

func TestLogWarning_MultipleFormatArgs(t *testing.T) {
	LogWarning("warning %s in %s at line %d: %v", "deprecation", "tokens.json", 10, false)
}
