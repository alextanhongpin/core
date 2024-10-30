package probs

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

// Use this to track latency of the server for sql operations, api requests etc.
// We use this together with top-k to see the top performing api requests.
type TDigest struct {
	Client *redis.Client
}

func NewTDigest(client *redis.Client) *TDigest {
	return &TDigest{
		Client: client,
	}
}

// Create needs to be called.
func (t *TDigest) CreateWithCompression(ctx context.Context, key string, compression int64) (string, error) {
	return t.Client.TDigestCreateWithCompression(ctx, key, compression).Result()
}

func (t *TDigest) Create(ctx context.Context, key string) (string, error) {
	return t.Client.TDigestCreate(ctx, key).Result()
}

func (t *TDigest) Add(ctx context.Context, key string, values ...float64) (string, error) {
	return t.Client.TDigestAdd(ctx, key, values...).Result()
}

func (t *TDigest) CDF(ctx context.Context, key string, values ...float64) ([]float64, error) {
	return t.Client.TDigestCDF(ctx, key, values...).Result()
}

func (t *TDigest) Quantile(ctx context.Context, key string, values ...float64) ([]float64, error) {
	return t.Client.TDigestQuantile(ctx, key, values...).Result()
}

func (t *TDigest) Min(ctx context.Context, key string) (float64, error) {
	return t.Client.TDigestMin(ctx, key).Result()
}

func (t *TDigest) Max(ctx context.Context, key string) (float64, error) {
	return t.Client.TDigestMax(ctx, key).Result()
}

func (t *TDigest) Rank(ctx context.Context, key string, values ...float64) ([]int64, error) {
	return t.Client.TDigestRank(ctx, key, values...).Result()
}

func (t *TDigest) RevRank(ctx context.Context, key string, values ...float64) ([]int64, error) {
	return t.Client.TDigestRevRank(ctx, key, values...).Result()
}

func (t *TDigest) ByRank(ctx context.Context, key string, values ...uint64) ([]float64, error) {
	return t.Client.TDigestByRank(ctx, key, values...).Result()
}

func (t *TDigest) ByRevRank(ctx context.Context, key string, values ...uint64) ([]float64, error) {
	return t.Client.TDigestByRevRank(ctx, key, values...).Result()
}

func (t *TDigest) TrimmedMean(ctx context.Context, key string, lo, hi float64) (float64, error) {
	return t.Client.TDigestTrimmedMean(ctx, key, lo, hi).Result()
}

func ErrTDigestKeyExists(err error) bool {
	return redis.HasErrorPrefix(err, "T-Digest: key already exists")
}
