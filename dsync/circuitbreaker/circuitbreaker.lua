-- The script is used to implement the circuit breaker pattern in Redis.
-- This script is faster to execute based on benchmark compared to running
-- the logic in golang code, and it is less prone to race conditions too,
-- especially when writing and resetting values.
local key = KEYS[1]
local now_ns = tonumber(ARGV[1])
local period_ns = tonumber(ARGV[2])
local failure_threshold = tonumber(ARGV[3])
local failure_percentage = tonumber(ARGV[4])
local success_threshold = tonumber(ARGV[5])
local recover_duration_ns = tonumber(ARGV[6])
local op = tonumber(ARGV[7] or 0)

local CLOSED = 0
local OPEN = 1
local HALF_OPEN = 2
local ISOLATED = 3

local NOOP = 0
local FAILURE = 1
local SUCCESS = 2


-- expand converts a table into key-value list.
local function expand(t)
    local res = {}
    for k, v in pairs(t) do
        res[#res + 1] = k
        res[#res + 1] = v
    end
    return res
end

-- retrieve the hash table values and converts the list to table.
local function hgetall(key)
	local table = redis.call('HGETALL', key)
	local state = {}

	-- initial status is closed.
	if #table == 0 then
		state['status'] = CLOSED
		state['reset_at'] = now_ns + period_ns
		state['update'] = true
		return state
	end

	-- set the list values to table.
	for i = 1, #table, 2 do
			state[table[i]] = table[i + 1]
	end

	return state
end

-- set the hash table values.
local function hsetall(state)
	if state['update'] then
		state['update'] = nil
		redis.call('HSET', key, unpack(expand(state)))
	end
end

-- eval checks if the status can be transitioned.
local function eval(state)
	local status = tonumber(state['status'])
	if status == CLOSED then
		local reset_at = tonumber(state['reset_at'])
		if now_ns >= reset_at then
			state['reset_at'] = now_ns + period_ns
			state['count'] = 0
			state['total'] = 0
			state['update'] = true
		end

		local count = tonumber(state['count'] or 0)
		local total = tonumber(state['total'] or 0)

		local is_failure_threshold_exceeded = count >= failure_threshold
		local is_failure_percentage_exceeded = count / total >= failure_percentage

		if is_failure_threshold_exceeded and is_failure_percentage_exceeded then
			state['update'] = true
			return {OPEN, true}
		end
	end

	if status == OPEN then
		local recover_at = tonumber(state['recover_at'])
		if now_ns >= recover_at then
			state['update'] = true
			return {HALF_OPEN, true}
		end
	end

	if status == HALF_OPEN then
		local count = tonumber(state['count'] or 0)
		if count >= success_threshold then
			state['update'] = true
			return {CLOSED, true}
		end
	end

	return {status, false}
end

-- entry returns the new state when entering a new status.
local function entry(status)
	local state = {}
	state['status'] = status
	state['count'] = 0
	state['total'] = 0
	state['reset_at'] = nil
	state['recover_at'] = nil
	state['update'] = true

	if status == OPEN then
		state['recover_at'] = now_ns + recover_duration_ns
	end

	if status == CLOSED then
		state['reset_at'] = now_ns + period_ns
	end

	return state
end

-- process updates the state based on the success/failure.
local function process(state)
	local status = tonumber(state['status'])
	local count = tonumber(state['count'] or 0)
	local total = tonumber(state['total'] or 0)

	if status == HALF_OPEN then
		if op == SUCCESS then
			state['count'] = count + 1
			state['update'] = true
		end

		if op == FAILURE then
			state = {}
			state['count'] = 0
			state['status'] = OPEN
			state['recover_at'] = now_ns + recover_duration_ns
			state['update'] = true
		end
	end

	if status == CLOSED then
		if op == FAILURE then
			state['count'] = count + 1
		end

		state['total'] = total + 1
		state['update'] = true
	end

	return state
end

-- transition returns the new state if the status changed, otherwise returns
-- the old state.
local function transition(state)
	local result = eval(state)
	if result[2] then return entry(result[1]) else return state end
end

-- main flow
-- 1. get the last state
local state = hgetall(key)
local prev_status = state['status']

-- 2. transition the state
state = transition(state)
local next_status = state['status']

-- 3. process the state
if op == NOOP then
	hsetall(state)
	return {prev_status, next_status}
else
	local state = process(state)
	next_status = state['status']

	hsetall(state)
	return {prev_status, next_status}
end
