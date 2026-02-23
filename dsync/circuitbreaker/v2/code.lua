#!lua name=circuitbreaker

local OPENED = 1 -- We store the status as +int, indicating the timeout after in unix.
local HALF_OPEN = 0
local CLOSED = -1
local DISABLED = -3
local FORCED_OPEN = -2
local UNKNOWN = -99


local function now_ms()
	local time = redis.pcall('TIME')
	local seconds = time[1]
	local microseconds = time[2]
	return seconds * 1000 + microseconds/1000 -- in milliseconds
end

local function get_status(keys)
	return tonumber(redis.pcall('HGET', keys[1], 'status') or CLOSED)
end

local function set_status(key, value)
	return redis.pcall('HSET', key, 'status', value)
end

local function transition(key, status, timeout_after)
	if status == HALF_OPEN then
		set_status(key, status)
		-- reset success counter
		redis.pcall('HSET', key, 'success', 0)
	elseif status == CLOSED then
		set_status(key, status)
		-- reset failure counter
		redis.pcall('HSET', key, 'failure', 0)
	else
		-- opened
    -- start timeout timer
		set_status(key, timeout_after)
	end

	return status
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
	local total_count = tonumber(redis.pcall('HINCRBY', key, 'failure', failure_count))
	if total_count == failure_count then
		redis.pcall('HPEXPIRE', key, failure_period, 'FIELDS', 'failure')
	end

  -- if failure threshold exceeded
	if total_count >= failure_threshold then
		local timeout_after = now_ms() + open_timeout
		return transition(key, OPENED, timeout_after)
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
		local total_count = tonumber(redis.pcall('HINCRBY', key, 'success', success_count))
		if total_count == success_count then
			redis.pcall('HPEXPIRE', key, success_period, 'FIELDS', 'success')
		end

		-- if success count threshold reached
		if total_count > success_threshold then
				return transition(key, CLOSED, 0)
		end

		return HALF_OPEN
	end

	local timeout_after = now_ms() + open_timeout
	return transition(key, OPENED, timeout_after)
end


local function begin(keys, args)
	local status = get_status(keys)

	if status >= OPENED then
		-- if timeout timer expired
		if now_ms() >= status then
			transition(keys[1], HALF_OPEN, 0)
			return HALF_OPEN
		end

		return OPENED
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
