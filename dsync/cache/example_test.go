package cache_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alextanhongpin/core/dsync/cache"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	redis "github.com/redis/go-redis/v9"
)

// exampleClient returns a client for the test Redis and empties it, so each
// example starts from a clean database. In your code, create the client once
// with your own address.
func exampleClient() *redis.Client {
	client := redis.NewClient(&redis.Options{Addr: redistest.Addr()})
	client.FlushAll(context.Background())
	return client
}

// Store and load typed values in Redis.
func Example() {
	ctx := context.Background()
	client := exampleClient()
	defer client.Close()

	type User struct {
		Name string `json:"name"`
	}
	users := cache.MustNew[User](cache.Config{Storage: cache.MustNewRedis(client)})

	if err := users.Store(ctx, "user:1", User{Name: "Ada"}, time.Hour); err != nil {
		panic(err)
	}
	u, err := users.Load(ctx, "user:1")
	fmt.Println(u.Name, err)

	_, err = users.Load(ctx, "user:2")
	fmt.Println(errors.Is(err, cache.ErrNotExist))
	// Output:
	// Ada <nil>
	// true
}

// Read-through caching within one process: create runs only on a miss.
func ExampleCache_LoadOrCreate() {
	ctx := context.Background()
	client := exampleClient()
	defer client.Close()

	c := cache.MustNew[string](cache.Config{Storage: cache.MustNewRedis(client)})
	create := func(ctx context.Context, key string) (string, time.Duration, error) {
		fmt.Println("computing", key)
		return "hello", time.Minute, nil
	}
	for range 2 {
		v, loaded, err := c.LoadOrCreate(ctx, "greeting", create)
		fmt.Println(v, loaded, err)
	}
	// Output:
	// computing greeting
	// hello false <nil>
	// hello true <nil>
}

// Fill a cache entry once across all processes. Callers that arrive while a
// fill is running wait up to Wait and then receive the stored value.
func ExampleLock_LoadOrCreate() {
	ctx := context.Background()
	client := exampleClient()
	defer client.Close()

	lock := cache.MustNewLock(client)
	cfg := cache.LoadOrCreateConfig[[]byte]{
		Wait: 5 * time.Second,
		Create: func(ctx context.Context, key string) ([]byte, time.Duration, error) {
			return []byte("expensive report"), time.Minute, nil
		},
	}
	for range 2 {
		v, loaded, err := lock.LoadOrCreate(ctx, "report:daily", cfg)
		fmt.Printf("%s %v %v\n", v, loaded, err)
	}
	// Output:
	// expensive report false <nil>
	// expensive report true <nil>
}

// LoadOrCreateT stores typed values as JSON.
func ExampleLock_LoadOrCreateT() {
	ctx := context.Background()
	client := exampleClient()
	defer client.Close()

	type Stats struct{ Count int }
	lock := cache.MustNewLock(client)
	s, loaded, err := lock.LoadOrCreateT(ctx, "stats", cache.LoadOrCreateConfig[Stats]{
		Create: func(ctx context.Context, key string) (Stats, time.Duration, error) {
			return Stats{Count: 42}, time.Minute, nil
		},
	})
	fmt.Println(s.Count, loaded, err)
	// Output: 42 false <nil>
}

// Run a job in at most one process at a time.
func ExampleLock_AdvisoryLock() {
	ctx := context.Background()
	client := exampleClient()
	defer client.Close()

	lock := cache.MustNewLock(client)
	err := lock.AdvisoryLock(ctx, "job:cleanup", cache.AdvisoryLockConfig{
		Lock:         10 * time.Second,
		RefreshRatio: 0.5, // renew every 5s while Do runs
		Do: func(ctx context.Context, key string, token []byte) error {
			fmt.Println("running", key)
			return nil
		},
	})
	if errors.Is(err, cache.ErrLocked) {
		fmt.Println("another process is running it")
	}
	fmt.Println(err)
	// Output:
	// running job:cleanup
	// <nil>
}

// Wrap a function with a shared Redis cache.
func ExampleFunc() {
	ctx := context.Background()
	client := exampleClient()
	defer client.Close()

	getUser := cache.MustFunc(func(ctx context.Context, id int) (string, time.Duration, error) {
		fmt.Println("db query", id)
		return fmt.Sprintf("user-%d", id), time.Minute, nil
	}, cache.FuncConfig[int]{
		Lock:  cache.MustNewLock(client),
		KeyFn: func(ctx context.Context, id int) (string, error) { return fmt.Sprintf("user:%d", id), nil },
		Wait:  5 * time.Second,
	})
	for range 2 {
		name, loaded, err := getUser(ctx, 7)
		fmt.Println(name, loaded, err)
	}
	// Output:
	// db query 7
	// user-7 false <nil>
	// user-7 true <nil>
}

// Replay the first response for an idempotency key and reject reuse of the
// key with a different request.
func ExampleIdempotent() {
	ctx := context.Background()
	client := exampleClient()
	defer client.Close()

	type Payment struct {
		Key    string
		Amount int
	}
	pay := cache.MustIdempotent(func(ctx context.Context, p Payment) (string, time.Duration, error) {
		return fmt.Sprintf("charged %d", p.Amount), 24 * time.Hour, nil
	}, cache.FuncConfig[Payment]{
		Lock:  cache.MustNewLock(client),
		KeyFn: func(ctx context.Context, p Payment) (string, error) { return "payment:" + p.Key, nil },
	})

	fmt.Println(pay(ctx, Payment{Key: "abc", Amount: 100}))
	fmt.Println(pay(ctx, Payment{Key: "abc", Amount: 100}))
	_, _, err := pay(ctx, Payment{Key: "abc", Amount: 999})
	fmt.Println(errors.Is(err, cache.ErrConflict))
	// Output:
	// charged 100 false <nil>
	// charged 100 true <nil>
	// true
}

// Persist a small cache in one JSON file.
func ExampleNewFile() {
	ctx := context.Background()
	dir, _ := os.MkdirTemp("", "cache")
	defer os.RemoveAll(dir)

	f, err := cache.NewFile(filepath.Join(dir, "cache.json"))
	if err != nil {
		panic(err)
	}
	defer f.Close() // releases the file lock

	_ = f.Store(ctx, "token", []byte("secret"), time.Hour)
	v, err := f.Load(ctx, "token")
	fmt.Println(string(v), err)
	// Output: secret <nil>
}

// Memoize a function on disk: the result survives program restarts.
func ExampleGoFunc() {
	ctx := context.Background()
	dir, _ := os.MkdirTemp("", "cache")
	defer os.RemoveAll(dir)

	fs, err := cache.NewFS(dir)
	if err != nil {
		panic(err)
	}
	defer fs.Close()

	square := cache.GoFunc(func(ctx context.Context, n int) (int, error) {
		fmt.Println("computing", n)
		return n * n, nil
	}, fs, nil) // nil codec means JSON

	fmt.Println(square(ctx, 4))
	fmt.Println(square(ctx, 4))
	// Output:
	// computing 4
	// 16 <nil>
	// 16 <nil>
}

func ExampleSeparator() {
	sep := cache.Separator(':')
	key := sep.Join("users", "42")
	fmt.Println(key, sep.Split(key))
	// Output: users:42 [users 42]
}
