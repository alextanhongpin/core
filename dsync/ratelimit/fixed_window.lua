local key = KEYS[1]

local limit = tonumber(ARGV[1])
local now = tonumber(ARGV[2])
local period = tonumber(ARGV[3])
local token = tonumber(ARGV[4])


local ok = 0
local count = tonumber(redis.call('GET', key) or 0)

if count + token <= limit then
	ok = 1
	count = count + token
	redis.call('SET', key, count, 'PX', period)
end

return {ok, count}
