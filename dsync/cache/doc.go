// Package cache provides typed key/value storage on top of Redis or the local
// filesystem.
//
// The package has two layers:
//
//   - Storage backends that store raw bytes: [Redis] (shared, networked),
//     [File] (one JSON snapshot file) and [FS] (one file per key in a
//     directory). All implement [Storage].
//   - [Cache], a typed wrapper that encodes values with a [Codec] (JSON by
//     default) before handing them to a backend. [GoFunc] memoizes a function
//     in any backend.
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
// LoadOrCreate only coalesces fills inside one process. To fill a missing
// entry once across all processes use the Cache type in
// github.com/alextanhongpin/core/dsync/singleflight; for leases use
// github.com/alextanhongpin/core/dsync/lock; for idempotency keys use
// github.com/alextanhongpin/core/dsync/idempotent.
//
// TTL convention for every backend: ttl > 0 expires the key, ttl <= 0 keeps it
// forever.
//
// CompareAndSwap and CompareAndDelete on [Redis] require Redis 8.4 or newer
// (SET IFDEQ and DELEX). [File] uses flock(2) and only builds on Unix-like
// systems.
package cache
