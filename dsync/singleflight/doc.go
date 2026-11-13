// Package singleflight deduplicates work across processes with Redis: for a
// given key, one caller runs the function and the others wait for it to finish.
//
// Group.Do is the low-level primitive: it runs a callback at most once at a
// time per key. Callers in the same process share the leader's result; callers
// in other processes only learn that the leader finished.
//
// Cache.LoadOrStore builds a stampede-safe read-through cache on top of it: on a
// miss, one caller computes the value and stores it in Redis, and every other
// caller reads that stored value.
//
// Minimal use:
//
//	users, err := singleflight.NewCache[User](rdb, singleflight.CacheConfig{})
//	if err != nil {
//		return err
//	}
//	u, _, err := users.LoadOrStore(ctx, "user:42", func(ctx context.Context) (User, error) {
//		return db.LoadUser(ctx, 42)
//	}, time.Minute)
//
// Requires Redis 8.4+ on a single primary (not Redis Cluster).
package singleflight
