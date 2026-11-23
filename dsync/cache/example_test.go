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
