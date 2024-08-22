package redistest

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"

	dockertest "github.com/ory/dockertest/v3"
	redis "github.com/redis/go-redis/v9"
)

var c *client
var once sync.Once

func Init(opts ...Option) func() {
	var stop func()
	once.Do(func() {
		var err error
		c, err = newClient(opts...)
		if err != nil {
			panic(err)
		}
		stop = c.close
	})

	return stop
}

func Addr() string {
	if c.addr == "" {
		panic("redistest: InitDB must be called at TestMain")
	}

	return c.addr
}

func New(t *testing.T, opts ...Option) *testClient {
	return newTestClient(t, opts...)
}

type Option func(c *config) error

type config struct {
	Repository string
	Tag        string
}

func newConfig() *config {
	return &config{
		Repository: "redis",
		Tag:        "latest",
	}
}

func (c *config) apply(opts ...Option) error {
	for _, o := range opts {
		if err := o(c); err != nil {
			return err
		}
	}

	return nil
}

func Image(image string) Option {
	return func(c *config) error {
		repo, tag, ok := strings.Cut(image, ":")
		if !ok {
			return fmt.Errorf("invalid image: %q", image)
		}

		c.Repository = repo
		c.Tag = tag
		return nil
	}
}

type client struct {
	cfg   *config
	addr  string
	close func()
}

func newClient(opts ...Option) (*client, error) {
	cfg := newConfig()
	if err := cfg.apply(opts...); err != nil {
		return nil, err
	}

	c := &client{
		cfg: cfg,
	}

	if err := c.init(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *client) Client() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: c.addr,
	})
}

func (c *client) init() error {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return fmt.Errorf("Could not construct pool: %s", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		return fmt.Errorf("Could not connect to Docker: %s", err)
	}

	resource, err := pool.Run(c.cfg.Repository, c.cfg.Tag, nil)
	if err != nil {
		return fmt.Errorf("Could not start resource: %s", err)
	}

	addr := resource.GetHostPort("6379/tcp")
	if err = pool.Retry(func() error {
		db := redis.NewClient(&redis.Options{
			Addr: addr,
		})

		ctx := context.Background()
		return db.Ping(ctx).Err()
	}); err != nil {
		return fmt.Errorf("Could not connect to docker: %s", err)
	}
	c.addr = addr
	c.close = func() {
		// When you're done, kill and remove the container
		if err := pool.Purge(resource); err != nil {
			log.Fatalf("Could not purge resource: %s", err)
		}
	}
	return nil
}

type testClient struct {
	t *testing.T
	c *client
}

func newTestClient(t *testing.T, opts ...Option) *testClient {
	c, err := newClient(opts...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.close)

	return &testClient{
		t: t,
		c: c,
	}
}

func (tc *testClient) Addr() string {
	return tc.c.addr
}

func (tc *testClient) Client() *redis.Client {
	client := tc.c.Client()
	tc.t.Cleanup(func() {
		client.Close()
	})

	return client
}
