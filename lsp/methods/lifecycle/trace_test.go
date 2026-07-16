package lifecycle

import (
	"context"
	"testing"

	"bennypowers.dev/asimonim/lsp/testutil"
	"bennypowers.dev/asimonim/lsp/types"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
)

func TestSetTrace(t *testing.T) {
	t.Run("handles off trace level", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		req := types.NewRequestContext(ctx, context.Background())

		params := &protocol.SetTraceParams{
			Value: "off",
		}

		err := SetTrace(req, params)
		assert.NoError(t, err)
	})

	t.Run("handles messages trace level", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		req := types.NewRequestContext(ctx, context.Background())

		params := &protocol.SetTraceParams{
			Value: "messages",
		}

		err := SetTrace(req, params)
		assert.NoError(t, err)
	})

	t.Run("handles verbose trace level", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		req := types.NewRequestContext(ctx, context.Background())

		params := &protocol.SetTraceParams{
			Value: "verbose",
		}

		err := SetTrace(req, params)
		assert.NoError(t, err)
	})

	t.Run("handles invalid trace level gracefully", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		req := types.NewRequestContext(ctx, context.Background())

		params := &protocol.SetTraceParams{
			Value: "invalid",
		}

		// Should not error, just log
		err := SetTrace(req, params)
		assert.NoError(t, err)
	})
}
