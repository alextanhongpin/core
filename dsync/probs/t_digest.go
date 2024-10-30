package probs

import (
	"context"

	redis "github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

// Use this to track latency of the server for sql operations, api requests etc.
// We use this together with top-k to see the top performing api requests.
type TDigest struct {
	Client *redis.Client
	group  singleflight.Group
}

func NewTDigest(client *redis.Client) *TDigest {
	return &TDigest{
		Client: client,
	}
}

// Create needs to be called.
func (t *TDigest) CreateWithCompression(ctx context.Context, key string, compression int64) (string, error) {
	status, err := t.Client.TDigestCreateWithCompression(ctx, key, compression).Result()
	if TDigestKeyAlreadyExistsError(err) {
		return "OK", nil
	}

	return status, err
}

func (t *TDigest) Create(ctx context.Context, key string) (string, error) {
	status, err := t.Client.TDigestCreate(ctx, key).Result()
	if TDigestKeyAlreadyExistsError(err) {
		return "OK", nil
	}

	return status, err
}

func (t *TDigest) Add(ctx context.Context, key string, values ...float64) (string, error) {
	status, err := t.Client.TDigestAdd(ctx, key, values...).Result()
	if err == nil {
		return status, nil
	}

	if create := TDigestKeyDoesNotExistsError(err); !create {
		return "", err
	}

	_, err, shared := t.group.Do(key, func() (any, error) {
		return t.Create(ctx, key)
	})
	if err != nil {
		return "", err
	}

	if !shared {
		// Clear key after created.
		t.group.Forget(key)
	}

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

func TDigestKeyAlreadyExistsError(err error) bool {
	return redis.HasErrorPrefix(err, "T-Digest: key already exists")
}

func TDigestKeyDoesNotExistsError(err error) bool {
	return redis.HasErrorPrefix(err, "T-Digest: key does not exist")
}
