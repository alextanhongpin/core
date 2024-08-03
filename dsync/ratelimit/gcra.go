package ratelimit

import (
	"context"
	_ "embed"
	"math"
	"time"

	redis "github.com/redis/go-redis/v9"
)

//go:embed gcra.lua
var gcraScript string

var gcra = redis.NewScript(gcraScript)

type GCRAOption struct {
	Burst  int
	Limit  int
	Period time.Duration
}

type GCRA struct {
	Now    func() time.Time
	client *redis.Client
	opt    *GCRAOption
}

func NewGCRA(client *redis.Client, opt *GCRAOption) *GCRA {
	return &GCRA{
		Now:    time.Now,
		client: client,
		opt:    opt,
	}
}

func (g *GCRA) Allow(ctx context.Context, key string) (*Result, error) {
	return g.AllowN(ctx, key, 1)
}

func (g *GCRA) AllowN(ctx context.Context, key string, n int) (*Result, error) {
	period := g.opt.Period
	limit := g.opt.Limit
	burst := g.opt.Burst

	interval := period / time.Duration(limit)
	now := g.Now()

	keys := []string{key}
	argv := []any{
		burst,
		interval.Milliseconds(),
		limit,
		now.UnixMilli(),
		period.Milliseconds(),
		n,
	}
	res, err := gcra.Run(ctx, g.client, keys, argv...).Int64Slice()
	if err != nil {
		return nil, err
	}
	allow := res[0] == 1
	ts := time.UnixMilli(res[1])

	resetAt := now.Truncate(period).Add(period)
	remaining := int64(math.Floor(float64(resetAt.Sub(now)) / float64(interval)))

	return &Result{
		Allow:     allow,
		Limit:     int64(limit + burst),
		Remaining: remaining,
		ResetAt:   resetAt,
		RetryAt:   ts,
	}, nil
}
