#!lua name=ratelimit

-- The order matter. In lua, you cannot execute if the
-- function is not declared on top.
local function now_ms()
	local time = redis.call('TIME')
	-- The first component is the unix timestamp, in seconds.
	-- The second component is the microseconds.
	-- Convert all to milliseconds because redis PEXPIRE accepts milliseconds
	-- precision only.
	return time[1] * 1e3 + time[2] / 1e3
end


-- KEYS[1]: The key to rate limit
-- ARGS[1]: The request limit
-- ARGS[2]: The period of the request (in milliseconds)
-- ARGS[3]: The number of token consumed
local function fixed_window(KEYS, ARGS)
	local key = KEYS[1]
	local limit = tonumber(ARGS[1])
	local period = tonumber(ARGS[2])
	local count = tonumber(ARGS[3])

	local now = now_ms()
	local start = now - (now % period)

	local used = tonumber(redis.call('INCRBY', key, count))
	if used == count then
		redis.call('PEXPIRE', key, period)
	end

	-- Not allowed.
	local allow = 0
	local remaining = 0
	local retry_in = period - now%period
	local reset_in = period - now%period

	-- Allowed.
	if used <= limit then
		allow = 1
		remaining = limit - used
		retry_in = 0
	end

	return { allow, remaining, retry_in, reset_in }
end

-- KEYS[1]: ratelimit key
-- ARGV[1]: limit
-- ARGV[2]: period in milliseconds
local function sliding_window(KEYS, ARGS)
	local key = KEYS[1]
	local limit = tonumber(ARGS[1])
	local period = tonumber(ARGS[2]) -- in ms

	local now = now_ms()
	redis.call('ZREMRANGEBYSCORE', key, 0, now - period)

	local count = tonumber(redis.call('ZCARD', key))
	if count <= limit then
		redis.call('ZADD', key, now, now)
		redis.call('PEXPIRE', key, period)

		return 1
	end

	return 0
end

-- KEYS[1]: ratelimit key
-- ARGV[1]: limit
-- ARGV[2]: period in milliseconds
-- ARGV[3]: burst
-- ARGV[4]: token consumed
local function gcra(KEYS, ARGS)
	local key = KEYS[1]

	local limit = tonumber(ARGS[1])
	local period = tonumber(ARGS[2])
	local burst = tonumber(ARGS[3])
	local token = tonumber(ARGS[4]) or 1
	local interval = period / limit


	local now = now_ms()

	local window_start = now - (now % period)
	local window_end = window_start + period

	local batch = now - window_start
	local batch_start = batch - (batch % interval)
	local batch_end = batch_start + interval

	local tat = tonumber(redis.call('GET', key)) or 0
	if tat < now then
		tat = window_start + batch_start
	end

	local allow_at = tat - interval * burst
	local allow = now >= allow_at
	if allow then
		tat = tat + interval * token
		redis.call('SET', key, tat, 'PX', period)
	end

	local retry_in = batch_end - batch
	local reset_in = window_end - now
  local remaining = math.max(limit - (tat - window_start)/interval + burst, 0)

	if burst > 0 and remaining > burst then
		retry_in = 0
	end

	if remaining == 0 then
		retry_in = reset_in
	end

	return {
		allow and 1 or 0,
		remaining,
		retry_in,
		reset_in
	}
end


redis.register_function('gcra', gcra)
redis.register_function('sliding_window', sliding_window)
redis.register_function('fixed_window', fixed_window)
