// package main demonstrates using json cache.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/alextanhongpin/core/dsync/cache"
	"github.com/redis/go-redis/v9"
)

var client *redis.Client

func init() {
	client = redis.NewClient(&redis.Options{
		Addr: ":6379",
	})
	fmt.Println(client.FlushAll(context.Background()))
}

func main() {
	ctx := context.Background()
	repo := NewUserRepository()
	var wg sync.WaitGroup
	wg.Add(10)
	for range 10 {
		go func() {
			defer wg.Done()

			user, err := repo.Find(ctx, 1)
			fmt.Println(user, err)
		}()
	}

	wg.Wait()
}

type UserRepository struct {
	users map[int64]*User
	cache cache.Cacheable
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		cache: cache.New(client),
		users: map[int64]*User{
			1: {ID: 1, Name: "John Doe"},
		},
	}
}

func (u *UserRepository) Find(ctx context.Context, id int64) (*User, error) {
	user, loaded, err := cache.LoadOrStore(ctx, u.cache, fmt.Sprint(id), func() (*User, error) {
		slog.Info("loading user from database", "id", id)
		user, ok := u.users[id]
		if !ok {
			return nil, fmt.Errorf("user not found: %d", id)
		}
		return user, nil
	}, time.Minute)
	if loaded {
		slog.Info("user loaded from cache", "id", id)
	} else {
		slog.Info("user loaded from database", "id", id)
	}
	return user, err
}

type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
