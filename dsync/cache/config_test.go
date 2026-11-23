package cache_test

import (
	"github.com/alextanhongpin/core/dsync/cache"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestConfigurationAndClientOwnership(t *testing.T) {
	_, err := cache.New[string](cache.Config{})
	require.Error(t, err)
	_, err = cache.NewRedis(nil)
	require.Error(t, err)
	client := newClient(t)
	storage := cache.MustNewRedis(client)
	cfg := cache.Config{Storage: storage}
	c, err := cache.New[string](cfg)
	require.NoError(t, err)
	cfg.Storage = nil
	require.NoError(t, c.Store(t.Context(), "owned", "value", time.Minute))
	require.NoError(t, c.Close())
	require.NoError(t, client.Ping(t.Context()).Err())
	require.Nil(t, cfg.Codec)
}
