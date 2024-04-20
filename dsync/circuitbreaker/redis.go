package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var script = redis.NewScript(`
	-- KEYS[1]: The key to rate limit
	-- KEYS[2]: The old status for optimistic locking
	local key = KEYS[1]
	local status = KEYS[2]
	local new_status = ARGV[2]

	-- This returns bool if not exists.
	local old_status = redis.call('HGET', key, 'status')

	-- Update only if 
	-- 1) it is not set
	-- 2) the status is the same
	if not old_status or old_status == status then
		if old_status == new_status then
			for k = 3, #KEYS do
				local event = KEYS[k]
				if event == 'total:reset' then
					redis.call('HSET', key, 'total', 0)
				elseif event == 'total:inc' then
					redis.call('HINCRBY', key, 'total', 1)
				elseif event == 'count:reset' then
					redis.call('HSET', key, 'count', 0)
				elseif event == 'count:inc' then
					redis.call('HINCRBY', key, 'count', 1)
				elseif event == 'count:dec' then
					redis.call('HINCRBY', key, 'count', -1)
				elseif event == 'reset_at:set' then
					redis.call('HSET', key, 'resetAt', ARGV[8])
				end
			end
		else
			-- overwrite.
			redis.call('HSET', key, unpack(ARGV))
		end

		return 1
	end

	-- This will return redis nil.
	return false
`)

var ErrConcurrentWrite = errors.New("circuitbreaker: concurrent write")

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{
		client: client,
	}
}

var _ store = (*RedisStore)(nil)

func (r *RedisStore) Get(ctx context.Context, key string) (*State, error) {
	m, err := r.client.HGetAll(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		// Null object.
		return &State{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return &State{}, nil
	}

	status, err := strconv.Atoi(m["status"])
	if err != nil {
		return nil, err
	}

	count, err := strconv.Atoi(m["count"])
	if err != nil {
		return nil, err
	}

	total, err := strconv.Atoi(m["total"])
	if err != nil {
		return nil, err
	}

	resetAt, err := strconv.ParseInt(m["resetAt"], 10, 64)
	if err != nil {
		return nil, err
	}

	closeAt, err := strconv.ParseInt(m["closeAt"], 10, 64)
	if err != nil {
		return nil, err
	}

	return &State{
		status:  Status(status),
		Status:  Status(status),
		Count:   count,
		Total:   total,
		ResetAt: time.UnixMilli(resetAt),
		CloseAt: time.UnixMilli(closeAt),
	}, nil
}

func (r *RedisStore) Set(ctx context.Context, key string, res *State) error {
	keys := append([]string{
		key,
		strconv.Itoa(int(res.status)),
	}, res.events...)
	argv := []any{
		"status", strconv.Itoa(int(res.Status)),
		"count", strconv.Itoa(res.Count),
		"total", strconv.Itoa(res.Total),
		"resetAt", fmt.Sprint(res.ResetAt.UnixMilli()),
		"closeAt", fmt.Sprint(res.CloseAt.UnixMilli()),
	}
	err := script.Run(ctx, r.client, keys, argv...).Err()
	if errors.Is(err, redis.Nil) {
		return ErrConcurrentWrite
	}

	return err
}

// copied from redis source code
func formatMs(dur time.Duration) int64 {
	if dur > 0 && dur < time.Millisecond {
		return 1
	}

	return int64(dur / time.Millisecond)
}
