package probs

import (
	"context"
	"sync"

	redis "github.com/redis/go-redis/v9"
)

type TDigestKeyCreator struct {
	*TDigest
	Compression int64
	keys        map[string]struct{}
	mu          sync.RWMutex
}

func NewTDigestKeyCreator(client *redis.Client, compression int64) *TDigestKeyCreator {
	return &TDigestKeyCreator{
		TDigest:     NewTDigest(client),
		Compression: compression,
		keys:        make(map[string]struct{}),
	}
}

// Add automatically creates the key if it doesn't exist.
func (t *TDigestKeyCreator) Add(ctx context.Context, key string, values ...float64) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	_, ok := t.keys[key]
	if ok {
		return t.TDigest.Add(ctx, key, values...)
	}

	_, err := t.CreateWithCompression(ctx, key, t.Compression)
	if err != nil {
		// ERR T-Digest: key already exists
		if !ErrTDigestKeyExists(err) {
			return "", err
		}
	}
	t.keys[key] = struct{}{}

	return t.TDigest.Add(ctx, key, values...)
}

func (t *TDigestKeyCreator) Keys() []any {
	t.mu.RLock()
	res := make([]any, 0, len(t.keys))
	for k := range t.keys {
		res = append(res, k)
	}
	t.mu.RUnlock()

	return res
}
