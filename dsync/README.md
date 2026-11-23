# dsync

Building blocks for coordinating work **across processes and replicas** using
Redis: locks, rate limits, circuit breakers, idempotency, cache fills and
broadcast streams.

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

For Bloom/Cuckoo filters, Count-Min Sketch, Top-K, t-digest and HyperLogLog,
call go-redis directly (`BFAdd`, `CFAdd`, `CMSIncrBy`, `TopKAdd`, `TDigestAdd`,
`PFAdd`, ...); Redis 8 ships those data types.

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
- **You own the Redis client.** Constructors take a
  [go-redis v9](https://github.com/redis/go-redis) client (`*redis.Client`;
  `ratelimit` and `circuitbreaker` accept any `redis.UniversalClient`). Share
  one client across packages and close it yourself when you are done.
- **Errors are sentinels.** Compare with `errors.Is` (for example
  `errors.Is(err, ratelimit.ErrLimitExceeded)`); errors may be wrapped.
- **Everything takes a `context.Context`** and stops when it is cancelled.

## Before you deploy

- **Lua functions must be installed.** `ratelimit` and `circuitbreaker` run their
  logic as Redis Functions. Call the package's `Setup` method on startup (or from
  a deploy step). Upgrading the Go module alone does not update the code inside
  Redis.
- **Redis version.** Minimum versions differ per package:

  | Package | Minimum Redis | Why |
  | --- | --- | --- |
  | cache (compare-and-swap, leases), idempotent, lock, singleflight | 8.4 | token-checked `SET IFDEQ` / `DELEX` |
  | circuitbreaker | 7.4 | Redis Functions + hash field expiry |
  | ratelimit | 7.0 | Redis Functions |
  | channel | 5.0 | Streams |
- **Locks are leases, not fencing tokens.** If a process pauses longer than the
  lease TTL, or Redis fails over, two processes can briefly both believe they
  hold a lock. Nothing here gives exactly-once side effects; make protected
  resources tolerate a duplicate (for example with a version check) when that
  matters.
- **Redis Cluster.** `ratelimit` and `circuitbreaker` work with a
  `*redis.ClusterClient` (`Setup` installs the functions on every master). The
  other packages target a single primary; operations that touch more than one
  key need those keys in the same hash slot.

## Developing

Every module is tested independently against a real Redis started in Docker
(via [dockertest](https://github.com/ory/dockertest)):

```sh
cd dsync/lock          # or any other module
go vet ./...
go test -race ./...
```

Docker must be running; the first run pulls the Redis image.
