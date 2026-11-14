package probs_test

import (
	"context"
	"fmt"

	"github.com/alextanhongpin/core/dsync/probs"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	redis "github.com/redis/go-redis/v9"
)

func newExampleClient(keys ...string) (*redis.Client, func()) {
	client := redis.NewClient(&redis.Options{Addr: redistest.Addr()})
	return client, func() {
		client.Del(context.Background(), keys...)
		client.Close()
	}
}

// Count unique visitors per day, then for the whole period.
func ExampleHyperLogLog() {
	ctx := context.Background()
	client, done := newExampleClient("visitors:mon", "visitors:tue", "visitors:week")
	defer done()

	hll := probs.MustNewHyperLogLog(client)
	_, _ = hll.Add(ctx, "visitors:mon", "alice", "bob", "alice")
	_, _ = hll.Add(ctx, "visitors:tue", "bob", "carol")

	mon, _ := hll.Count(ctx, "visitors:mon")
	both, _ := hll.Count(ctx, "visitors:mon", "visitors:tue") // union
	fmt.Println("monday:", mon)
	fmt.Println("monday or tuesday:", both)

	_, _ = hll.Merge(ctx, "visitors:week", "visitors:mon", "visitors:tue")
	week, _ := hll.Count(ctx, "visitors:week")
	fmt.Println("week:", week)
	// Output:
	// monday: 2
	// monday or tuesday: 3
	// week: 3
}

// Show each ad to a user at most once.
func ExampleBloomFilter() {
	ctx := context.Background()
	client, done := newExampleClient("ads:seen:user1")
	defer done()

	bf := probs.MustNewBloomFilter(client)
	// Optional: size the filter (1% false positives, 10k items).
	if _, err := bf.Reserve(ctx, "ads:seen:user1", 0.01, 10_000); err != nil {
		panic(err)
	}

	for _, ad := range []string{"ad-1", "ad-2", "ad-1"} {
		added, err := bf.Add(ctx, "ads:seen:user1", ad)
		if err != nil {
			panic(err)
		}
		if added {
			fmt.Println("show", ad)
		} else {
			fmt.Println("skip", ad, "(probably shown already)")
		}
	}
	seen, _ := bf.Exists(ctx, "ads:seen:user1", "ad-3")
	fmt.Println("ad-3 seen:", seen)
	// Output:
	// show ad-1
	// show ad-2
	// skip ad-1 (probably shown already)
	// ad-3 seen: false
}

// Track items that can later be removed, such as active sessions.
func ExampleCuckooFilter() {
	ctx := context.Background()
	client, done := newExampleClient("sessions")
	defer done()

	cf := probs.MustNewCuckooFilter(client)
	_, _ = cf.AddNX(ctx, "sessions", "sess-1")
	_, _ = cf.AddNX(ctx, "sessions", "sess-2")
	_, _ = cf.Delete(ctx, "sessions", "sess-1") // logged out

	active, _ := cf.MExists(ctx, "sessions", "sess-1", "sess-2")
	fmt.Println(active)
	// Output:
	// [false true]
}

// Count API calls per user without storing a counter per user.
func ExampleCountMinSketch() {
	ctx := context.Background()
	client, done := newExampleClient("api:calls")
	defer done()

	cms := probs.MustNewCountMinSketch(client)
	// The sketch is created on first use. Counts come back sorted by name:
	// alice, bob.
	counts, created, err := cms.IncrBy(ctx, "api:calls", map[string]int64{"bob": 1, "alice": 3})
	if err != nil {
		panic(err)
	}
	fmt.Println(counts, created)

	counts, _ = cms.Query(ctx, "api:calls", "alice", "bob", "carol")
	fmt.Println(counts)
	// Output:
	// [3 1] true
	// [3 1 0]
}

// Find the most called endpoints.
func ExampleTopK() {
	ctx := context.Background()
	client, done := newExampleClient("api:top")
	defer done()

	top := probs.MustNewTopK(client)
	if _, err := top.Create(ctx, "api:top", 3); err != nil { // track the top 3
		panic(err)
	}
	_, _ = top.Add(ctx, "api:top",
		"/login", "/orders", "/orders", "/search", "/search", "/search")

	ranked, err := top.Ranked(ctx, "api:top")
	if err != nil {
		panic(err)
	}
	for _, it := range ranked {
		fmt.Println(it.Item, it.Count)
	}
	// Output:
	// /search 3
	// /orders 2
	// /login 1
}

// Track request latency percentiles.
func ExampleTDigest() {
	ctx := context.Background()
	client, done := newExampleClient("latency:ms")
	defer done()

	td := probs.MustNewTDigest(client)
	latencies := make([]float64, 0, 100)
	for i := 1; i <= 100; i++ {
		latencies = append(latencies, float64(i))
	}
	if _, err := td.Add(ctx, "latency:ms", latencies...); err != nil {
		panic(err)
	}

	q, _ := td.Quantile(ctx, "latency:ms", 0.5, 0.99)
	maxMs, _ := td.Max(ctx, "latency:ms")
	fmt.Printf("p50=%.0fms p99=%.0fms max=%.0fms\n", q[0], q[1], maxMs)
	// Output:
	// p50=50ms p99=100ms max=100ms
}
