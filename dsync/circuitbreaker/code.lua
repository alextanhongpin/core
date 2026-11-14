#!lua name=circuitbreaker
-- Circuit breaker functions. Installed by circuitbreaker.Setup.
--
-- Compatibility: function names, argument order, return values and the hash
-- layout (fields status, generation, counter, timeout) are a contract with
-- every deployed version of the Go package. Only change them additively.
--
-- Each breaker is one hash:
--   status      CLOSED/HALF_OPEN/OPENED/DISABLED/FORCED_OPEN (missing = CLOSED)
--   generation  incremented on every transition; results from calls admitted
--               in an older generation are ignored
--   counter     failure weight (CLOSED) or successes (HALF_OPEN); the field
--               expires after the failure/success period without updates
--   timeout     Redis time in ms when an OPENED breaker may go HALF_OPEN

local CLOSED, HALF_OPEN, OPENED, DISABLED, FORCED_OPEN = 1, 2, 3, 4, 5

-- get the current time in milliseconds
local function now_ms()
	local t = redis.call("TIME")
	local seconds = tonumber(t[1])
	local microseconds = tonumber(t[2])
	return seconds * 1000 + math.floor(microseconds / 1000)
end

-- transition the circuit breaker to a new status, incrementing the generation
-- and setting the timeout if necessary
local function transition(key, status, timeout)
	redis.call("HINCRBY", key, "generation", 1)
	redis.call("HSET", key, "status", status)
	redis.call("HDEL", key, "counter", "timeout")

	if status == OPENED then
		redis.call("HSET", key, "timeout", now_ms() + timeout)
	end

	return status
end

-- begin a circuit breaker transaction, returning the current status and
-- generation
local function begin(keys, args)
	local key = keys[1]
	redis.call("HSETNX", key, "generation", 0)
	local result = redis.call("HMGET", key, "status", "timeout")
	local status = tonumber(result[1] or CLOSED)
	local timeout = tonumber(result[2] or 0)

	if status == OPENED and now_ms() >= timeout then
		status = transition(key, HALF_OPEN, 0)
	end

	local generation = tonumber(redis.call("HGET", key, "generation") or 0)
	return { status, generation }
end

-- commit a circuit breaker transaction, updating the counter and potentially
local function commit(keys, args)
	local key = keys[1]
	local result = redis.call("HMGET", key, "status", "generation")
	local status = tonumber(result[1] or CLOSED)
	local generation = tonumber(result[2] or 0)
	if generation ~= tonumber(args[8]) then
		return status
	end
	local failure = tonumber(args[1])
	local failure_threshold = tonumber(args[2])
	local failure_ttl = tonumber(args[3])
	local success = tonumber(args[4])
	local success_threshold = tonumber(args[5])
	local success_ttl = tonumber(args[6])
	local timeout = tonumber(args[7])

	if status ~= CLOSED and status ~= HALF_OPEN then
		return status
	end

	if status == HALF_OPEN and failure > 0 then
		return transition(key, OPENED, timeout)
	end

	local increment, threshold, ttl
	if status == CLOSED then
		if failure == 0 then
			return status
		end
		increment, threshold, ttl = failure, failure_threshold, failure_ttl
	else
		increment, threshold, ttl = success, success_threshold, success_ttl
	end

	local total = redis.call("HINCRBY", key, "counter", increment)
	redis.call("HPEXPIRE", key, ttl, "FIELDS", 1, "counter")
	if total >= threshold then
		if status == CLOSED then
			return transition(key, OPENED, timeout)
		end
		return transition(key, CLOSED, 0)
	end
	return status
end

-- set the circuit breaker status directly, incrementing the generation and
local function set_status(keys, args)
	local key = keys[1]
	local status = tonumber(args[1])
	local timeout = tonumber(args[2] or 0)
	return transition(key, status, timeout)
end

redis.register_function("cb_begin", begin)
redis.register_function("cb_commit", commit)
redis.register_function("cb_set_status", set_status)
