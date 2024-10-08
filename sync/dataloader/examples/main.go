package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/dataloader"
)

func main() {
	type User struct {
		ID int
	}

	ctx := context.Background()
	dl := dataloader.New(ctx, &dataloader.Options[int, User]{
		//BatchTimeout: 15 * time.Millisecond,
		BatchFn: func(ctx context.Context, keys []int) (map[int]User, error) {
			fmt.Println("fetching...", keys)

			res := make(map[int]User)
			for _, k := range keys {
				// Max ID 10.
				if k > 10 {
					continue
				}
				res[k] = User{ID: k}
			}

			return res, nil
		},
	})

	defer dl.Stop()

	// When using dataloader, we can fetch individual keys in separate goroutines.
	// The keys will be batched and ideally only one query will be made to fetch
	// all the keys.
	for id := range 10 {
		id %= 7
		go func(id int) {
			fmt.Println(dl.Load(id))
		}(id)
	}

	// Manually flush since we know we are not loading any more keys.
	time.Sleep(5 * time.Millisecond)

	// When loading multiple keys, some keys may not return any results.
	// Only keys not present will be fetched.
	res, err := dl.LoadMany([]int{1, 3, 5, 7, 9, 11})
	fmt.Println("LoadMany:", res, err)
	for i, r := range res {
		fmt.Println(i, r, errors.Is(r.Err, dataloader.ErrNoResult))
		if errors.Is(r.Err, dataloader.ErrNoResult) {
			var keyErr *dataloader.KeyError
			if errors.As(r.Err, &keyErr) {
				fmt.Println("key error:", keyErr.Key, keyErr.Unwrap())
			}
		}
	}
	fmt.Println("done")
}
