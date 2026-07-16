package lifecycle

import (
	"context"
	"errors"
	"testing"

	"bennypowers.dev/asimonim/lsp/testutil"
	"bennypowers.dev/asimonim/lsp/types"
	"github.com/stretchr/testify/assert"
	protocol "github.com/bennypowers/glsp/protocol_3_17"
)

func TestInitialized(t *testing.T) {
	t.Run("stores GLSP context", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		bgCtx := context.Background()
		req := types.NewRequestContext(ctx, bgCtx)

		params := &protocol.InitializedParams{}

		err := Initialized(req, params)
		assert.NoError(t, err)

		// Verify context was stored
		assert.Equal(t, bgCtx, ctx.GLSPContext())
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
