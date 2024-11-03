package metrics_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
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
	tracker := metrics.NewTracker(redistest.Client(t))
	ctx := context.Background()

	is := assert.New(t)
	for range 100 {
		userID := strconv.Itoa(rand.IntN(10))
		is.Nil(tracker.Record(ctx, t.Name(), "GET /foo", userID, randDuration(10*time.Second)))
		is.Nil(tracker.Record(ctx, t.Name(), "GET /bar", userID, randDuration(5*time.Second)))
	}
	stats, err := tracker.Stats(ctx, t.Name())
	is.Nil(err)
	for _, s := range stats {
		t.Log(s.String())
		t.Log()
	}
}

func TestTrackerHandler(t *testing.T) {
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello world")
	})

	now := time.Now()
	year := now.Format("2006")
	month := now.Format("2006-01")
	date := now.Format("2006-01-02")
	keys := []string{year, month, date}

	tracker := metrics.NewTracker(redistest.Client(t))
	h = metrics.TrackerHandler(h, func() []string {
		return keys
	}, tracker, func(r *http.Request) string {
		return "user-id"
	})
	mux := http.NewServeMux()
	mux.Handle("GET /", h)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := srv.Client()
	is := assert.New(t)
	for range 100 {
		_, err := client.Get(srv.URL)
		is.Nil(err)
	}

	ctx := context.Background()
	for _, key := range keys {
		stats, err := tracker.Stats(ctx, key)
		is.Nil(err)
		for _, s := range stats {
			t.Log(s.String())
			t.Log()
		}
	}
}

func randDuration(duration time.Duration) time.Duration {
	return time.Duration(rand.Int64N(duration.Milliseconds())) * time.Millisecond
}
