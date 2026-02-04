#!lua name=ratelimit

local function now_ms()
	local time = redis.call('TIME')
	local seconds = time[1]
	local microseconds = time[2]
	return seconds * 1000 + microseconds/1000 -- in milliseconds
end

local function gcra(keys, args)
	local key = keys[1]
	local increment = tonumber(args[1])
	local offset = tonumber(args[2])
	local period = tonumber(args[3])
	local token = tonumber(args[4])

	local now = now_ms()
	local ts = tonumber(redis.call('GET', key) or 0)
	ts = math.max(ts, now)

	if ts - offset <= now then
		ts = ts + token * increment
		redis.call('SET', key, ts, 'PX', period)
		return 1
	end

	return 0
end

redis.register_function('gcra', gcra)

local function fixed_window(keys, args)
	local key = keys[1]

	local limit = tonumber(args[1])
	local period = tonumber(args[2])
	local token = tonumber(args[3])

	local count = tonumber(redis.call('GET', key) or 0)

	if count + token <= limit then
		redis.call('SET', key, count + token, 'PX', period)
		return 1
	end

	return 0
end

redis.register_function('fixed_window', fixed_window)
