package main

import (
	"context"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/dataloader"
)

func main() {
	type User struct {
		ID string
	}

	dl := dataloader.New(dataloader.Option[string, User]{
		//BatchTimeout: 15 * time.Millisecond,
		BatchFn: func(ctx context.Context, keys []string) ([]User, error) {
			fmt.Println("fetching...", keys)

			res := make([]User, len(keys))
			for i, k := range keys {
				res[i] = User{
					ID: k,
				}
			}

			return res, nil
		},
		KeyFn: func(u User) (string, error) {
			return u.ID, nil
		},
		PromiseFn: dataloader.Copier[User],
	})

	ctx := context.Background()
	go func() {
		u := dl.Load(ctx, "1")
		fmt.Println(u.Await())
	}()

	go func() {
		u := dl.Load(ctx, "2")
		fmt.Println(u.Await())
	}()

	time.Sleep(5 * time.Millisecond)

	dl.Stop()

	u := dl.LoadMany(ctx, []string{"1", "2", "3"})
	fmt.Println("done")
	fmt.Println(u.Await())
}
