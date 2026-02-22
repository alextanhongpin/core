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
	local remaining = -1
	ts = math.max(ts, now)

	-- Allow hitting above quantity*delta, because we can lazily evaluate the
	-- usage later.
	if ts - burst * delta <= now then
		ts = ts + quantity * delta
		local max = now + delta
		local min = ts - burst * delta
		remaining = math.max(0, math.floor((max - min) / delta))
		redis.call('SET', key, ts, 'PX', period)
	end

	local retryAfter = ts - burst * delta - now
	return {remaining, math.max(0, retryAfter)}
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
