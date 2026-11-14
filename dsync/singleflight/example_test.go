package singleflight_test

import (
	"context"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/dsync/singleflight"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	"github.com/redis/go-redis/v9"
)

func ExampleGroup_Do() {
	rdb := redis.NewClient(&redis.Options{Addr: redistest.Addr()})
	defer rdb.Close()

	g, err := singleflight.New(rdb, singleflight.Config{})
	if err != nil {
		panic(err)
	}

	did, err := g.Do(context.Background(), "example:refresh:prices", func(ctx context.Context) error {
		fmt.Println("refreshing prices")
		return nil
	})
	fmt.Println("did:", did, "err:", err)
	// Output:
	// refreshing prices
	// did: true err: <nil>
}

func ExampleCache_LoadOrStore() {
	rdb := redis.NewClient(&redis.Options{Addr: redistest.Addr()})
	defer rdb.Close()

	type User struct{ Name string }
	users, err := singleflight.NewCache[User](rdb, singleflight.CacheConfig{})
	if err != nil {
		panic(err)
	}

	loadUser := func(ctx context.Context) (User, error) {
		fmt.Println("querying database")
		return User{Name: "Alice"}, nil
	}

	ctx := context.Background()
	for range 2 {
		u, loaded, err := users.LoadOrStore(ctx, "example:user:42", loadUser, time.Minute)
		if err != nil {
			panic(err)
		}
		fmt.Println(u.Name, "loaded from cache:", loaded)
	}
	// Output:
	// querying database
	// Alice loaded from cache: false
	// Alice loaded from cache: true
}
