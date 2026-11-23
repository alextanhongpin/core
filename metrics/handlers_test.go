package metrics_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/alextanhongpin/core/metrics"
	"github.com/stretchr/testify/assert"
)

func TestTracker(t *testing.T) {
	tracker := metrics.NewTracker(t.Name(), redisClient(t))
	ctx := context.Background()

	is := assert.New(t)
	for range 100 {
		userID := strconv.Itoa(rand.IntN(10))
		is.Nil(tracker.Record(ctx, "GET /foo", userID, randDuration(10*time.Second)))
		is.Nil(tracker.Record(ctx, "GET /bar", userID, randDuration(5*time.Second)))
	}
	stats, err := tracker.Stats(ctx, time.Now())
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

	userFn := func(r *http.Request) string {
		return "user-id"
	}

	tracker := metrics.NewTracker(t.Name(), redisClient(t))
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	h = metrics.TrackerHandler(h, tracker, userFn, logger)
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

	{
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		h := metrics.TrackerStatsHandler(tracker)
		h.ServeHTTP(w, r)
		res := w.Result()
		is.Equal(200, res.StatusCode)
		b, err := io.ReadAll(res.Body)
		is.Nil(err)
		t.Log(string(b))
	}
}

func TestTrackerStatsMissingDay(t *testing.T) {
	tracker := metrics.NewTracker(t.Name(), redisClient(t))
	ctx := context.Background()
	is := assert.New(t)

	// Nothing recorded yet.
	stats, err := tracker.Stats(ctx, time.Now())
	is.Nil(err)
	is.Empty(stats)

	day1 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	day2 := day1.AddDate(0, 0, 1)

	tracker.Now = func() time.Time { return day1 }
	is.Nil(tracker.Record(ctx, "GET /foo", "a", time.Second))
	is.Nil(tracker.Record(ctx, "GET /bar", "a", time.Second))

	tracker.Now = func() time.Time { return day2 }
	is.Nil(tracker.Record(ctx, "GET /foo", "b", 2*time.Second))
	is.Nil(tracker.Record(ctx, "GET /foo", "c", 2*time.Second))

	// A top path without requests on day2 is skipped instead of failing on
	// its missing latency digest.
	stats, err = tracker.Stats(ctx, day2)
	is.Nil(err)
	is.Len(stats, 1)
	is.Equal("GET /foo", stats[0].Path)
	is.Equal(int64(2), stats[0].Total)
	is.Equal(int64(2), stats[0].Unique)
	is.Equal(2.0, stats[0].P50)

	// Paths are ranked by all-time frequency: /foo (3) before /bar (1).
	stats, err = tracker.Stats(ctx, day1)
	is.Nil(err)
	is.Len(stats, 2)
	is.Equal("GET /foo", stats[0].Path)
	is.Equal("GET /bar", stats[1].Path)

	// A day without any requests returns no stats.
	stats, err = tracker.Stats(ctx, day2.AddDate(0, 0, 1))
	is.Nil(err)
	is.Empty(stats)
}

func randDuration(duration time.Duration) time.Duration {
	return time.Duration(rand.Int64N(duration.Milliseconds())) * time.Millisecond
}
