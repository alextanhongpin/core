#!lua name=ratelimit

local function now_ms()
	local time = redis.call('TIME')
	local seconds = time[1]
	local microseconds = time[2]
	return seconds * 1000 + microseconds/1000 -- in milliseconds
end

local function gcra(keys, args)
	local key = keys[1]
	local burst = tonumber(args[1])
	local limit = tonumber(args[2])
	local period = tonumber(args[3])
	local quantity = tonumber(args[4])

	local delta = period / limit;
	local now = now_ms()
	local ts = tonumber(redis.call('GET', key) or 0)
	local allow = 0
	ts = math.max(ts, now)

	-- Allow hitting above quantity*delta, because we can lazily evaluate the
	-- usage later.
	if ts - burst * delta <= now then
		allow = 1
		ts = ts + quantity * delta
		redis.call('SET', key, ts, 'PX', period)
	end

	-- Retry After.
	return {allow, (ts - burst * delta) - now}
end

redis.register_function('gcra', gcra)

local function fixed_window(keys, args)
	local key = keys[1]

	local limit = tonumber(args[1])
	local period = tonumber(args[2])
	local quantity = tonumber(args[3])

	local count = tonumber(redis.call('INCRBY', key, quantity))
	if count == quantity then
		redis.call('PEXPIRE', key, period)
	end

	if count <= limit then
		return limit - count
	end

	return -1
end

redis.register_function('fixed_window', fixed_window)
