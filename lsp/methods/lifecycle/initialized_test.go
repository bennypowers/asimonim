package lifecycle

import (
	"context"
	"errors"
	"testing"

	"bennypowers.dev/asimonim/lsp/testutil"
	"bennypowers.dev/asimonim/lsp/types"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
)

func TestInitialized(t *testing.T) {
	t.Run("does not overwrite server context with request context", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		serverCtx := context.Background()
		ctx.SetServerCtx(serverCtx)

		reqCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		req := types.NewRequestContext(ctx, reqCtx)

		err := Initialized(req, &protocol.InitializedParams{})
		assert.NoError(t, err)

		// Cancel the request context — server context must survive
		cancel()

		// Server context should still be the original, not the canceled request context
		assert.Equal(t, serverCtx, ctx.ServerCtx())
	})

	t.Run("calls LoadTokensFromConfig", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		req := types.NewRequestContext(ctx, context.Background())
		params := &protocol.InitializedParams{}

		err := Initialized(req, params)
		assert.NoError(t, err)
		assert.True(t, ctx.LoadTokensCalled, "LoadTokensFromConfig should be called")
	})

	t.Run("calls RegisterFileWatchers", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		req := types.NewRequestContext(ctx, context.Background())
		params := &protocol.InitializedParams{}

		err := Initialized(req, params)
		assert.NoError(t, err)
		assert.True(t, ctx.RegisterWatchersCalled, "RegisterFileWatchers should be called")
	})

	t.Run("continues on LoadTokensFromConfig error", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		ctx.LoadTokensFunc = func() error {
			return errors.New("load error")
		}

		req := types.NewRequestContext(ctx, context.Background())
		params := &protocol.InitializedParams{}

		// Should not fail, just log warning
		err := Initialized(req, params)
		assert.NoError(t, err)
		assert.True(t, ctx.LoadTokensCalled)
	})

	t.Run("continues on RegisterFileWatchers error", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		ctx.RegisterWatchersFunc = func(context.Context) error {
			return errors.New("watcher error")
		}

		req := types.NewRequestContext(ctx, context.Background())
		params := &protocol.InitializedParams{}

		// Should not fail, just log warning
		err := Initialized(req, params)
		assert.NoError(t, err)
		assert.True(t, ctx.RegisterWatchersCalled)
	})
}
