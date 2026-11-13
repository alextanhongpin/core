// Package idempotent makes a function safe to retry: for a given idempotency
// key it runs the function once, stores the request and response in Redis,
// and replays the stored response to every repeat of that request.
//
// Typical use is an HTTP API that accepts an Idempotency-Key header, or a job
// queue that may deliver the same job more than once.
//
// Minimal usage:
//
//	idem, err := idempotent.NewWithRedis(rdb) // *redis.Client, Redis 8.4+
//	if err != nil { ... }
//	h, err := idem.HandlerFunc(func(ctx context.Context, req ChargeRequest) (Receipt, error) {
//		return payments.Charge(ctx, req)
//	}, idempotent.HandlerConfig{}) // zero config: LockTTL=10s, KeepTTL=24h
//	if err != nil { ... }
//
//	receipt, shared, err := h.Do(ctx, "tenant-1:charge:"+idempotencyKey, req)
//	// shared == true means receipt was replayed from Redis.
//
// Outcomes of Do for the same key:
//   - first call: runs the function and stores the result for KeepTTL.
//   - repeat with the same request: returns the stored result, shared=true.
//   - repeat with a different request: [ErrRequestMismatch].
//   - repeat while the first call is still running in another process:
//     [ErrRequestInFlight]. Callers in the same process wait instead.
//   - the function returned an error: nothing is stored, so a retry runs it
//     again.
//
// Requests and responses are stored as JSON, so both types must round-trip
// through encoding/json. This is not an exactly-once guarantee: if a lease
// expires, Redis loses data, or KeepTTL passes, the function can run again.
package idempotent
