package probs_test

import (
	"context"
	"os"
	"testing"

	"github.com/alextanhongpin/dbtx/testing/redistest"
)

var ctx = context.Background()

// redisImage ships the probabilistic data structures (Bloom, Cuckoo,
// Count-Min Sketch, Top-K and t-digest) built in, as every Redis 8+ image does.
const redisImage = "redis:8.6.2"

func TestMain(m *testing.M) {
	stop := redistest.Init(redistest.Options{Image: redisImage})
	code := m.Run()
	_ = stop()
	os.Exit(code)
}
