// package main demonstrates using json cache.
package main

import (
	"context"
	"errors"
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
	cache *cache.JSON
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		cache: cache.NewJSON(client),
		users: map[int64]*User{
			1: {ID: 1, Name: "John Doe"},
		},
	}
}

func (u *UserRepository) Find(ctx context.Context, id int64) (*User, error) {
	key := fmt.Sprint(id)
	var user *User
	err := u.cache.Load(ctx, key, &user)
	if err == nil {
		slog.Info("cache hit")
		return user, nil
	}
	if !errors.Is(err, cache.ErrNotExist) {
		return nil, err
	}

	slog.Info("cache miss")
	user, ok := u.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}

	if err := u.cache.Store(ctx, key, user, time.Minute); err != nil {
		return nil, err
	}

	return user, nil
}

type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
