package probs

import (
	"context"

	redis "github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

// TDigest estimates percentiles (p50, p99, ...) of a stream of numbers, for
// example request latencies, using a small fixed amount of memory (Redis
// TDIGEST.* commands). Estimates are most accurate near the extremes
// (p1, p99).
//
// Add creates a missing digest with the Redis default compression (100),
// so calling Create first is optional. A TDigest is safe for concurrent
// use.
type TDigest struct {
	client *redis.Client
	group  singleflight.Group
}

// NewTDigest returns a TDigest that uses client. The client is borrowed:
// the caller keeps ownership and must close it. It returns an error if
// client is nil.
func NewTDigest(client *redis.Client) (*TDigest, error) {
	if client == nil {
		return nil, errNilClient
	}
	return &TDigest{client: client}, nil
}

// MustNewTDigest is like NewTDigest but panics on error.
func MustNewTDigest(client *redis.Client) *TDigest {
	v, err := NewTDigest(client)
	if err != nil {
		panic(err)
	}
	return v
}

// CreateWithCompression creates an empty digest at key (TDIGEST.CREATE
// COMPRESSION). Higher compression is more accurate and uses more memory.
// It is idempotent: if key exists it returns OK with exists set to true.
func (t *TDigest) CreateWithCompression(ctx context.Context, key string, compression int64) (status string, exists bool, err error) {
	status, err = t.client.TDigestCreateWithCompression(ctx, key, compression).Result()
	if KeyAlreadyExistsError(err) {
		return OK, true, nil
	}

	return status, false, err
}

// Create creates an empty digest at key with the default compression
// (TDIGEST.CREATE). It is idempotent like CreateWithCompression.
func (t *TDigest) Create(ctx context.Context, key string) (status string, exists bool, err error) {
	status, err = t.client.TDigestCreate(ctx, key).Result()
	if KeyAlreadyExistsError(err) {
		return OK, true, nil
	}

	return status, false, err
}

// Add adds observations to the digest at key, creating it with Create if
// it does not exist (TDIGEST.ADD).
func (t *TDigest) Add(ctx context.Context, key string, values ...float64) (string, error) {
	status, err := t.client.TDigestAdd(ctx, key, values...).Result()
	if err == nil {
		return status, nil
	}

	if !KeyDoesNotExistError(err) {
		return "", err
	}

	_, err, _ = t.group.Do(key, func() (any, error) {
		_, _, err := t.Create(ctx, key)
		return nil, err
	})
	if err != nil {
		return "", err
	}

	return t.client.TDigestAdd(ctx, key, values...).Result()
}

// CDF returns, for each value, the estimated fraction of observations
// that are less than or equal to it (TDIGEST.CDF), between 0 and 1.
func (t *TDigest) CDF(ctx context.Context, key string, values ...float64) ([]float64, error) {
	return t.client.TDigestCDF(ctx, key, values...).Result()
}

// Quantile returns the estimated value at each quantile (TDIGEST.QUANTILE).
// Quantiles are fractions: 0.5 is the median, 0.99 is p99. An empty digest
// returns NaN.
func (t *TDigest) Quantile(ctx context.Context, key string, quantiles ...float64) ([]float64, error) {
	return t.client.TDigestQuantile(ctx, key, quantiles...).Result()
}

// Min returns the smallest observation (TDIGEST.MIN), or NaN if the digest
// is empty.
func (t *TDigest) Min(ctx context.Context, key string) (float64, error) {
	return t.client.TDigestMin(ctx, key).Result()
}

// Max returns the largest observation (TDIGEST.MAX), or NaN if the digest
// is empty.
func (t *TDigest) Max(ctx context.Context, key string) (float64, error) {
	return t.client.TDigestMax(ctx, key).Result()
}

// Rank returns, for each value, the estimated number of observations
// smaller than it (TDIGEST.RANK); 0 is the smallest. Redis returns -1
// for a value below the minimum and -2 for an empty digest.
func (t *TDigest) Rank(ctx context.Context, key string, values ...float64) ([]int64, error) {
	return t.client.TDigestRank(ctx, key, values...).Result()
}

// RevRank returns, for each value, the estimated number of observations
// larger than it (TDIGEST.REVRANK); 0 is the largest. Redis returns -1
// for a value above the maximum and -2 for an empty digest.
func (t *TDigest) RevRank(ctx context.Context, key string, values ...float64) ([]int64, error) {
	return t.client.TDigestRevRank(ctx, key, values...).Result()
}

// ByRank returns the estimated observation at each rank, counting from the
// smallest (rank 0) (TDIGEST.BYRANK).
func (t *TDigest) ByRank(ctx context.Context, key string, ranks ...uint64) ([]float64, error) {
	return t.client.TDigestByRank(ctx, key, ranks...).Result()
}

// ByRevRank returns the estimated observation at each rank, counting from
// the largest (rank 0) (TDIGEST.BYREVRANK).
func (t *TDigest) ByRevRank(ctx context.Context, key string, ranks ...uint64) ([]float64, error) {
	return t.client.TDigestByRevRank(ctx, key, ranks...).Result()
}

// TrimmedMean returns the mean of the observations between the lo and hi
// quantiles (TDIGEST.TRIMMED_MEAN), for example 0.1 and 0.9 to ignore the
// lowest and highest 10%.
func (t *TDigest) TrimmedMean(ctx context.Context, key string, lo, hi float64) (float64, error) {
	return t.client.TDigestTrimmedMean(ctx, key, lo, hi).Result()
}
