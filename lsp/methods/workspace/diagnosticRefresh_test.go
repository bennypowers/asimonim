package workspace

import (
	"testing"

	"github.com/bennypowers/glsp"
	protocol "github.com/bennypowers/glsp/protocol_3_17"
	"github.com/stretchr/testify/assert"
)

func TestNotifyDiagnosticRefresh_NilContext(t *testing.T) {
	// Should not panic
	NotifyDiagnosticRefresh(nil)
}

func TestNotifyDiagnosticRefresh_NilNotify(t *testing.T) {
	// Context with nil Notify should not panic
	NotifyDiagnosticRefresh(&glsp.Context{})
}

func TestNotifyDiagnosticRefresh_SendsNotification(t *testing.T) {
	var notifiedMethod string
	ctx := &glsp.Context{
		Notify: func(method string, params any) {
			notifiedMethod = method
		},
	}

	NotifyDiagnosticRefresh(ctx)
	assert.Equal(t, protocol.MethodWorkspaceDiagnosticRefresh, notifiedMethod)
}
