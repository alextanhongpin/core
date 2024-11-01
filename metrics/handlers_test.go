package metrics_test

import (
	"context"
	"math/rand/v2"
	"strconv"
	"testing"
	"time"

	"github.com/alextanhongpin/core/metrics"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	stop := redistest.Init(redistest.Image("redis/redis-stack:6.2.6-v17"))
	defer stop()

	m.Run()
}
func TestTracker(t *testing.T) {
	tracker := metrics.NewTracker(t.Name(), redistest.Client(t))
	ctx := context.Background()

	is := assert.New(t)
	for range 100 {
		userID := strconv.Itoa(rand.IntN(10))
		is.Nil(tracker.Record(ctx, "GET /foo", userID, -time.Duration(rand.Int64N(10_000))*time.Millisecond, "now"))
		is.Nil(tracker.Record(ctx, "GET /bar", userID, -time.Duration(rand.Int64N(5_000))*time.Millisecond, "now"))
	}
	stats, err := tracker.Stats(ctx, "now")
	is.Nil(err)
	for _, s := range stats {
		t.Log(s.String())
		t.Log()
	}
}
