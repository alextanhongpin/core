package circuitbreaker

import "github.com/redis/go-redis/v9"

var script = redis.NewScript(`
	-- KEYS[1]: The key to rate limit
	-- KEYS[2]: The old status for optimistic locking
	-- KEYS[3]: The ttl duration in milliseconds.
	local key = KEYS[1]
	local status = KEYS[2]
	local ttl_ms = tonumber(KEYS[3]) or 60000 -- Default 60s

	-- This returns bool if not exists.
	local old_status = redis.call('HGET', key, 'status')

	-- Update only if 
	-- 1) it is not set
	-- 2) the status is the same
	if not old_status or old_status == status then
		redis.call('HSET', key, unpack(ARGV))
		return redis.call('PEXPIRE', key, ttl_ms)
	end

	-- This will return redis nil.
	return false
`)
