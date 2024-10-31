package metrics

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"expvar"
	"fmt"
	"net/http"
	"time"

	"github.com/alextanhongpin/core/dsync/probs"
	"github.com/alextanhongpin/core/http/httputil"
	redis "github.com/redis/go-redis/v9"
)

var (
	StatusTotal   = expvar.NewMap("status_total")
	RequestsTotal = expvar.NewMap("requests_total")
	ErrorsTotal   = expvar.NewMap("errors_total")
)

// CounterHandler tracks the success and error count.
// Install the expvar.Handler:
// mux.Handle("GET /debug/vars", expvar.Handler())
func CounterHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wr := httputil.NewResponseWriterRecorder(w)
		h.ServeHTTP(wr, r)

		path := fmt.Sprintf("%s - %d", cmp.Or(r.Pattern, r.URL.Path), wr.StatusCode())
		RequestsTotal.Add("ALL", 1)
		RequestsTotal.Add(path, 1)
		StatusTotal.Add(fmt.Sprint(wr.StatusCode()), 1)
	})
}

func TrackerHandler(h http.Handler, tracker *Tracker, userFn func(r *http.Request) string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wr := httputil.NewResponseWriterRecorder(w)
		h.ServeHTTP(wr, r)

		path := fmt.Sprintf("%s - %d", cmp.Or(r.Pattern, r.URL.Path), wr.StatusCode())
		tracker.Record(r.Context(), path, userFn(r), start)
	})
}

type Tracker struct {
	name string
	// t-digest add (path) - measure api latency
	td *probs.TDigest
	// cms add(path, count) - track total api calls
	cms *probs.CountMinSketch
	// hyperloglog.add(path, user) - measure unique api calls
	hll *probs.HyperLogLog
	// top-k add (path) - find top api calls
	topK *probs.TopK
}

func NewTracker(name string, client *redis.Client) *Tracker {
	return &Tracker{
		name: name,
		// Track frequency of API calls.
		cms: probs.NewCountMinSketch(client),

		// Track unique page views by user
		hll: probs.NewHyperLogLog(client),

		// Track API latency
		td: probs.NewTDigest(client),

		// Track top-10 requests.
		topK: probs.NewTopK(client),
	}
}

func (t *Tracker) Record(ctx context.Context, path, userID string, start time.Time) error {
	cms := fmt.Sprintf("%s:cms:api", t.name)
	hll := fmt.Sprintf("%s:hll:api", t.name)
	td := fmt.Sprintf("%s:td:api", t.name)
	topK := fmt.Sprintf("%s:top_k:api", t.name)

	return errors.Join(
		t.countUnique(ctx, cms, path, userID, time.Now()),
		t.countOccurences(ctx, hll, path),
		t.rank(ctx, topK, path),
		t.recordLatency(ctx, td, path, start),
	)
}

func (t *Tracker) Handler(at time.Time) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stats, err := t.Stats(r.Context(), at)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		if err := json.NewEncoder(w).Encode(stats); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func (t *Tracker) Stats(ctx context.Context, at time.Time) ([]Stats, error) {
	cms := fmt.Sprintf("%s:cms:api", t.name)
	hll := fmt.Sprintf("%s:hll:api", t.name)
	td := fmt.Sprintf("%s:td:api", t.name)
	topK := fmt.Sprintf("%s:top_k:api", t.name)

	paths, err := t.rankings(ctx, topK)
	if err != nil {
		return nil, err
	}

	stats := make([]Stats, len(paths))
	for i, path := range paths {
		daily, monthly, yearly, err := t.totalUnique(ctx, cms, path, at)
		if err != nil {
			return nil, err
		}

		vals, err := t.latency(ctx, td, path)
		if err != nil {
			return nil, err
		}

		total, err := t.totalOccurences(ctx, hll, path)
		if err != nil {
			return nil, err
		}

		stats[i] = Stats{
			Path:    path,
			P50:     vals[0],
			P90:     vals[1],
			P95:     vals[2],
			Total:   total,
			Daily:   daily,
			Monthly: monthly,
			Yearly:  yearly,
		}
	}

	return stats, nil
}

func (t *Tracker) recordLatency(ctx context.Context, key, path string, start time.Time) error {
	_, err := t.td.Add(ctx, fmt.Sprintf("%s:%s", key, path), time.Since(start).Seconds())
	return err
}

func (t *Tracker) latency(ctx context.Context, key, path string) ([]float64, error) {
	return t.td.Quantile(ctx, fmt.Sprintf("%s:%s", key, path), 0.5, 0.9, 0.95)
}

func (t *Tracker) countOccurences(ctx context.Context, key, path string) error {
	_, err := t.cms.IncrBy(ctx, key, map[string]int64{
		path: 1,
	})
	return err
}

func (t *Tracker) totalOccurences(ctx context.Context, key, path string) (int64, error) {
	counts, err := t.cms.Query(ctx, key, path)
	if err != nil {
		return 0, err
	}

	return counts[0], nil
}

func (t *Tracker) countUnique(ctx context.Context, prefix, page, userID string, at time.Time) error {
	key := Prefix(prefix + ":%s:ts:%s")

	_, err1 := t.hll.Add(ctx, key.Format(page, at.Format("2006-01-02")), userID)
	_, err2 := t.hll.Add(ctx, key.Format(page, at.Format("2006-01")), userID)
	_, err3 := t.hll.Add(ctx, key.Format(page, at.Format("2006")), userID)
	return errors.Join(err1, err2, err3)
}

func (t *Tracker) totalUnique(ctx context.Context, prefix, page string, at time.Time) (day, month, year int64, err error) {
	key := Prefix(prefix + ":%s:ts:%s")

	day, err1 := t.hll.Count(ctx, key.Format(page, at.Format("2006-01-02")))
	month, err2 := t.hll.Count(ctx, key.Format(page, at.Format("2006-01")))
	year, err3 := t.hll.Count(ctx, key.Format(page, at.Format("2006")))
	return day, month, year, errors.Join(err1, err2, err3)
}

func (t *Tracker) rank(ctx context.Context, key, path string) error {
	_, err := t.topK.Add(ctx, key, path)
	return err
}

func (t *Tracker) rankings(ctx context.Context, key string) ([]string, error) {
	return t.topK.List(ctx, key)
}

type Stats struct {
	Path    string
	P50     float64
	P90     float64
	P95     float64
	Daily   int64
	Monthly int64
	Yearly  int64
	Total   int64
}

func (s *Stats) String() string {
	return fmt.Sprintf(`%s
Requests total: %d
p50/p90/p95 (in seconds): %f, %f, %fs
d/m/y: %d/%d/%d`,
		s.Path,
		s.Total,
		s.P50, s.P90, s.P95,
		s.Daily, s.Monthly, s.Yearly,
	)
}

type Prefix string

func (p Prefix) Format(args ...any) string {
	return fmt.Sprintf(string(p), args...)
}
