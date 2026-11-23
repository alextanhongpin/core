package metrics

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// Sizing of the Redis structures created on first use.
const (
	topK                = 10    // paths kept in the all-time ranking
	cmsErrorRate        = 0.001 // count-min sketch overestimate, as a fraction of the total
	cmsErrorProbability = 0.002 // probability of exceeding cmsErrorRate
)

// Tracker keeps, in Redis, per-day request counts (count-min sketch), unique
// users (HyperLogLog) and latency quantiles (t-digest) per path, plus an
// all-time top 10 of paths (top-k). It requires the RedisBloom module
// (e.g. Redis Stack). Keys are created on first use and never expire.
type Tracker struct {
	Name   string
	Now    func() time.Time
	client *redis.Client
}

// NewTracker returns a Tracker whose keys are prefixed with name. It panics
// if client is nil.
func NewTracker(name string, client *redis.Client) *Tracker {
	if client == nil {
		panic("metrics: Redis client is required")
	}
	return &Tracker{
		Name:   name,
		Now:    time.Now,
		client: client,
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

		unique, err := t.client.PFCount(ctx, join(key, "hll", day, path)).Result()
		if err != nil {
			return nil, err
		}

		vals, err := t.client.TDigestQuantile(ctx, join(key, "td", day, path), 0.5, 0.9, 0.95).Result()
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

func (t *Tracker) recordLatency(ctx context.Context, key string, duration time.Duration) error {
	return createOnMissing(
		func() error { return t.client.TDigestAdd(ctx, key, duration.Seconds()).Err() },
		func() error { return t.client.TDigestCreate(ctx, key).Err() },
	)
}

func (t *Tracker) countOccurences(ctx context.Context, key, path string) error {
	return createOnMissing(
		func() error { return t.client.CMSIncrBy(ctx, key, path, 1).Err() },
		func() error { return t.client.CMSInitByProb(ctx, key, cmsErrorRate, cmsErrorProbability).Err() },
	)
}

func (t *Tracker) totalOccurences(ctx context.Context, key, path string) (int64, error) {
	counts, err := t.client.CMSQuery(ctx, key, path).Result()
	if keyDoesNotExist(err) {
		// Nothing was recorded on that day.
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return counts[0], nil
}

func (t *Tracker) countUnique(ctx context.Context, key, userID string) error {
	return t.client.PFAdd(ctx, key, userID).Err()
}

func (t *Tracker) rank(ctx context.Context, key, path string) error {
	return createOnMissing(
		func() error { return t.client.TopKAdd(ctx, key, path).Err() },
		func() error { return t.client.TopKReserve(ctx, key, topK).Err() },
	)
}

// rankings returns the top paths, most frequent first.
func (t *Tracker) rankings(ctx context.Context, key string) ([]string, error) {
	ranked, err := t.client.TopKListWithCount(ctx, key).Result()
	if keyDoesNotExist(err) {
		// Nothing was ever recorded.
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Some RedisBloom versions return unused slots as nil, read as "".
	delete(ranked, "")
	paths := make([]string, 0, len(ranked))
	for path := range ranked {
		paths = append(paths, path)
	}
	slices.SortFunc(paths, func(a, b string) int {
		if c := ranked[b] - ranked[a]; c != 0 {
			if c > 0 {
				return 1
			}
			return -1
		}
		return strings.Compare(a, b)
	})
	return paths, nil
}

// createOnMissing runs op, and when the key does not exist yet, creates it
// and runs op again. A concurrent creation is not an error.
func createOnMissing(op, create func() error) error {
	err := op()
	if !keyDoesNotExist(err) {
		return err
	}
	if err := create(); err != nil && !keyAlreadyExists(err) {
		return err
	}
	return op()
}

// RedisBloom replies "<TYPE>: key does not exist" and "... key already exists".
func keyDoesNotExist(err error) bool {
	return err != nil && strings.HasSuffix(err.Error(), "key does not exist")
}

func keyAlreadyExists(err error) bool {
	return err != nil && strings.HasSuffix(err.Error(), "key already exists")
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
