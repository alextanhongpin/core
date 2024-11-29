package ratelimit

import (
	"context"
	_ "embed"
	"time"

	redis "github.com/redis/go-redis/v9"
)

//go:embed gcra.lua
var gcraScript string

var gcra = redis.NewScript(gcraScript)

type GCRA struct {
	Now    func() time.Time
	burst  int
	client *redis.Client
	limit  int
	period int64
}

func NewGCRA(client *redis.Client, limit int, period time.Duration, burst int) *GCRA {
	return &GCRA{
		Now:    time.Now,
		burst:  burst,
		client: client,
		limit:  limit,
		period: period.Milliseconds(),
	}
}

func (g *GCRA) Allow(ctx context.Context, key string) (bool, error) {
	return g.AllowN(ctx, key, 1)
}

func (g *GCRA) AllowN(ctx context.Context, key string, n int) (bool, error) {
	burst := g.burst
	limit := g.limit
	now := g.Now()
	period := g.period

	interval := period / int64(limit)

	keys := []string{key}
	argv := []any{
		burst,
		interval,
		now.UnixMilli(),
		period,
		n,
	}
	ok, err := gcra.Run(ctx, g.client, keys, argv...).Int()
	if err != nil {
		return false, err
	}

	return ok == 1, nil
}
