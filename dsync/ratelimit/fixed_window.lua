local key = KEYS[1]

local limit = tonumber(ARGV[1])
local period = tonumber(ARGV[2])
local token = tonumber(ARGV[3])

local count = tonumber(redis.call('GET', key) or 0)

if count + token <= limit then
	redis.call('SET', key, count + token, 'PX', period)
	return 1
end

return 0
