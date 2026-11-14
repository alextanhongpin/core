// Package cache provides typed key/value caching on top of Redis or the local
// filesystem, plus Redis leases that make sure only one process fills a
// missing cache entry at a time.
//
// The package has three layers:
//
//   - Storage backends that store raw bytes: [Redis] (shared, networked),
//     [File] (one JSON snapshot file) and [FS] (one file per key in a
//     directory). All implement [Storage].
//   - [Cache], a typed wrapper that encodes values with a [Codec] (JSON by
//     default) before handing them to a backend.
//   - [Lock], which coordinates cache fills and critical sections across
//     processes using a Redis lease, and the [Func], [Idempotent] and [GoFunc]
//     helpers built on top of it.
//
// Minimal usage:
//
//	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	defer client.Close()
//
//	users := cache.MustNew[User](cache.Config{Storage: cache.MustNewRedis(client)})
//	err := users.Store(ctx, "user:1", User{Name: "Ada"}, time.Hour)
//	u, err := users.Load(ctx, "user:1") // err is cache.ErrNotExist on a miss
//
// Fill a missing entry exactly once across all processes:
//
//	lock := cache.MustNewLock(client)
//	report, loaded, err := lock.LoadOrCreate(ctx, "report", cache.LoadOrCreateConfig[[]byte]{
//		Wait: 5 * time.Second, // wait up to 5s for another process's fill
//		Create: func(ctx context.Context, key string) ([]byte, time.Duration, error) {
//			return buildReport(ctx), time.Minute, nil
//		},
//	})
//
// TTL convention for every backend: ttl > 0 expires the key, ttl <= 0 keeps it
// forever.
//
// The Redis features require Redis 8.4 or newer (SET IFDEQ and DELEX are used
// for compare-and-swap) and a single primary; [Lock.LoadOrCreate] is not Redis
// Cluster compatible. [File] uses flock(2) and only builds on Unix-like systems.
package cache
