local key = KEYS[1]

local burst = tonumber(ARGV[1])
local interval = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local period = tonumber(ARGV[4])
local token = tonumber(ARGV[5])


local ts = tonumber(redis.call('GET', key) or 0)
ts = math.max(ts, now)

if ts - burst*interval <= now then
	ts = ts + token*interval
	redis.call('SET', key, ts, 'PX', period)
	return 1
end

return 0
