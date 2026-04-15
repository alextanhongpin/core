package cache_test

import (
	"context"
	"os"
	"testing"

	"github.com/alextanhongpin/core/dsync/cache"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	redis "github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func TestMain(m *testing.M) {
	stop := redistest.Init(redistest.Options{
		Image: "redis:8.6.2",
	})
	code := m.Run()
	stop()
	os.Exit(code)
}

func TestRedis(t *testing.T) {
	c := cache.NewRedis(newClient(t))

	testStorage(t, c)
}

func newClient(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})

	t.Helper()
	t.Cleanup(func() {
		client.FlushAll(ctx).Err()
		client.Close()
	})

	return client
}
