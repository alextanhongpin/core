#!lua name=circuitbreaker

local UNKNOWN = 0
local CLOSED = 1
local HALF_OPEN = 2
local OPENED = 3
local DISABLED = 4
local FORCED_OPEN = 5

local function now_ms()
	local time = redis.pcall('TIME')
	local seconds = time[1]
	local microseconds = time[2]
	return seconds * 1000 + microseconds/1000 -- in milliseconds
end

local function get_status(keys)
	return tonumber(redis.pcall('HGET', keys[1], 'status') or CLOSED)
end

local function on_half_opened(key)
		redis.pcall('HMSET', key, 'status', HALF_OPEN, 'counter', 0)
		return HALF_OPEN
end

local function on_closed(key)
		redis.pcall('HMSET', key, 'status', CLOSED, 'counter', 0)
		return CLOSED
end

local function on_opened(key, timeout_after)
		redis.pcall('HMSET', key, 'status', OPENED, 'timeout', timeout_after)
		return OPENED
end

local function inc(key, value, ttl)
	local total = tonumber(redis.pcall('HINCRBY', key, 'counter', value))
	if total == value then
		redis.pcall('HPEXPIRE', key, ttl, 'FIELDS', 'counter')
	end

	return total
end

local function close(keys, args)
	local key = keys[1]

	local failure_count = tonumber(args[1])
	local failure_threshold = tonumber(args[2])
	local failure_period = tonumber(args[3])
	local success_count = tonumber(args[4])
	local success_threshold = tonumber(args[5])
	local success_period = tonumber(args[6])
	local open_timeout = tonumber(args[7])

	-- if success
	if failure_count == 0 then
		return CLOSED
	end

	-- increment failure counter
	local total_count = inc(key, failure_count, failure_period)

  -- if failure threshold exceeded
	if total_count >= failure_threshold then
		return on_opened(key, now_ms() + open_timeout)
	end

	return CLOSED
end


local function halfOpen(keys, args)
	local key = keys[1]

	local failure_count = tonumber(args[1])
	local failure_threshold = tonumber(args[2])
	local failure_period = tonumber(args[3])
	local success_count = tonumber(args[4])
	local success_threshold = tonumber(args[5])
	local success_period = tonumber(args[6])
	local open_timeout = tonumber(args[7])

	-- if success
	if failure_count == 0 then
		-- increment success counter
		local total_count = inc(key, success_count, success_period)

		-- if success count threshold reached
		if total_count > success_threshold then
			return on_closed(key)
		end

		return HALF_OPEN
	end

	local timeout_after = now_ms() + open_timeout
	return on_opened(key, timeout_after)
end


local function begin(keys, args)
	local key = keys[1]
	local result = redis.pcall('HMGET', key, 'status', 'timeout')
	local status = tonumber(result[1] or CLOSED)
	local timeout = tonumber(result[2] or 0)

	-- if timeout timer expired
	if status == OPENED and now_ms() >= timeout then
		return on_half_opened(key)
	end

	return status
end


local function commit(keys, args)
	local status = get_status(keys)

	if status == CLOSED then
		return close(keys, args)
	elseif status == HALF_OPEN then
		return halfOpen(keys, args)
	else
		-- Not possible
		return UNKNOWN
	end
end


redis.register_function('begin', begin)
redis.register_function('commit', commit)
