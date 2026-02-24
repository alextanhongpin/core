package ratelimit_test

import (
	"os"
	"testing"

	"github.com/alextanhongpin/core/dsync/ratelimit"
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

	client := redistest.Client(t)
	if err := ratelimit.Setup(t.Context(), client); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	return client
}
