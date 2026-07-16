package workspace

import (
	"testing"

	"bennypowers.dev/asimonim/lsp/testutil"
	"github.com/stretchr/testify/assert"
)

func TestNotifyDiagnosticRefresh_ViaServerContext(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	assert.False(t, ctx.NotifyDiagnosticRefreshCalled)

	ctx.NotifyDiagnosticRefresh()
	assert.True(t, ctx.NotifyDiagnosticRefreshCalled, "NotifyDiagnosticRefresh should set the flag")
}
