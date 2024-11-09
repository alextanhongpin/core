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

	tracker := metrics.NewTracker(t.Name(), redistest.Client(t))
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

func randDuration(duration time.Duration) time.Duration {
	return time.Duration(rand.Int64N(duration.Milliseconds())) * time.Millisecond
}
