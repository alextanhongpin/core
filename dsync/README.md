# dsync

Building blocks for coordinating work **across processes and replicas** using
Redis: locks, rate limits, circuit breakers, idempotency, cache fills, broadcast
streams and probabilistic counters.

Each directory is its own Go module, so you install only what you use:

```sh
go get github.com/alextanhongpin/core/dsync/ratelimit
```

## Which package do I need?

| I want to… | Package |
| --- | --- |
| Share an API quota or request limit across all replicas | [ratelimit](ratelimit/) |
| Stop calling a failing dependency from every replica at once | [circuitbreaker](circuitbreaker/) |
| Make sure only one replica runs a job / touches a key at a time | [lock](lock/) |
| Return the same result when a client retries a request (idempotency keys) | [idempotent](idempotent/) |
| Store typed values in Redis or on disk, and let only one caller refill a missing entry | [cache](cache/) |
| Collapse duplicate concurrent calls, locally or across replicas | [singleflight](singleflight/) |
| Broadcast events where every reader keeps its own position (replayable) | [channel](channel/) |
| Ask "have I seen this?", "how many?", "what's popular?" cheaply (Bloom, Cuckoo, Count-Min, Top-K, t-digest) | [probs](probs/) |

Each package README has a copy-paste quick start, an explanation of how it works,
the configuration table and the errors you should handle.

## Shared conventions

The packages follow the same general shape (exact constructor names differ; see
each README), so once you've used one the rest feel familiar:

```go
client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
defer client.Close() // you own the client; packages never close it

x, err := pkg.New(client, pkg.Config{ /* zero fields get sensible defaults */ })
if err != nil {
	return err // invalid configuration is reported here, not at first use
}
```

- **Config is a plain struct.** Zero-valued fields are replaced with defaults,
  then the whole config is validated. Packages that have one also expose
  `DefaultConfig()`. `MustNew` variants panic instead of returning an error, for
  use in `main`/init code.
- **You own the Redis client.** Constructors take a `*redis.Client`
  ([go-redis v9](https://github.com/redis/go-redis)); share one client across
  packages and close it yourself when you are done.
- **Errors are sentinels.** Compare with `errors.Is` (for example
  `errors.Is(err, ratelimit.ErrLimitExceeded)`); errors may be wrapped.
- **Everything takes a `context.Context`** and stops when it is cancelled.

## Before you deploy

- **Lua functions must be installed.** `ratelimit` and `circuitbreaker` run their
  logic as Redis Functions. Call the package's `Setup` method on startup (or from
  a deploy step). Upgrading the Go module alone does not update the code inside
  Redis.
- **Redis version.** Some packages rely on recent Redis commands; check the
  "Requirements" section of each README. `probs` needs Redis Stack (or the
  RedisBloom module).
- **Locks are leases, not fencing tokens.** If a process pauses longer than the
  lease TTL, or Redis fails over, two processes can briefly both believe they
  hold a lock. Nothing here gives exactly-once side effects; make protected
  resources tolerate a duplicate (for example with a version check) when that
  matters.
- **Single primary.** Scripts that touch more than one key assume all keys live
  on the same Redis primary. Redis Cluster is only supported where a README says
  so.

## Developing

Every module is tested independently against a real Redis started in Docker
(via [dockertest](https://github.com/ory/dockertest)):

```sh
cd dsync/lock          # or any other module
go vet ./...
go test -race ./...
```

Docker must be running. `probs` tests start a Redis Stack image.
