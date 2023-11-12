package lock

import "github.com/redis/go-redis/v9"

var unlock = redis.NewScript(`
	-- KEYS[1]: The key to rate limit
	-- ARGV[1]: The request limit
	local key = KEYS[1]
	local val = ARGV[1]

	if redis.call('GET', key) == val then
		return redis.call('DEL', key)
	end

	return 0
`)

var lock = redis.NewScript(`
	-- KEYS[1]: The key to lock
	-- ARGV[1]: The random value
	-- ARGV[2]: The lock duration in milliseconds.
	local key = KEYS[1]
	local val = ARGV[1]
	local ttl_ms = tonumber(ARGV[2]) or 60000 -- Default 60s
	return redis.call('SET', key, val, 'NX', 'PX', ttl_ms)
`)

var extend = redis.NewScript(`
	-- KEYS[1]: key
	-- ARGV[1]: value
	-- ARGV[2]: lock duration in milliseconds.
	local key = KEYS[1]
	local val = ARGV[1]
	local ttl_ms = tonumber(ARGV[2]) or 60000 -- Default 60s

	if redis.call('GET', key) == val then
		return redis.call('PEXPIRE', key, ttl_ms)
	end

	return 0
`)

func parseScriptResult(unk any) error {
	if unk == nil {
		return nil
	}

	switch v := unk.(type) {
	case string:
		if v != "OK" {
			return ErrKeyNotFound
		}

		return nil
	case int64:
		if v == 0 {
			return ErrKeyNotFound
		}

		return nil
	default:
		return ErrKeyNotFound
	}
}
