# cache

Typed storage wrappers, file storage, and Redis cache-fill leases.

```go
storage, err := cache.NewRedis(client)
if err != nil { return err }
c, err := cache.New[User](cache.Config{Storage: storage})
if err != nil { return err }
err = c.Store(ctx, "user:123", User{Name: "Ada"}, time.Hour)
user, err := c.Load(ctx, "user:123")
```

Configuration is passed by value, defaulted and validated locally. JSON is the
default codec; codecs and storage are shared dependencies and must support
concurrent calls. `MustNew`, `MustNewRedis`, and `MustNewLock` are startup helpers.
`Cache.Close` forwards to storage; Redis wrappers borrow the client and their
`Close` does nothing. File and FS own their handles and require `Close`.

`LoadOrCreate` on a storage instance coalesces concurrent local fills. Followers
wait for the leader, whose context drives the factory. Factories can access other
keys; recursively filling the same key deadlocks. File/FS factories run outside
the storage mutex. Separate processes require Redis lease coordination.

```go
l, err := cache.NewLock(client)
if err != nil { return err }
value, loaded, err := l.LoadOrCreate(ctx, "report", cache.LoadOrCreateConfig[[]byte]{
    Lock: 10*time.Second, RefreshRatio: .7, Wait: 5*time.Second,
    Create: func(ctx context.Context, key string) ([]byte, time.Duration, error) {
        return buildReport(ctx), time.Minute, nil
    },
})
```

Lease config values use a ten-second default `Lock`; zero `Wait` rejects immediate
contention with `ErrLocked`. Positive `Wait` bounds the total admission wait and
expires with `context.DeadlineExceeded`. Zero `RefreshRatio` disables renewal;
positive ratios must be below one. Callbacks run synchronously and must cooperate
with cancellation. Renewal loss cancels them; the call joins renewal and callback
before bounded cleanup. Panic propagation also releases the lease.

Lease keys live in the reserved `cache:lease:` namespace, separate from data.
Cache publication checks ownership atomically on a single Redis primary. Leases
are not fencing tokens for external writes and cannot promise exactly-once work
under pauses, failover, or network partitions. The two-key publication script is
not Redis Cluster compatible. `UseStream` is deprecated and ignored; admission
uses polling. Redis conditional operations require SETIFDEQ/DELEX.

`Func` and `Idempotent` return `(function, error)` and accept a value
`FuncConfig[K]` with required `KeyFn` and `Lock`, and default JSON `Codec`.
`MustFunc` and `MustIdempotent` are startup variants. Callback, codec, and backend
errors remain inspectable. An idempotent request mismatch returns `ErrConflict`.

File snapshots use atomic rename while a stable sidecar `.lock` file holds the
lifetime advisory lock. All cooperating writers must lock the same sidecar file.
File clones bytes and persists expiry changes. FS requires a sole directory owner; it does not coordinate independent processes. FS value and TTL
index writes are separate, so storage errors may leave partial changes.

Zero storage TTL means permanent. Redis TTL and Expire retain millisecond
precision. File/FS calls after close return `ErrClosed`.

Run `go test -race -timeout 120s ./...` and `go vet ./...`; Redis tests use Docker.
