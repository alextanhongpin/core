# singleflight

Coalesce concurrent operations locally and coordinate them with renewable leases
on one Redis primary.

```go
g, err := singleflight.New(client, singleflight.Config{})
if err != nil { return err }
did, err := g.Do(ctx, "refresh:123", func(ctx context.Context) error {
    return refresh(ctx)
})
```

`Config` is copied, defaulted, validated, and stored privately. Zero `LockTTL` and
`WaitTTL` select ten seconds; zero `PollInterval` selects ten milliseconds. Leases
must last at least one millisecond. `MustNew` is a startup helper. Redis clients
are borrowed and must remain open until calls finish.

The leader executes synchronously, renews at three quarters of the lease, and
joins renewal before bounded release. Cancellation or renewal failure cancels the
callback context; callbacks must cooperate with it. A call waits for its own
callback to finish even after cancellation. Panics propagate after cleanup;
local followers get a leader-panic error. Reentering the same key from its leader
would deadlock and is unsupported.

`did` is true for a successful leader and false for followers. Local followers
share the leader's result error and can cancel their own wait independently.
Remote followers only observe lease release/absence; they cannot infer success
or receive the leader's error. Remote waits use acknowledged pub/sub plus polling
and return `ErrTimeout` when their wait budget expires. Redis errors propagate.
Lease expiry/failover/partitions can allow overlapping work; use external fencing
when the downstream system requires exclusivity.

```go
c, err := singleflight.NewCache[User](client, singleflight.CacheConfig{})
if err != nil { return err }
user, loaded, err := c.LoadOrStore(ctx, "user:123", func(ctx context.Context) (User, error) {
    return loadUser(ctx)
}, time.Minute)
```

`CacheConfig` also has a `Suffix` defaulting to `fetch`. Cache fills recheck Redis
inside the leader, encode JSON, and atomically verify the lease before publishing.
Only the caller that stores a new value reports `loaded=false`. A failed remote
fill leaves a cache miss, which is returned as `redis.Nil`. Zero value TTL means
permanent; positive TTL must be at least one millisecond.

The reserved `singleflight:lease:` namespace separates tokens from values. Cache
publication uses two keys on a single primary and is not Redis Cluster compatible.
Redis must support SETIFDEQ/DELEX. Do not mix older key-scheme workers with v0.1
workers during migration.

Migration: construct with value Config/CacheConfig and handle errors (or use Must
helpers). Public mutable fields and obsolete BackOff helpers were removed. Lease
and wait durations now belong to Config rather than each Do call.

Run `go test -race -timeout 120s ./...` and `go vet ./...`; tests use Redis Docker.
