local key = KEYS[1]

local limit = tonumber(ARGV[1])
local period = tonumber(ARGV[2])
local burst = tonumber(ARGV[3])
local now = tonumber(ARGV[4])
local n = tonumber(ARGV[5])

-- State.
local retry_at = tonumber(redis.call('HGET', key, 'retry_at') or 0)
local count = tonumber(redis.call('HGET', key, 'count') or 0)

-- Computed.
local interval = period / limit
local expire = retry_at == 0 or count == 0
local allow = false
local checkpoint = now - (now%interval)

if checkpoint >= retry_at then
	retry_at = checkpoint + interval * n
	redis.call('HSET', key, 'retry_at', retry_at)

	allow = true
elseif count + n <= burst then
	count = count + n
	redis.call('HSET', key, 'count', count)

	allow = true
end

if expire then
	redis.call('PEXPIRE', key, period)
end

local start = (now - now%period)
local finish = start + period
local left = math.floor((finish - retry_at) / interval)
local remaining = math.max(0, burst - count) + left
if count < burst then
	retry_at = now
end

local reset_at = finish

return {
	allow and 1 or 0,
	remaining,
	retry_at,
	reset_at
}
