package metrics

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
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

		path := fmt.Sprintf("%s - %d", r.Pattern, wr.StatusCode())
		RequestsTotal.Add("ALL", 1)
		RequestsTotal.Add(path, 1)
		StatusTotal.Add(fmt.Sprint(wr.StatusCode()), 1)
	})
}

func TrackerHandler(h http.Handler, tracker *Tracker, userFn func(r *http.Request) string, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wr := httputil.NewResponseWriterRecorder(w)
		h.ServeHTTP(wr, r)

		path := fmt.Sprintf("%s - %d", r.Pattern, wr.StatusCode())
		user := userFn(r)
		took := time.Since(start)
		err := tracker.Record(r.Context(), path, user, took)
		if err != nil {
			logger.Error(err.Error())
		}
	})
}

func TrackerStatsHandler(tracker *Tracker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		if at := r.URL.Query().Get("at"); at != "" {
			t, err := time.Parse(time.DateOnly, at)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)

				return
			}
			now = t
		}

		var sb strings.Builder
		ctx := r.Context()
		stats, err := tracker.Stats(ctx, now)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}
		for _, stat := range stats {
			_, err = sb.WriteString(fmt.Sprintf("at: %s\n%s\n\n", now, stat.String()))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)

				return
			}
		}

		fmt.Fprint(w, sb.String())
	})
}

type Tracker struct {
	Name string
	Now  func() time.Time
	cms  *probs.CountMinSketch // Track frequency of API calls.
	hll  *probs.HyperLogLog    // Track unique page views by user.
	td   *probs.TDigest        // Track API latency.
	topK *probs.TopK           // Track top-10 requests.
}

func NewTracker(name string, client *redis.Client) *Tracker {
	return &Tracker{
		Name: name,
		Now:  time.Now,
		cms:  probs.NewCountMinSketch(client),
		hll:  probs.NewHyperLogLog(client),
		td:   probs.NewTDigest(client),
		topK: probs.NewTopK(client),
	}
}

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

func (t *Tracker) Stats(ctx context.Context, at time.Time) ([]Stats, error) {
	key := t.Name
	day := at.Format(time.DateOnly)
	paths, err := t.rankings(ctx, join(key, "top_k"))
	if err != nil {
		return nil, err
	}

	stats := make([]Stats, len(paths))
	for i, path := range paths {
		occurences, err := t.totalOccurences(ctx, join(key, "cms", day), path)
		if err != nil {
			return nil, err
		}

		unique, err := t.totalUnique(ctx, join(key, "hll", day, path))
		if err != nil {
			return nil, err
		}

		vals, err := t.latency(ctx, join(key, "td", day, path))
		if err != nil {
			return nil, err
		}

		stats[i] = Stats{
			Path:   path,
			P50:    vals[0],
			P90:    vals[1],
			P95:    vals[2],
			Total:  occurences,
			Unique: unique,
		}
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
	return t.topK.List(ctx, key)
}

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
p50/p90/p95 (in seconds): %v, %s, %s`,
		s.Path,
		s.Unique,
		s.Total,
		seconds(s.P50),
		seconds(s.P90),
		seconds(s.P95),
	)
}

type Prefix string

func (p Prefix) Format(args ...any) string {
	return fmt.Sprintf(string(p), args...)
}

func join(s ...string) string {
	return strings.Join(s, ":")
}

func seconds(f float64) time.Duration {
	return time.Duration(f * float64(time.Second))
}
