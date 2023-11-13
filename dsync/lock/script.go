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

	return nil
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

	return nil
`)
