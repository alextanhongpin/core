package probs_test

import (
	"context"
	"testing"

	"github.com/alextanhongpin/core/storage/redis/redistest"
)

var ctx = context.Background()

func TestMain(m *testing.M) {
	stop := redistest.Init(redistest.Image("redis/redis-stack:6.2.6-v17"))
	defer stop()

	m.Run()
}
