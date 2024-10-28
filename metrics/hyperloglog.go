package metrics

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

// app:data_type:key:value:key:value

type HyperLogLog struct {
	Client *redis.Client
}

// use this to track unique page views.
func NewHyperLogLog(client *redis.Client) *HyperLogLog {
	return &HyperLogLog{
		Client: client,
	}
}

func (c *HyperLogLog) Store(ctx context.Context, key, val string) (stored bool, err error) {
	n, err := c.Client.PFAdd(ctx, key, val).Result()
	return n == 1, err
}

func (c *HyperLogLog) Load(ctx context.Context, key string) (count int64, err error) {
	return c.Client.PFCount(ctx, key).Result()
}

// Use this to track unique page views/actions. For unique count, use hyperloglog
type BloomFilter struct {
	Client *redis.Client
}

func (bf *BloomFilter) Add(ctx context.Context) {
	bf.Client.BFAdd(ctx)
}

func (bf *BloomFilter) Exists()  {}
func (bf *BloomFilter) Reserve() {}

// Similar to bloomfilter, except it can be deleted.
type CuckooFilter struct {
	Client *redis.Client
}

func (cf *CuckooFilter) Add()    {}
func (cf *CuckooFilter) Exists() {}
func (cf *CuckooFilter) Delete() {}

// Use this to track latency of the server for sql operations, api requests etc.
// We use this together with top-k to see the top performing api requests.
type TDigest struct {
	Client *redis.Client
}

func (t *TDigest) Create()   {}
func (t *TDigest) Add()      {}
func (t *TDigest) CDF()      {}
func (t *TDigest) Quantile() {}
func (t *TDigest) Min()      {}
func (t *TDigest) Max()      {}

type TopK struct {
	Client redis.Client
}

func (t *TopK) Create() {}
func (t *TopK) Add()    {}
func (t *TopK) List()   {}

// Use this to count the stream of events. Prefer this over counter (?).
// Does not track uniqueness.
type CountMinSketch struct {
	Client *redis.Client
}

func (cms *CountMinSketch) Init()   {}
func (cms *CountMinSketch) IncrBy() {}
func (cms *CountMinSketch) Query()  {}
