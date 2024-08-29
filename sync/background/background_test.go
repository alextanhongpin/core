package background_test

import (
	"context"
	"testing"

	"github.com/alextanhongpin/core/sync/background"
	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

func TestBackground(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		is := assert.New(t)
		bg, stop := background.New(ctx, func(ctx context.Context, n int) {
			is.Equal(42, n)
		}, nil)
		defer stop()

		is.Nil(bg.Send(42))
	})

	t.Run("early stop", func(t *testing.T) {
		is := assert.New(t)
		bg, stop := background.New(ctx, func(ctx context.Context, n int) {
			is.Equal(42, n)
		}, nil)
		stop()

		is.ErrorIs(bg.Send(1), background.ErrTerminated)
	})
}
