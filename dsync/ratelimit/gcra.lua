local key = KEYS[1]

local burst = tonumber(ARGV[1])
local interval = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local now = tonumber(ARGV[4])
local period = tonumber(ARGV[5])
local token = tonumber(ARGV[6])


local ok = 0
local ts = tonumber(redis.call('GET', key) or 0)
ts = math.max(ts, now)

if ts - burst*interval <= now then
	ok = 1
	ts = ts + token*interval
	redis.call('SET', key, ts, 'PX', period)
end


return {ok, ts}
