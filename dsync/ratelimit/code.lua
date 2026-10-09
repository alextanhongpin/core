#!lua name=ratelimit
local function now_ms()
 local t=redis.call('TIME')
 return tonumber(t[1])*1000+math.floor(tonumber(t[2])/1000)
end
local function fixed_window(keys,args)
 local key=keys[1]
 local limit,period,n=tonumber(args[1]),tonumber(args[2]),tonumber(args[3])
 local count=tonumber(redis.call('GET',key) or 0)
 local ttl=redis.call('PTTL',key)
 local allowed=n<=limit-count
 if allowed and n>0 then
  count=count+n
  if ttl<0 then redis.call('SET',key,count,'PX',period);ttl=period
  else redis.call('SET',key,count,'KEEPTTL') end
 end
 if ttl<0 then ttl=period end
 local retry=0
 if not allowed then retry=ttl end
 return {allowed and 1 or 0,math.max(limit-count,0),retry,ttl}
end
local function gcra(keys,args)
 local key=keys[1]
 local burst,limit,period,n=tonumber(args[1]),tonumber(args[2]),tonumber(args[3]),tonumber(args[4])
 if n>burst+1 then return {0,0,0,0} end
 local delta=period/limit
 local now=now_ms()
 local last=math.max(tonumber(redis.call('GET',key) or now),now)
 local allowed=last-now<=(burst+1-n)*delta
 if allowed and n>0 then
  last=last+n*delta
  redis.call('SET',key,string.format('%.17g',last),'PX',math.max(math.ceil(last-now),1))
 end
 local remaining=math.max(math.floor((now+(burst+1)*delta-last)/delta+0.000001),0)
 local retry=0
 if not allowed then retry=math.max(math.ceil(last-now-(burst+1-n)*delta),0) end
 return {allowed and 1 or 0,remaining,retry,math.max(math.ceil(last-now),0)}
end
redis.register_function('rl_fixed_window',fixed_window)
redis.register_function('rl_gcra',gcra)
