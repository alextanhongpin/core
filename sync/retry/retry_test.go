package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
	"github.com/go-openapi/testify/assert"
)

func TestExec(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		err := retry.Exec(t.Context(), func(context.Context) error {
			return nil
		})
		assert.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		var count int
		err := retry.Exec(t.Context(), func(context.Context) error {
			count++
			return assert.AnError
		}, retry.NoWait, retry.N(5))

		is := assert.New(t)
		is.ErrorIs(err, assert.AnError)
		is.ErrorIs(err, retry.ErrLimitExceeded, "did not complete within 5 attempts")
		is.Equal(6, count, "initial plus 5 retries")
	})

	t.Run("context timeout", func(t *testing.T) {
		var timeoutErr = errors.New("timeout")
		ctx, cancel := context.WithTimeoutCause(t.Context(), time.Millisecond, timeoutErr)
		defer cancel()

		err := retry.Exec(ctx, func(context.Context) error {
			return assert.AnError
		}, retry.Constant(time.Millisecond))

		is := assert.New(t)
		is.ErrorIs(err, assert.AnError)
		is.ErrorIs(err, timeoutErr, "context timeout")
	})

	t.Run("zero times", func(t *testing.T) {
		var count int
		err := retry.Exec(t.Context(), func(context.Context) error {
			count++
			return assert.AnError
		}, retry.N(0))
		is := assert.New(t)
		is.ErrorIs(err, assert.AnError)
		is.ErrorIs(err, retry.ErrLimitExceeded)
		is.Equal(count, 1)
	})
}

func TestDo(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		v, err := retry.Do(t.Context(), func(context.Context) (string, error) {
			return t.Name(), nil
		})
		is := assert.New(t)
		is.NoError(err)
		is.Equal(t.Name(), v)
	})

	t.Run("error", func(t *testing.T) {

		var count int
		v, err := retry.Do(t.Context(), func(ctx context.Context) (string, error) {
			count++
			return "", assert.AnError
		}, retry.NoWait, retry.N(5))

		is := assert.New(t)
		is.ErrorIs(err, assert.AnError)
		is.ErrorIs(err, retry.ErrLimitExceeded, "did not complete within 5 attempts")
		is.Equal(6, count, "initial plus 5 retries")
		is.Empty(v)
	})

	t.Run("context timeout", func(t *testing.T) {
		var timeoutErr = errors.New("timeout")
		ctx, cancel := context.WithTimeoutCause(t.Context(), time.Millisecond, timeoutErr)
		defer cancel()

		v, err := retry.Do(ctx, func(context.Context) (string, error) {
			return "", assert.AnError
		}, retry.Constant(time.Millisecond))

		is := assert.New(t)
		is.ErrorIs(err, assert.AnError)
		is.ErrorIs(err, timeoutErr, "context timeout")
		is.Empty(v)
	})

	t.Run("zero times", func(t *testing.T) {
		var count int
		v, err := retry.Do(t.Context(), func(context.Context) (string, error) {
			count++
			return "", assert.AnError
		}, retry.N(0))

		is := assert.New(t)
		is.ErrorIs(err, assert.AnError)
		is.ErrorIs(err, retry.ErrLimitExceeded)
		is.Equal(count, 1)
		is.Empty(v)
	})
}
