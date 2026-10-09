package singleflight_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/alextanhongpin/core/dsync/lock"
	"github.com/alextanhongpin/core/dsync/singleflight"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestValueConfiguration(t *testing.T) {
	_, err := singleflight.New(nil, singleflight.Config{})
	require.Error(t, err)
	_, err = singleflight.NewCache[string](nil, singleflight.CacheConfig{})
	require.Error(t, err)
	client := redistest.Client(t)
	for _, cfg := range []singleflight.Config{{LockTTL: -1}, {LockTTL: time.Nanosecond}, {WaitTTL: -1}, {PollInterval: -1}} {
		_, err = singleflight.New(client, cfg)
		require.Error(t, err)
	}
	cfg := singleflight.Config{}
	g, err := singleflight.New(client, cfg)
	require.NoError(t, err)
	require.Zero(t, cfg.LockTTL)
	cfg.LockTTL = -1
	did, err := g.Do(t.Context(), t.Name(), func(context.Context) error { return nil })
	require.NoError(t, err)
	require.True(t, did)
	require.NoError(t, client.Ping(t.Context()).Err())
}
func TestLostLeaseStopsCachePublication(t *testing.T) {
	client := redistest.Client(t)
	c := singleflight.MustNewCache[string](client, singleflight.CacheConfig{LockTTL: time.Second})
	key := t.Name()
	_, _, err := c.LoadOrStore(t.Context(), key, func(context.Context) (string, error) {
		lease := fmt.Sprintf("singleflight:lease:%x", sha256.Sum256([]byte(key+":fetch")))
		require.NoError(t, client.Set(t.Context(), lease, "another-owner", time.Second).Err())
		return "stale", nil
	}, time.Minute)
	require.ErrorIs(t, err, lock.ErrExpired)
	require.EqualValues(t, 0, client.Exists(t.Context(), key).Val())
}
