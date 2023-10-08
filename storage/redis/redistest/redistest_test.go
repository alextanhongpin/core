package redistest_test

import (
	"context"
	"os"
	"testing"

	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/redis/go-redis/v9"
)

func TestMain(m *testing.M) {
	stop := redistest.Init()
	code := m.Run()
	stop()
	os.Exit(code)
}

func TestRedisPing(t *testing.T) {
	db := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})

	ctx := context.Background()
	if err := db.Ping(ctx).Err(); err != nil {
		t.Fatal(err)
	}
}
