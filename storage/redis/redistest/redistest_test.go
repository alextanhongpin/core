package redistest_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/alextanhongpin/core/storage/redis/redistest"
	redis "github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func TestMain(m *testing.M) {
	stop := redistest.Init()
	code := m.Run()
	stop()
	os.Exit(code)
}

func TestRedisPing(t *testing.T) {
	t.Run("with addr", func(t *testing.T) {
		db := redis.NewClient(&redis.Options{
			Addr: redistest.Addr(),
		})

		if err := db.Ping(ctx).Err(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("with client", func(t *testing.T) {
		db := redistest.Client(t)
		if err := db.Ping(ctx).Err(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestRedisNew(t *testing.T) {
	db1 := redistest.New(t).Client()
	db2 := redistest.New(t).Client()

	if err := db1.Set(ctx, t.Name(), "1", time.Second).Err(); err != nil {
		t.Fatal(err)
	}

	got, err := db1.Get(ctx, t.Name()).Result()
	if err != nil {
		t.Fatal(err)
	}
	if want := "1"; want != got {
		t.Fatalf("want %s, got %s", want, got)
	}

	err = db2.Get(ctx, t.Name()).Err()
	if !errors.Is(err, redis.Nil) {
		t.Fatalf("want redis.Nil, got %v", err)
	}
}
