package metrics

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/alextanhongpin/core/dsync/probs"
	redis "github.com/redis/go-redis/v9"
)

// Expvar counters updated by CounterHandler. They are published under these
// names when the package is imported.
var (
	StatusTotal   = expvar.NewMap("status_total")
	RequestsTotal = expvar.NewMap("requests_total")
)

// CounterHandler counts requests in total ("ALL"), by
// "<route pattern> - <status code>" in RequestsTotal, and by status code in
// StatusTotal. Expose them with:
//
//	mux.Handle("GET /debug/vars", expvar.Handler())
func CounterHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wr := &responseRecorder{ResponseWriter: w, code: http.StatusOK}
		h.ServeHTTP(wr, r)

		path := fmt.Sprintf("%s - %d", r.Pattern, wr.code)
		RequestsTotal.Add("ALL", 1)
		RequestsTotal.Add(path, 1)
		StatusTotal.Add(fmt.Sprint(wr.code), 1)
	})
}

// TrackerHandler records each request in tracker, keyed by
// "<route pattern> - <status code>" and the user returned by userFn. Record
// errors are logged to logger, or to slog.Default when logger is nil.
func TrackerHandler(h http.Handler, tracker *Tracker, userFn func(r *http.Request) string, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wr := &responseRecorder{ResponseWriter: w, code: http.StatusOK}
		h.ServeHTTP(wr, r)

		path := fmt.Sprintf("%s - %d", r.Pattern, wr.code)
		user := userFn(r)
		took := time.Since(start)
		err := tracker.Record(r.Context(), path, user, took)
		if err != nil {
			logger.Error(err.Error())
		}
	})
}

// TrackerStatsHandler writes tracker.Stats as plain text. The optional "at"
// query parameter (YYYY-MM-DD) selects the day; it defaults to today.
func TrackerStatsHandler(tracker *Tracker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := tracker.Now()
		if at := r.URL.Query().Get("at"); at != "" {
			t, err := time.Parse(time.DateOnly, at)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)

				return
			}
			now = t
		}

		stats, err := tracker.Stats(r.Context(), now)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		var sb strings.Builder
		for _, stat := range stats {
			fmt.Fprintf(&sb, "at: %s\n%s\n\n", now.Format(time.DateOnly), stat.String())
		}

		fmt.Fprint(w, sb.String())
	})
}

// Tracker keeps, in Redis, per-day request counts, unique users and latency
// quantiles per path, plus an all-time top 10 of paths. It requires the
// RedisBloom module (e.g. Redis Stack).
type Tracker struct {
	Name string
	Now  func() time.Time
	cms  *probs.CountMinSketch // Track frequency of API calls.
	hll  *probs.HyperLogLog    // Track unique page views by user.
	td   *probs.TDigest        // Track API latency.
	topK *probs.TopK           // Track top-10 requests.
}

// NewTracker returns a Tracker whose keys are prefixed with name. It panics
// if client is nil.
func NewTracker(name string, client *redis.Client) *Tracker {
	return &Tracker{
		Name: name,
		Now:  time.Now,
		cms:  probs.MustNewCountMinSketch(client),
		hll:  probs.MustNewHyperLogLog(client),
		td:   probs.MustNewTDigest(client),
		topK: probs.MustNewTopK(client),
	}
}

// Record adds one request to path by userID that took duration.
func (t *Tracker) Record(ctx context.Context, path, userID string, duration time.Duration) error {
	day := t.Now().Format(time.DateOnly)
	key := t.Name

	return errors.Join(
		// We calculate the all-time rank.
		t.rank(ctx, join(key, "top_k"), path),
		t.countOccurences(ctx, join(key, "cms", day), path),
		t.countUnique(ctx, join(key, "hll", day, path), userID),
		t.recordLatency(ctx, join(key, "td", day, path), duration),
	)
}

// Stats returns the statistics for the day of at, for each all-time top path
// that has requests on that day. It returns no stats for a day without
// requests.
func (t *Tracker) Stats(ctx context.Context, at time.Time) ([]Stats, error) {
	key := t.Name
	day := at.Format(time.DateOnly)
	paths, err := t.rankings(ctx, join(key, "top_k"))
	if err != nil {
		return nil, err
	}

	stats := make([]Stats, 0, len(paths))
	for _, path := range paths {
		occurences, err := t.totalOccurences(ctx, join(key, "cms", day), path)
		if err != nil {
			return nil, err
		}
		if occurences == 0 {
			// Not requested on that day; the latency digest does not exist.
			continue
		}

		unique, err := t.totalUnique(ctx, join(key, "hll", day, path))
		if err != nil {
			return nil, err
		}

		vals, err := t.latency(ctx, join(key, "td", day, path))
		if err != nil {
			return nil, err
		}

		stats = append(stats, Stats{
			Path:   path,
			P50:    vals[0],
			P90:    vals[1],
			P95:    vals[2],
			Total:  occurences,
			Unique: unique,
		})
	}

	return stats, nil
}

func (t *Tracker) recordLatency(ctx context.Context, path string, duration time.Duration) error {
	_, err := t.td.Add(ctx, path, duration.Seconds())
	return err
}

func (t *Tracker) latency(ctx context.Context, path string) ([]float64, error) {
	return t.td.Quantile(ctx, path, 0.5, 0.9, 0.95)
}

func (t *Tracker) countOccurences(ctx context.Context, key, path string) error {
	_, _, err := t.cms.IncrBy(ctx, key, map[string]int64{
		path: 1,
	})
	return err
}

func (t *Tracker) totalOccurences(ctx context.Context, key, path string) (int64, error) {
	counts, err := t.cms.Query(ctx, key, path)
	if probs.KeyDoesNotExistError(err) {
		// Nothing was recorded on that day.
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return counts[0], nil
}

func (t *Tracker) countUnique(ctx context.Context, key, userID string) error {
	_, err := t.hll.Add(ctx, key, userID)
	return err
}

func (t *Tracker) totalUnique(ctx context.Context, key string) (int64, error) {
	return t.hll.Count(ctx, key)
}

func (t *Tracker) rank(ctx context.Context, key, path string) error {
	_, err := t.topK.Add(ctx, key, path)
	return err
}

func (t *Tracker) rankings(ctx context.Context, key string) ([]string, error) {
	paths, err := t.topK.List(ctx, key)
	if probs.KeyDoesNotExistError(err) {
		// Nothing was ever recorded.
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Some RedisBloom versions return unused slots as nil.
	return slices.DeleteFunc(paths, func(p string) bool { return p == "" }), nil
}

// Stats summarizes one path for one day. P50, P90 and P95 are latencies in
// seconds.
type Stats struct {
	Path   string
	P50    float64
	P90    float64
	P95    float64
	Unique int64
	Total  int64
}

func (s *Stats) String() string {
	return fmt.Sprintf(`%s
unique/total: %d/%d
p50/p90/p95: %s, %s, %s`,
		s.Path,
		s.Unique,
		s.Total,
		seconds(s.P50),
		seconds(s.P90),
		seconds(s.P95),
	)
}

func join(s ...string) string {
	return strings.Join(s, ":")
}

func seconds(f float64) time.Duration {
	return time.Duration(f * float64(time.Second))
}
