#!lua name=lock

-- KEYS[1]: The key to lock
-- ARGS[1]: The random value
-- ARGS[2]: The lock duration in seconds.
local function lock(KEYS, ARGS)
	local key = KEYS[1]
	local val = ARGS[1]
	local lock_duration_in_seconds = tonumber(ARGS[2]) or 60
	return redis.call('SET', key, val, 'NX', 'EX', lock_duration_in_seconds)
end

-- KEYS[1]: The key to rate limit
-- ARGS[1]: The request limit
local function unlock(KEYS, ARGS)
	local key = KEYS[1]
	local val = ARGS[1]

	if redis.call('GET', key) == val then
		return redis.call('DEL', key)
	end

	return 0
end

-- KEYS[1]: key
-- ARGV[1]: value
-- ARGV[2]: lock duration in seconds
local function refresh(KEYS, ARGS)
	local key = KEYS[1]
	local val = ARGS[1]
	local lock_duration_in_seconds = tonumber(ARGS[2]) or 60

	if redis.call('GET', key) == val then
		return redis.call('EXPIRE', key, lock_duration_in_seconds)
	end

	return 0
end


redis.register_function('lock', lock)
redis.register_function('unlock', unlock)
redis.register_function('refresh', refresh)
