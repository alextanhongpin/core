package probs

import (
	"context"
	"errors"
	"fmt"
	"math"

	redis "github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

// DefaultTopK is the number of items tracked by a Top-K that Add or IncrBy
// creates automatically. Call Create first to choose a different size.
const DefaultTopK = 10

// TopK keeps an approximate list of the k most frequent items, for example
// the most called API endpoints or the most common errors (Redis TOPK.*
// commands). Memory depends on k, not on the number of distinct items.
//
// Add and IncrBy create a missing Top-K with Create(key, DefaultTopK). A
// TopK is safe for concurrent use.
type TopK struct {
	client *redis.Client
	group  singleflight.Group
}

// TopKItem is one entry of TopK.Ranked.
type TopKItem struct {
	Item  string
	Count int64
}

// NewTopK returns a TopK that uses client. The client is borrowed: the
// caller keeps ownership and must close it. It returns an error if client
// is nil.
func NewTopK(client *redis.Client) (*TopK, error) {
	if client == nil {
		return nil, errNilClient
	}
	return &TopK{client: client}, nil
}

// MustNewTopK is like NewTopK but panics on error.
func MustNewTopK(client *redis.Client) *TopK {
	v, err := NewTopK(client)
	if err != nil {
		panic(err)
	}
	return v
}

// Add counts one occurrence of each value (TOPK.ADD). For each value it
// returns the item that was pushed out of the top-k list as a result, or ""
// if none was.
func (t *TopK) Add(ctx context.Context, key string, values ...any) ([]string, error) {
	vals, err := t.client.TopKAdd(ctx, key, values...).Result()
	if err == nil {
		return vals, nil
	}

	if err := t.create(ctx, key, err); err != nil {
		return nil, err
	}

	return t.client.TopKAdd(ctx, key, values...).Result()
}

// Count returns the estimated count of each value (TOPK.COUNT). Counts of
// items outside the top-k list are rough; Redis discourages relying on
// this command. Use Ranked for the counts of the items in the list.
func (t *TopK) Count(ctx context.Context, key string, values ...any) ([]int64, error) {
	return t.client.TopKCount(ctx, key, values...).Result()
}

// IncrBy adds each map value to the count of its item (TOPK.INCRBY). Like
// Add it returns the items pushed out of the list; because a map has no
// order, results are in ascending order of the item names.
func (t *TopK) IncrBy(ctx context.Context, key string, kvs map[string]int64) ([]string, error) {
	args := sortedPairs(kvs)

	vals, err := t.client.TopKIncrBy(ctx, key, args...).Result()
	if err == nil {
		return vals, nil
	}
	if err := t.create(ctx, key, err); err != nil {
		return nil, err
	}

	return t.client.TopKIncrBy(ctx, key, args...).Result()
}

// List returns the items currently in the top-k list, most frequent first
// (TOPK.LIST).
func (t *TopK) List(ctx context.Context, key string) ([]string, error) {
	return t.client.TopKList(ctx, key).Result()
}

// ListWithCount returns the items in the top-k list with their estimated
// counts (TOPK.LIST WITHCOUNT). A map loses the ranking; use Ranked to keep
// it.
func (t *TopK) ListWithCount(ctx context.Context, key string) (map[string]int64, error) {
	return t.client.TopKListWithCount(ctx, key).Result()
}

// Ranked returns the items in the top-k list with their estimated counts,
// most frequent first (TOPK.LIST WITHCOUNT).
func (t *TopK) Ranked(ctx context.Context, key string) ([]TopKItem, error) {
	vals, err := t.client.Do(ctx, "TOPK.LIST", key, "WITHCOUNT").Slice()
	if err != nil {
		return nil, err
	}
	if len(vals)%2 != 0 {
		return nil, fmt.Errorf("probs: unexpected TOPK.LIST reply length %d", len(vals))
	}
	items := make([]TopKItem, 0, len(vals)/2)
	for i := 0; i < len(vals); i += 2 {
		item, ok1 := vals[i].(string)
		count, ok2 := vals[i+1].(int64)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("probs: unexpected TOPK.LIST reply entry %T, %T", vals[i], vals[i+1])
		}
		items = append(items, TopKItem{Item: item, Count: count})
	}
	return items, nil
}

// Query reports, for each value, whether it is currently in the top-k list
// (TOPK.QUERY).
func (t *TopK) Query(ctx context.Context, key string, values ...any) ([]bool, error) {
	return t.client.TopKQuery(ctx, key, values...).Result()
}

// Reserve creates an empty Top-K at key that tracks k items, using the
// Redis default width, depth and decay (TOPK.RESERVE). It returns an error
// if key exists. Prefer Create, which sizes the sketch for k and is
// idempotent.
func (t *TopK) Reserve(ctx context.Context, key string, k int64) (string, error) {
	return t.client.TopKReserve(ctx, key, k).Result()
}

// ReserveWithOptions is like Reserve with explicit sketch dimensions: width
// (counters per row), depth (rows) and decay (0 < decay <= 1, the
// probability of decrementing a colliding counter). It returns an error if
// key exists.
func (t *TopK) ReserveWithOptions(ctx context.Context, key string, k, width, depth int64, decay float64) (string, error) {
	return t.client.TopKReserveWithOptions(ctx, key, k, width, depth, decay).Result()
}

// Create creates an empty Top-K at key that tracks the k most frequent
// items, with width k*ln(k), depth max(ln(k), 5) and decay 0.9 as
// recommended by Redis. It is idempotent: if key exists it returns OK and
// leaves it unchanged. k must be positive.
func (t *TopK) Create(ctx context.Context, key string, k int64) (string, error) {
	if k <= 0 {
		return "", errors.New("probs: k must be positive")
	}
	logK := math.Log(float64(k))
	width := max(int64(float64(k)*logK), 1)
	depth := int64(max(logK, 5))
	decay := 0.9
	status, err := t.client.TopKReserveWithOptions(ctx, key, k, width, depth, decay).Result()
	if KeyAlreadyExistsError(err) {
		return OK, nil
	}

	return status, err
}

// create creates key with DefaultTopK if err says it is missing, once per
// key across concurrent callers. Any other err is returned unchanged.
func (t *TopK) create(ctx context.Context, key string, err error) error {
	if !KeyDoesNotExistError(err) {
		return err
	}

	_, err, _ = t.group.Do(key, func() (any, error) {
		return t.Create(ctx, key, DefaultTopK)
	})
	return err
}
