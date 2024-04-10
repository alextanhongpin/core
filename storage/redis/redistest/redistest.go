package redistest

import (
	"context"
	"fmt"
	"log"

	"github.com/ory/dockertest/v3"
	"github.com/redis/go-redis/v9"
)

var addr string

func Addr() string {
	if addr == "" {
		panic("redistest: InitDB must be called at TestMain")
	}
	return addr
}

type Option interface {
	isOption()
}

type Tag string

func (t Tag) isOption() {}

func Init(opts ...Option) func() {
	stop, err := initClient(opts...)
	if err != nil {
		panic(err)
	}

	return stop
}

func initClient(opts ...Option) (func(), error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("Could not construct pool: %s", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		return nil, fmt.Errorf("Could not connect to Docker: %s", err)
	}

	tag := Tag("7.2.4")
	for _, opt := range opts {
		switch v := opt.(type) {
		case Tag:
			if v != "" {
				tag = v
			}
		default:
			return nil, fmt.Errorf("unsupported option: %v", v)
		}

	}

	resource, err := pool.Run("redis", string(tag), nil)
	if err != nil {
		return nil, fmt.Errorf("Could not start resource: %s", err)
	}

	if err = pool.Retry(func() error {
		addr = resource.GetHostPort("6379/tcp")
		db := redis.NewClient(&redis.Options{
			Addr: addr,
		})

		ctx := context.Background()
		return db.Ping(ctx).Err()
	}); err != nil {
		return nil, fmt.Errorf("Could not connect to docker: %s", err)
	}

	return func() {
		// When you're done, kill and remove the container
		if err := pool.Purge(resource); err != nil {
			log.Fatalf("Could not purge resource: %s", err)
		}
	}, nil
}
