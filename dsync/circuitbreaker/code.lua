#!lua name=circuitbreaker
local CLOSED, HALF_OPEN, OPENED, DISABLED, FORCED_OPEN = 1, 2, 3, 4, 5
local function now_ms()
 local t=redis.call('TIME')
 return tonumber(t[1])*1000+tonumber(t[2])/1000
end
local function transition(key,status,timeout)
 redis.call('HINCRBY',key,'generation',1)
 redis.call('HSET',key,'status',status)
 redis.call('HDEL',key,'counter','timeout')
 if status==OPENED then redis.call('HSET',key,'timeout',now_ms()+timeout) end
 return status
end
local function begin(keys,args)
 local key=keys[1]
 redis.call('HSETNX',key,'generation',0)
 local status=tonumber(redis.call('HGET',key,'status') or CLOSED)
 local timeout=tonumber(redis.call('HGET',key,'timeout') or 0)
 if status==OPENED and now_ms()>=timeout then status=transition(key,HALF_OPEN,0) end
 return {status,tonumber(redis.call('HGET',key,'generation'))}
end
local function commit(keys,args)
 local key=keys[1]
 local status=tonumber(redis.call('HGET',key,'status') or CLOSED)
 local generation=tonumber(redis.call('HGET',key,'generation') or 0)
 if generation~=tonumber(args[8]) then return status end
 local failure=tonumber(args[1])
 local success=tonumber(args[4])
 if status~=CLOSED and status~=HALF_OPEN then return status end
 if status==HALF_OPEN and failure>0 then return transition(key,OPENED,tonumber(args[7])) end
 local increment,threshold,ttl
 if status==CLOSED then
  if failure==0 then return status end
  increment,threshold,ttl=failure,tonumber(args[2]),tonumber(args[3])
 else increment,threshold,ttl=success,tonumber(args[5]),tonumber(args[6]) end
 local total=redis.call('HINCRBY',key,'counter',increment)
 redis.call('HPEXPIRE',key,ttl,'FIELDS',1,'counter')
 if total>=threshold then
  if status==CLOSED then return transition(key,OPENED,tonumber(args[7])) end
  return transition(key,CLOSED,0)
 end
 return status
end
local function set_status(keys,args)
 return transition(keys[1],tonumber(args[1]),tonumber(args[2]))
end
redis.register_function('cb_begin',begin)
redis.register_function('cb_commit',commit)
redis.register_function('cb_set_status',set_status)
