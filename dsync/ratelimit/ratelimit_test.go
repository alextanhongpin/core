package ratelimit_test

import (
	"context"
	"os"
	"testing"

	"github.com/alextanhongpin/core/storage/redis/redistest"
	redis "github.com/redis/go-redis/v9"
)

func TestMain(m *testing.M) {
	stop := redistest.Init()
	code := m.Run()
	stop()
	os.Exit(code)
}

func newClient(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})

	t.Cleanup(func() {
		client.FlushAll(context.Background())
		client.Close()
	})
	return client
}
