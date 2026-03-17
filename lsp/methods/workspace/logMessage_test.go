package workspace

import (
	"testing"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestLogError_NilContext(t *testing.T) {
	// Should not panic with nil context
	LogError(nil, "test error: %s", "message")
	// If we get here, it didn't panic - success!
}

func TestLogWarning_NilContext(t *testing.T) {
	// Should not panic with nil context
	LogWarning(nil, "test warning: %s", "message")
}

func TestShowMessage_NilContext(t *testing.T) {
	// Should not panic with nil context
	ShowMessage(nil, 1, "test message")
}

func TestLogError_WithContext(t *testing.T) {
	// We can't easily test with a real context without a full LSP server
	// but we can verify it doesn't panic
	// In production, this would send messages via the LSP connection
	var ctx *glsp.Context // nil, but typed
	LogError(ctx, "test error: %s", "message")
}

func TestLogWarning_WithContext(t *testing.T) {
	var ctx *glsp.Context
	LogWarning(ctx, "test warning: %d items", 42)
}

func TestShowMessage_WithNilContext(t *testing.T) {
	// Should not panic with nil context and various message types
	ShowMessage(nil, protocol.MessageTypeError, "error message")
	ShowMessage(nil, protocol.MessageTypeWarning, "warning message")
	ShowMessage(nil, protocol.MessageTypeInfo, "info message")
}

func TestLogError_FormatsMessage(t *testing.T) {
	// Verify format string works with multiple args
	LogError(nil, "error %s: code %d", "test", 42)
}

func TestLogWarning_FormatsMessage(t *testing.T) {
	LogWarning(nil, "warning %s: count %d", "test", 7)
}

func TestLogError_NoArgs(t *testing.T) {
	// Format string with no args should not panic
	LogError(nil, "plain error message")
}

func TestLogWarning_NoArgs(t *testing.T) {
	LogWarning(nil, "plain warning message")
}

func TestShowMessage_AllTypes(t *testing.T) {
	// Exercise all message types with nil context
	types := []protocol.MessageType{
		protocol.MessageTypeError,
		protocol.MessageTypeWarning,
		protocol.MessageTypeInfo,
		protocol.MessageTypeLog,
	}
	for _, mt := range types {
		ShowMessage(nil, mt, "test message")
	}
}

func TestLogError_MultipleFormatArgs(t *testing.T) {
	LogError(nil, "error %s in %s at line %d: %v", "parse", "file.json", 42, true)
}

func TestLogWarning_MultipleFormatArgs(t *testing.T) {
	LogWarning(nil, "warning %s in %s at line %d: %v", "deprecation", "tokens.json", 10, false)
}

// noopNotify is a no-op Notify function for testing the non-nil context path
func noopNotify(_ string, _ any) {}

func TestLogError_WithNonNilContext(t *testing.T) {
	// Non-nil context with no-op Notify exercises the goroutine branch
	ctx := &glsp.Context{Notify: noopNotify}
	LogError(ctx, "error %s", "test")
}

func TestLogWarning_WithNonNilContext(t *testing.T) {
	ctx := &glsp.Context{Notify: noopNotify}
	LogWarning(ctx, "warning %s", "test")
}

func TestShowMessage_WithNonNilContext(t *testing.T) {
	ctx := &glsp.Context{Notify: noopNotify}
	ShowMessage(ctx, protocol.MessageTypeInfo, "test message")
}
