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


-- KEYS[1]: The key to rate limit. The namespace is to avoid collision between different algorithm.
-- ARGS[1]: The request limit
-- ARGS[2]: The period of the request (in milliseconds)
-- ARGS[3]: The number of token consumed
local function fixed_window(KEYS, ARGS)
	local key = string.format("ratelimit:fixed_window:%s", KEYS[1])
	local limit = tonumber(ARGS[1])
	local period = tonumber(ARGS[2])
	local count = tonumber(ARGS[3])

	local now = now_ms()
	local data = redis.call('GET', key) or '0:0'

	local t = {}
	for d in string.gmatch(data, "(%d+)") do
		table.insert(t, tonumber(d)) -- Push to array.
	end

	local reset_at = t[1]
	local total = t[2]

	if reset_at < now then
		reset_at = now + period
		total = 0
	end

	local allow = total + count <= limit
	if allow then
		total = total + count

		redis.call('SET', key, string.format('%d:%d', reset_at, total))
		redis.call('PEXPIRE', key, period)
	end

	local remaining = limit - total
	local retry_at = remaining > 0 and now or reset_at

	return {
		allow and 1 or 0,
		remaining,
		retry_at,
		reset_at
	}
end

-- KEYS[1]: ratelimit key
-- ARGV[1]: limit
-- ARGV[2]: period in milliseconds
-- ARGV[3]: burst
-- ARGV[4]: token consumed
local function leaky_bucket(KEYS, ARGS)
	local key = string.format('ratelimit:leaky_bucket:%s', KEYS[1])

	local limit = tonumber(ARGS[1])
	local period = tonumber(ARGS[2])
	local burst = tonumber(ARGS[3])
	local count = tonumber(ARGS[4]) or 1
	local interval = period / limit

	local now = now_ms()
	local data = redis.call('GET', key) or '0:0:0'

	local t = {}
	for d in string.gmatch(data, "(%d+)") do
		table.insert(t, tonumber(d)) -- Push to array.
	end

	local reset_at = t[1]
	local total = t[2]
	local prev_batch = t[3]

	if reset_at < now then
		reset_at = now + period
		total = 0
	end

	local start = reset_at - period
	local batch_period = now - start
	local batch = math.floor(batch_period / interval)
	local batch_start = start + batch * interval
	local batch_end = batch_start + interval

	if total + count <= burst then
		total = total + count

		redis.call('SET', key, string.format('%d:%d:%d', reset_at, total, prev_batch))
		redis.call('PEXPIRE', key, period)

		return {
			1,
			limit - total,
			now,
			reset_at
		}
	end


	local allow = false
	if batch + 1 > prev_batch and total + count <= limit then
		total = math.max(total, batch)
		total = total + count
		prev_batch = batch + 1
		allow = true

		redis.call('SET', key, string.format('%d:%d:%d', reset_at, total, prev_batch))
		redis.call('PEXPIRE', key, period)
	end

	local remaining = math.max(limit - total, 0)
	local retry_at = remaining > 0 and batch_end or reset_at

	return {
		allow and 1 or 0,
		remaining,
		retry_at,
		reset_at
	}
end


redis.register_function('leaky_bucket', leaky_bucket)
redis.register_function('fixed_window', fixed_window)
