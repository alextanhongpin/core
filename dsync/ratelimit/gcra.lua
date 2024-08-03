local key = KEYS[1]

local burst = tonumber(ARGV[1])
local interval = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local now = tonumber(ARGV[4])
local period = tonumber(ARGV[5])
local token = tonumber(ARGV[6])


local ok = 0
local ts = tonumber(redis.call('GET', key) or 0)
local lo = now - (now % interval) - (burst + token) * interval
if ts < lo then
	ts = lo
	redis.call('SET', key, ts, 'PX', period)
end

if ts + token*interval <= now then
	ok = 1
	ts = ts + token*interval
	redis.call('SET', key, ts)
end


return {ok, ts}
