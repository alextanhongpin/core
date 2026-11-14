// Package lock runs a function while holding a Redis-backed lease on a key,
// so that only one process at a time does the work for that key.
//
// A lease is a Redis key holding a random token with a TTL. Do acquires it,
// renews it in the background while the callback runs, and deletes it when the
// callback returns. Renewal and release only touch the key if it still holds
// the caller's token, so one process never releases another's lease.
//
// Minimal use:
//
//	locker, err := lock.New(lock.NewClient(rdb), lock.DefaultConfig())
//	if err != nil {
//		return err
//	}
//	err = locker.Do(ctx, "reports:daily", func(ctx context.Context) error {
//		return rebuildReport(ctx) // stop when ctx is canceled
//	})
//
// A lease is not a fencing token: if a process pauses longer than the TTL, or
// Redis fails over, two holders can overlap. Protect the downstream resource
// with its own check if overlapping writes are unsafe. Requires Redis 8.4+.
package lock
