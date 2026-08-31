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

	local delta = period / limit
	local now = now_ms()

	local lo = math.floor(now - (burst * delta))
	local up = math.floor(now) + 1
	local inc = quantity * delta

	local result = redis.call('INCREX', key, 'BYINT', inc, 'LBOUND', lo, 'UBOUND', up, 'PX', period)
	if result[1] + inc <= lo and result[2] == 0 then
		return redis.call('INCREX', key, 'BYINT', inc, 'LBOUND', lo, 'UBOUND', up, 'SATURATE', 'PX', period)
	end

	return result
end

redis.register_function('gcra', gcra)
