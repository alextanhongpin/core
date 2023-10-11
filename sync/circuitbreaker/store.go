package circuitbreaker

import (
	"context"

	"log/slog"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	opt *RedisStoreOption
}

type RedisStoreOption struct {
	Key    string
	Client *redis.Client
	Logger *slog.Logger
}

func NewRedisStore(opt *RedisStoreOption) *RedisStore {
	return &RedisStore{
		opt: opt,
	}
}

func (s *RedisStore) Get() (Status, bool) {
	opt := s.opt

	ctx := context.Background()
	state, err := opt.Client.Get(ctx, opt.Key).Int64()
	if err != nil {
		if opt.Logger != nil {
			opt.Logger.Error("failed to get circuitbreaker state",
				slog.String("key", opt.Key),
				slog.String("err", err.Error()),
			)
		}

		return 0, false
	}

	status := Status(state)
	return status, status.IsValid()
}

func (s *RedisStore) Set(status Status) {
	opt := s.opt

	ctx := context.Background()
	err := opt.Client.Set(ctx, opt.Key, status.Int64(), 0).Err()
	if err != nil && opt.Logger != nil {
		opt.Logger.Error("failed to set circuitbreaker state",
			slog.String("key", opt.Key),
			slog.String("err", err.Error()),
		)
	}
}
