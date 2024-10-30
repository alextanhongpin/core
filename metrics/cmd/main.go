package main

import (
	"context"
	"fmt"

	"github.com/alextanhongpin/core/metrics"
	redis "github.com/redis/go-redis/v9"
)

var client = redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
})
var ctx = context.Background()

func main() {
	//testTDigest()
	//testTopK()
	testCMS()
}

func testCMS() {
	cms := metrics.CountMinSketch{Client: client}
	fmt.Println(cms.InitByProb(ctx, "cms", 0.01, 0.02))
	fmt.Println(cms.IncrBy(ctx, "cms",
		metrics.Tuple[any, int]{"hello", 10},
		metrics.Tuple[any, int]{"world", 20},
	),
	)
	fmt.Println(cms.Query(ctx, "cms", "hello", "world"))
}

func testTopK() {
	tk := metrics.TopK{Client: client}
	fmt.Println(tk.Reserve(ctx, "topk", 10))
	fmt.Println(tk.Add(ctx, "topk", "hello", "world", 1, 1, 1))
	fmt.Println(tk.Count(ctx, "topk", "hello", "world", 1))
	fmt.Println(tk.IncrBy(ctx, "topk",
		metrics.Tuple[any, int]{"hello", 10},
		metrics.Tuple[any, int]{"world", 20},
	),
	)
	fmt.Println(tk.Count(ctx, "topk", "hello", "world", 1))
	fmt.Println(tk.List(ctx, "topk"))
	fmt.Println(tk.ListWithCount(ctx, "topk"))
	fmt.Println(tk.Query(ctx, "topk", "hello", "world", 100))

}

func testTDigest() {
	td := metrics.TDigest{Client: client}
	fmt.Println(td.CreateWithCompression(ctx, "tdigest", 100))
	fmt.Println(td.Add(ctx, "tdigest", 1, 2, 3, 4, 5, 5, 5))
	fmt.Println(td.CDF(ctx, "tdigest", 1, 2, 3, 4, 5))
	fmt.Println(td.Quantile(ctx, "tdigest", 0.5, 0.9, 0.95))
	fmt.Println(td.Min(ctx, "tdigest"))
	fmt.Println(td.Max(ctx, "tdigest"))
	fmt.Println(td.Rank(ctx, "tdigest", 5))
	fmt.Println(td.RevRank(ctx, "tdigest", 5))
	fmt.Println(td.ByRank(ctx, "tdigest", 1))
	fmt.Println(td.ByRevRank(ctx, "tdigest", 1))
	fmt.Println(td.TrimmedMean(ctx, "tdigest", 0.1, 0.9))
}

func testCF() {
	cf := metrics.CuckooFilter{Client: client}
	fmt.Println(cf.Add(ctx, "hello", "a"))
	fmt.Println(cf.AddNX(ctx, "hello", "a"))
	fmt.Println(cf.Exists(ctx, "hello", "a"))
	fmt.Println(cf.Count(ctx, "hello", "a"))
	fmt.Println(cf.MExists(ctx, "hello", "a", 1, true, false))
	fmt.Println(cf.Delete(ctx, "hello", "a"))
}

func testBF() {
	bf := metrics.BloomFilter{Client: client}
	fmt.Println(bf.MAdd(ctx, "hello", "a", 1, true))
	fmt.Println(bf.Exists(ctx, "hello", "a"))
	fmt.Println(bf.MExists(ctx, "hello", "a", 1, true, false))
}

func testHLL() {
	hll := metrics.NewHyperLogLog(client)
	fmt.Println(hll.Add(ctx, "hello", false))
	fmt.Println(hll.Add(ctx, "world", true))

	fmt.Println(hll.Count(ctx, "hello"))
	fmt.Println(hll.Count(ctx, "world"))
	fmt.Println(hll.Merge(ctx, "newkey", "unknown"))
	fmt.Println(hll.Count(ctx, "newkey"))
}
