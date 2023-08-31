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

-- KEYS[1]: ratelimit key
-- ARGV[1]: limit
-- ARGV[2]: period in milliseconds
-- ARGV[3]: burst
local function gcra(KEYS, ARGS)
	local key = KEYS[1]
	local limit = tonumber(ARGS[1])
	local period = tonumber(ARGS[2]) -- in ms
	local burst = tonumber(ARGS[3])

	local interval = period / limit
	local delay_tolerance = interval * burst

	-- Theoretical now time.
	local tat = redis.call('GET', key)
	if not tat then
		tat = 0
	else
		tat = tonumber(tat)
	end


	local now = now_ms()

	tat = math.max(tat, now)

	local allow_at = tat - delay_tolerance
	local allow = now >= allow_at and 1 or 0

	if allow == 1 then
		tat = tat + interval
		redis.call('SET', key, tat, 'PX', period*2)
	end

	local remaining = (period - (tat - now))/interval
	local reset_in = tat - now
	local retry_in = math.max(allow_at - now, 0)
	-- This will be treated as redis.Nil
	-- https://cndoc.github.io/redis-doc-cn/cn/commands/eval.html
	--return false
	--return redis.error_reply("")
	return {allow, remaining, retry_in, reset_in}
end

local function sliding_window(KEYS, ARGS)
	local key = KEYS[1]
	local limit = tonumber(ARGS[1])
	local period = tonumber(ARGS[2]) -- in ms

	local curr_ms = now_ms()
	local curr_window = curr_ms - (curr_ms % period)
	local prev_window = curr_window - period

	local prev_count = tonumber(redis.call('GET', string.format('%s:%s', key, prev_window)) or tostring(limit))
	local curr_count = tonumber(redis.call('GET', string.format('%s:%s', key, curr_window)) or '0')

	local prev_ms = period - (curr_ms % period)
	local ratio = prev_ms / period
	local total = curr_count + math.floor(ratio * prev_count + 1)

	local allow = 1
	local remaining = limit - total
	local reset_in = period
	local retry_in = reset_in
	if total > limit then
		allow = 0
		remaining = 0
	end

	if allow == 1 then
		redis.call('SET', string.format('%s:%s', key, curr_window), curr_count+1, 'PX', period*2)
		retry_in = 0
	end

	return {allow, remaining, retry_in, reset_in}
end


redis.register_function('gcra', gcra)
redis.register_function('sliding_window', sliding_window)
