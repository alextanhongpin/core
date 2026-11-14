// Package probs wraps the Redis probabilistic data structures so each
// question has a small, typed API:
//
//	HyperLogLog     how many distinct items?       (PFADD/PFCOUNT, core Redis)
//	BloomFilter     have I seen this item?         (BF.*, no deletes)
//	CuckooFilter    have I seen it? (deletable)    (CF.*)
//	CountMinSketch  how often did each item occur? (CMS.*)
//	TopK            which items occur most?        (TOPK.*)
//	TDigest         what is the p50/p99 value?     (TDIGEST.*)
//
// All answers are estimates with bounded error in exchange for small,
// fixed memory. Everything except HyperLogLog needs Redis 8+ or Redis Stack
// (the RedisBloom module).
//
// CountMinSketch.IncrBy, TopK.Add/IncrBy and TDigest.Add create their key
// on first use, and the Init/Create methods are idempotent, so no separate
// set-up step is required.
//
// Minimal usage:
//
//	hll := probs.MustNewHyperLogLog(client) // client is a *redis.Client you own
//	_, _ = hll.Add(ctx, "visitors:2024-05-01", "alice", "bob", "alice")
//	n, _ := hll.Count(ctx, "visitors:2024-05-01") // 2
package probs
