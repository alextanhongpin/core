package idempotent_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/alextanhongpin/core/dsync/idempotent"
	"github.com/alextanhongpin/dbtx/testing/redistest"
)

// Regression: requests were compared after decoding the stored JSON into K.
// With K=any, a large int64 decoded as float64 lost precision and every
// replay failed with ErrRequestMismatch.
func TestInterfaceRequestLargeIntegerIsNotAMismatch(t *testing.T) {
	idp := idempotent.MustNewWithRedis(redistest.Client(t)).MustHandlerFunc(
		func(ctx context.Context, req any) (string, error) { return "ok", nil },
		idempotent.HandlerConfig{},
	)
	req := map[string]any{"id": int64(1<<53 + 1)}

	_, shared, err := idp.Do(ctx, t.Name(), req)
	assert.NoError(t, err)
	assert.False(t, shared)

	res, shared, err := idp.Do(ctx, t.Name(), req)
	assert.NoError(t, err)
	assert.True(t, shared)
	assert.Equal(t, "ok", res)
}

// Regression: when the caller canceled while the task kept running (and then
// succeeded), lease renewal stopped and the finished result was discarded and
// deleted, so a retry executed the side effect a second time.
func TestSuccessAfterCallerCancelIsStored(t *testing.T) {
	var invoked atomic.Int64
	idp := idempotent.MustNewWithRedis(redistest.Client(t)).MustHandlerFunc(
		func(ctx context.Context, req string) (string, error) {
			invoked.Add(1)
			time.Sleep(300 * time.Millisecond) // ignores ctx, e.g. a non-cancelable external call
			return "charged", nil
		},
		idempotent.HandlerConfig{LockTTL: 100 * time.Millisecond},
	)

	cctx, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer cancel()
	res, shared, err := idp.Do(cctx, t.Name(), "req")
	assert.NoError(t, err)
	assert.False(t, shared)
	assert.Equal(t, "charged", res)

	res, shared, err = idp.Do(ctx, t.Name(), "req")
	assert.NoError(t, err)
	assert.True(t, shared)
	assert.Equal(t, "charged", res)
	assert.Equal(t, int64(1), invoked.Load())
}

// A failed task stores nothing, so the same key can be retried.
func TestErrorIsNotStored(t *testing.T) {
	var invoked atomic.Int64
	idp := idempotent.MustNewWithRedis(redistest.Client(t)).MustHandlerFunc(
		func(ctx context.Context, req string) (string, error) {
			if invoked.Add(1) == 1 {
				return "", assert.AnError
			}
			return "ok", nil
		},
		idempotent.HandlerConfig{},
	)
	_, _, err := idp.Do(ctx, t.Name(), "req")
	assert.ErrorIs(t, err, assert.AnError)

	res, shared, err := idp.Do(ctx, t.Name(), "req")
	assert.NoError(t, err)
	assert.False(t, shared)
	assert.Equal(t, "ok", res)
}
