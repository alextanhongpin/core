#!lua name=ratelimit
-- Rate limiter functions. Installed by ratelimit.Setup.
--
-- Compatibility: function names, argument order and the shape of the return
-- value are a contract with every deployed version of the Go package. Only
-- change them by adding new functions.
--
-- Both functions return {allowed (0/1), remaining, retry_after_ms, reset_after_ms}.
-- Lua numbers are doubles; Go validates that every input stays below 2^53 so
-- all integer arithmetic here is exact. Numbers returned to Redis are
-- truncated to integers.

-- Current Redis server time in milliseconds.
local function now_ms()
	local t = redis.call('TIME')
	return tonumber(t[1]) * 1000 + math.floor(tonumber(t[2]) / 1000)
end

-- KEYS[1] counter key
-- ARGV[1] limit, ARGV[2] period in ms, ARGV[3] n (0 = inspect only)
local function fixed_window(keys, args)
	local key = keys[1]
	local limit, period, n = tonumber(args[1]), tonumber(args[2]), tonumber(args[3])
	local count = tonumber(redis.call('GET', key) or 0)
	local ttl = redis.call('PTTL', key) -- -2: missing, -1: no expiry
	local allowed = n <= limit - count
	if allowed and n > 0 then
		count = redis.call('INCRBY', key, args[3])
		if ttl < 0 then
			-- First request of a new window (or a key without expiry, which
			-- would otherwise never reset).
			redis.call('PEXPIRE', key, args[2])
			ttl = period
		end
	end
	if ttl < 0 then
		ttl = period
	end
	local retry = 0
	if not allowed then
		retry = ttl
	end
	return { allowed and 1 or 0, math.max(limit - count, 0), retry, ttl }
end

-- KEYS[1] key holding the theoretical arrival time (TAT) in ms
-- ARGV[1] burst, ARGV[2] limit, ARGV[3] period in ms, ARGV[4] n (0 = inspect only)
local function gcra(keys, args)
	local key = keys[1]
	local burst, limit, period, n = tonumber(args[1]), tonumber(args[2]), tonumber(args[3]), tonumber(args[4])
	local capacity = burst + 1
	local interval = period / limit -- time "cost" of one request
	local now = now_ms()
	local tat = math.max(tonumber(redis.call('GET', key) or now), now)
	local fits = n <= capacity
	local allowed = fits and tat - now <= (capacity - n) * interval
	if allowed and n > 0 then
		tat = tat + n * interval
		redis.call('SET', key, string.format('%.17g', tat), 'PX', math.max(math.ceil(tat - now), 1))
	end
	local remaining = math.max(math.floor((now + capacity * interval - tat) / interval + 0.000001), 0)
	local retry = 0
	if fits and not allowed then
		retry = math.max(math.ceil(tat - now - (capacity - n) * interval), 0)
	end
	return { allowed and 1 or 0, remaining, retry, math.max(math.ceil(tat - now), 0) }
end

redis.register_function('rl_fixed_window', fixed_window)
redis.register_function('rl_gcra', gcra)
