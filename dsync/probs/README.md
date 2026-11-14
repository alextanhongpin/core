# probs

Typed Go wrappers for the probabilistic data structures built into Redis:
answer "how many?", "seen it?", "how often?", "which are most common?" and
"what is the p99?" with small, fixed memory and a known error.

| Type | Question it answers | Error characteristics | Redis commands |
|---|---|---|---|
| `HyperLogLog` | How many **distinct** items? (unique visitors) | ~0.81% standard error, max 12 KB per key | `PFADD`, `PFCOUNT`, `PFMERGE` |
| `BloomFilter` | Have I **seen** this item? (ad already shown) | "no" is always right; "yes" may be a false positive (1% by default). No deletes | `BF.*` |
| `CuckooFilter` | Have I seen it, with **delete** and per-item count | Like Bloom; deleting an item that was never added corrupts other entries | `CF.*` |
| `CountMinSketch` | How **often** did each item occur? (API calls per user) | Never under-counts; may over-count by up to 0.1% of all increments (default) | `CMS.*` |
| `TopK` | Which k items occur **most**? (top endpoints, top errors) | Approximate ranking and counts; ties and rare items may be dropped | `TOPK.*` |
| `TDigest` | What is the **p50 / p99** of these numbers? (latency) | Approximate quantiles, most accurate near the extremes | `TDIGEST.*` |

Use these when an estimate is good enough and an exact answer would need
memory proportional to the number of items. Don't use them for anything that
must be exact (billing, permissions, stock levels).

## Install

```sh
go get github.com/alextanhongpin/core/dsync/probs
```

Requirements:

- Go 1.27 or newer (from this module's `go.mod`).
- A `*redis.Client` from `github.com/redis/go-redis/v9`.
- `HyperLogLog`: any Redis version. All other types: Redis 8.0+ (the
  probabilistic structures are built in), or Redis Stack / Redis with the
  RedisBloom module.

API reference: [pkg.go.dev/github.com/alextanhongpin/core/dsync/probs](https://pkg.go.dev/github.com/alextanhongpin/core/dsync/probs)

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/alextanhongpin/core/dsync/probs"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // your Redis address
	})
	defer client.Close()

	// Unique visitors (works on any Redis).
	hll := probs.MustNewHyperLogLog(client)
	if _, err := hll.Add(ctx, "visitors:2024-05-01", "alice", "bob", "alice"); err != nil {
		log.Fatal(err)
	}
	visitors, err := hll.Count(ctx, "visitors:2024-05-01")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("unique visitors:", visitors) // 2

	// Most called endpoints (needs Redis 8+ or Redis Stack).
	// The Top-K is created on first use; call Create to pick k yourself.
	// Add takes ...any: pass strings directly, or convert a []string first.
	top := probs.MustNewTopK(client)
	if _, err := top.Add(ctx, "endpoints", "/search", "/orders", "/search"); err != nil {
		log.Fatal(err)
	}
	ranked, err := top.Ranked(ctx, "endpoints")
	if err != nil {
		log.Fatal(err)
	}
	for _, it := range ranked {
		fmt.Println(it.Item, it.Count) // /search 2, then /orders 1
	}

	// Latency percentiles.
	td := probs.MustNewTDigest(client)
	if _, err := td.Add(ctx, "latency:ms", 12, 15, 11, 250, 14); err != nil {
		log.Fatal(err)
	}
	q, err := td.Quantile(ctx, "latency:ms", 0.5, 0.99)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("p50=%.0fms p99=%.0fms\n", q[0], q[1])
}
```

Output on an empty Redis:

```
unique visitors: 2
/search 2
/orders 1
p50=14ms p99=250ms
```

The keys stay in Redis, so a second run accumulates: the visitor count stays
2 (same names), the Top-K counts double (`/search 4`, `/orders 2`), and the
percentiles are computed over both runs. Delete the keys (`DEL
visitors:2024-05-01 endpoints latency:ms`) to start over.

Runnable examples for every type are in [example_test.go](example_test.go)
and on pkg.go.dev.

## Why use this instead of raw go-redis?

go-redis already has a method for every command (`client.BFAdd`,
`client.CMSIncrBy`, ...). What this package adds differs per type:

- **`CountMinSketch`, `TopK`, `TDigest`: real convenience.** With raw
  go-redis you must create these keys before the first write, or every write
  fails with `key does not exist`. Here the first `IncrBy` / `Add` creates the
  key (once, even with concurrent callers), `Init` / `Create` are safe to call
  repeatedly, `IncrBy` accepts a `map[string]int64` and returns results in a
  fixed (sorted) order, `TopK.Create(k)` picks sensible sketch dimensions,
  `TopK.Ranked` returns the list with counts in rank order, and
  `CountMinSketch.Merge` into a new key creates it with the sources' size.
- **`HyperLogLog`, `BloomFilter`, `CuckooFilter`: thin pass-throughs.** Redis
  already creates these on first write, so the wrapper only gives you a
  smaller, documented API. Calling go-redis directly is just as good.

## API

Every type is created the same way; `New...` returns an error for a `nil`
client, `MustNew...` panics:

```go
func NewHyperLogLog(client *redis.Client) (*HyperLogLog, error)
func MustNewHyperLogLog(client *redis.Client) *HyperLogLog
// likewise NewBloomFilter, NewCuckooFilter, NewCountMinSketch, NewTopK, NewTDigest
```

Status strings are Redis's reply, normally `probs.OK` (`"OK"`). Results of
the variadic methods are in the same order as the values passed. Note that
the argument types differ per type: Bloom, HyperLogLog, CMS `Query` and
Top-K take `any` values, Cuckoo single-value methods take `string`, and a
`[]string` must be converted to `[]any` before spreading it with `...`.

### HyperLogLog

```go
func (*HyperLogLog) Add(ctx context.Context, key string, values ...any) (int64, error) // 1 if the estimate changed, else 0
func (*HyperLogLog) Count(ctx context.Context, keys ...string) (int64, error)         // several keys: size of the union; missing key = 0
func (*HyperLogLog) Merge(ctx context.Context, destKey string, srcKeys ...string) (string, error)
```

### BloomFilter

```go
func (*BloomFilter) Add(ctx context.Context, key string, value any) (bool, error) // true if newly added
func (*BloomFilter) MAdd(ctx context.Context, key string, values ...any) ([]bool, error)
func (*BloomFilter) Exists(ctx context.Context, key string, value any) (bool, error) // missing key = false
func (*BloomFilter) MExists(ctx context.Context, key string, values ...any) ([]bool, error)
func (*BloomFilter) Reserve(ctx context.Context, key string, errorRate float64, capacity int64) (string, error) // errorRate 0.01 = 1%
```

### CuckooFilter

```go
func (*CuckooFilter) Add(ctx context.Context, key, value string) (bool, error)   // always adds another copy
func (*CuckooFilter) AddNX(ctx context.Context, key, value string) (bool, error) // false if (probably) present
func (*CuckooFilter) Exists(ctx context.Context, key, value string) (bool, error)
func (*CuckooFilter) MExists(ctx context.Context, key string, values ...any) ([]bool, error)
func (*CuckooFilter) Count(ctx context.Context, key, value string) (int64, error)  // copies stored; may over-count
func (*CuckooFilter) Delete(ctx context.Context, key, value string) (bool, error) // false if not found
func (*CuckooFilter) Reserve(ctx context.Context, key string, capacity int64) (string, error)
```

### CountMinSketch

`exists` is true when the key was already there (nothing changed);
`created` is true when `IncrBy` had to create the sketch.

```go
func (*CountMinSketch) Init(ctx context.Context, key string) (status string, exists bool, err error)
func (*CountMinSketch) InitByProb(ctx context.Context, key string, errorRate, errorProbability float64) (status string, exists bool, err error)
func (*CountMinSketch) InitByDim(ctx context.Context, key string, width, depth int64) (status string, exists bool, err error)
func (*CountMinSketch) IncrBy(ctx context.Context, key string, kvs map[string]int64) (counts []int64, created bool, err error) // counts in sorted item order
func (*CountMinSketch) Query(ctx context.Context, key string, values ...any) ([]int64, error) // unseen item = 0
func (*CountMinSketch) Merge(ctx context.Context, destKey string, sourceKeys ...string) (string, error)
func (*CountMinSketch) MergeWithWeight(ctx context.Context, destKey string, sourceKeys map[string]int64) (string, error) // map: source key -> weight
```

### TopK

```go
type TopKItem struct {
	Item  string
	Count int64
}

func (*TopK) Create(ctx context.Context, key string, k int64) (string, error) // idempotent, sized for k
func (*TopK) Reserve(ctx context.Context, key string, k int64) (string, error) // errors if key exists
func (*TopK) ReserveWithOptions(ctx context.Context, key string, k, width, depth int64, decay float64) (string, error)
func (*TopK) Add(ctx context.Context, key string, values ...any) ([]string, error)          // per value: item pushed out of the list, or ""
func (*TopK) IncrBy(ctx context.Context, key string, kvs map[string]int64) ([]string, error) // like Add, in sorted item order
func (*TopK) Ranked(ctx context.Context, key string) ([]TopKItem, error)                   // most frequent first
func (*TopK) List(ctx context.Context, key string) ([]string, error)                       // most frequent first
func (*TopK) ListWithCount(ctx context.Context, key string) (map[string]int64, error)      // unordered
func (*TopK) Query(ctx context.Context, key string, values ...any) ([]bool, error)         // in the list now?
func (*TopK) Count(ctx context.Context, key string, values ...any) ([]int64, error)        // rough outside the list
```

`ReserveWithOptions` parameters, in order: `k` (items to track), `width`
(counters per row), `depth` (rows), `decay` (0 < decay <= 1, probability of
decrementing a colliding counter). Redis's own defaults are width 8, depth
7, decay 0.9.

### TDigest

```go
func (*TDigest) Create(ctx context.Context, key string) (status string, exists bool, err error)
func (*TDigest) CreateWithCompression(ctx context.Context, key string, compression int64) (status string, exists bool, err error)
func (*TDigest) Add(ctx context.Context, key string, values ...float64) (string, error)
func (*TDigest) Quantile(ctx context.Context, key string, quantiles ...float64) ([]float64, error) // 0.5 = p50, 0.99 = p99
func (*TDigest) CDF(ctx context.Context, key string, values ...float64) ([]float64, error)        // fraction <= value, 0..1
func (*TDigest) Min(ctx context.Context, key string) (float64, error)
func (*TDigest) Max(ctx context.Context, key string) (float64, error)
func (*TDigest) Rank(ctx context.Context, key string, values ...float64) ([]int64, error)    // -1 below min
func (*TDigest) RevRank(ctx context.Context, key string, values ...float64) ([]int64, error) // -1 above max
func (*TDigest) ByRank(ctx context.Context, key string, ranks ...uint64) ([]float64, error)
func (*TDigest) ByRevRank(ctx context.Context, key string, ranks ...uint64) ([]float64, error)
func (*TDigest) TrimmedMean(ctx context.Context, key string, lo, hi float64) (float64, error) // lo, hi are quantiles
```

### Errors and constants

```go
func KeyDoesNotExistError(err error) bool
func KeyAlreadyExistsError(err error) bool

const OK = "OK"
const DefaultTopK = 10
const DefaultCMSErrorRate = 0.001
const DefaultCMSErrorProbability = 0.002
```

## How it works

Each type is a stateless handle around your `*redis.Client`; all data lives
in Redis under the keys you pass. One handle can serve any number of keys
and is safe for concurrent use.

Automatic creation (CMS, Top-K, t-digest):

```
IncrBy(key, ...) --> CMS.INCRBY --ok--> return counts
                         |
                 "key does not exist"
                         v
             CMS.INITBYPROB (one caller per key; "already exists" is fine)
                         v
                     CMS.INCRBY again --> return counts, created=true
```

The retry happens at most once, so other errors (wrong type, module not
loaded, missing merge source) are returned straight away.

## Defaults

| What | Default | How to change |
|---|---|---|
| Count-Min Sketch created by `Init` / `IncrBy` | error rate `DefaultCMSErrorRate` (0.001), probability `DefaultCMSErrorProbability` (0.002) | `InitByProb` or `InitByDim` before the first write |
| Top-K created by `Add` / `IncrBy` | k = `DefaultTopK` (10), sized as in `Create` | `Create(ctx, key, k)` before the first write |
| `TopK.Create(k)` | width k·ln(k), depth max(ln(k), 5), decay 0.9 | `ReserveWithOptions` |
| t-digest created by `Add` | Redis default compression (100) | `CreateWithCompression` before the first write |
| Bloom filter created by `Add` | Redis default: 1% error rate, capacity 100, grows when full | `Reserve` before the first write |
| Cuckoo filter created by `Add` | Redis default: capacity 1024, grows when full | `Reserve` before the first write |

## Errors

The package has no sentinel errors; it returns the Redis error. Two helpers
classify the common ones:

| Situation | How to detect | What to do |
|---|---|---|
| Reading a CMS / Top-K / t-digest key that does not exist (e.g. `Query`, `List`, `Quantile`); Redis says `ERR T-Digest: key does not exist` or similar | `probs.KeyDoesNotExistError(err)` | treat as "no data yet" |
| Reading a t-digest that exists but has no values (created with `Create*`, never `Add`ed) | no error: `Quantile`, `Min`, `Max`, `CDF`, `TrimmedMean` return `NaN`; `Rank`, `RevRank` return -2 | check `math.IsNaN` / -2 |
| Creating a CMS / Top-K / t-digest that exists (`Reserve*` only; `Init*` and `Create*` already return success) | `probs.KeyAlreadyExistsError(err)` | ignore, or use `Init` / `Create` instead |
| `BloomFilter.Reserve` / `CuckooFilter.Reserve` on an existing key | Redis error `ERR item exists` | reserve only once, before the first `Add` |
| Module not available | Redis error `unknown command 'BF.ADD'` (or similar) | use Redis 8+ or Redis Stack |
| `nil` client passed to a constructor | `New...` returns an error, `MustNew...` panics | fix the wiring |

An estimate is never reported as an error: a false positive or an
over-count is a normal successful result.

## Caveats

- **Size before you write.** Auto-created structures use the defaults above.
  Once a key exists, `Init` / `Create` do **not** resize it; delete the key
  first if you need different settings.
- **Results of `IncrBy` follow sorted item names**, not the map's order:
  `IncrBy(map{"bob":1, "alice":3})` returns `[alice's, bob's]`.
- **CMS merge** needs all sketches to have the same width and depth, and
  overwrites the destination.
- **Counts only grow.** CMS and HyperLogLog cannot remove items. Use a key
  per time window (e.g. `visitors:2024-05-01`) with `EXPIRE` to forget old
  data.
- **Cuckoo `Add` is not idempotent**; adding twice stores two copies. Use
  `AddNX` for set semantics.
- **Concurrent auto-creation** is de-duplicated within one process; across
  processes the "already exists" case is handled, so it is still safe.

## Running the tests

Tests start a `redis:8.6.2` container through Docker:

```sh
cd dsync/probs
go test -race ./...
```
