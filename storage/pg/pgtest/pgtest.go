package pgtest

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	txdb "github.com/DATA-DOG/go-txdb"
	"github.com/alextanhongpin/core/storage/pg"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	dockertest "github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var ids atomic.Int64

func nextID() int64 {
	return ids.Add(1)
}

var once sync.Once
var c *client

func Init(opts ...Option) func() {
	var fn = func() {}

	once.Do(func() {
		var err error
		c, err = newClient(opts...)
		if err != nil {
			panic(err)
		}

		fn = c.close
	})

	return fn
}

func New(t *testing.T, opts ...Option) *testClient {
	return newTestClient(t, opts...)
}

// Tx runs everything as a single transaction.
// The operations will be rollbacked at the end, reducing the need to manually
// create transactions and rollbacking.
func Tx(t *testing.T) *sql.DB {
	db := c.Tx()
	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// DB returns a non-transaction *sql.DB.
// The reason is there is a need for testing stuff that should not be in the
// same transactions, e.g. when generating current_timestamp, or locking in
// different connection.
func DB(t *testing.T) *sql.DB {
	db := c.DB()
	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func DSN() string {
	return c.dsn
}

type Option func(*config) error

func Hook(fn func(*sql.DB) error) func(*config) error {
	return func(cfg *config) error {
		cfg.Hook = fn

		return nil
	}
}

func Image(image string) func(*config) error {
	return func(cfg *config) error {
		image, tag, ok := strings.Cut(image, ":")
		if !ok {
			return fmt.Errorf("pgtest: invalid Image(%q) format", image)
		}
		cfg.Image = image
		cfg.Tag = tag

		return nil
	}
}

type config struct {
	Image string
	Tag   string
	Hook  func(*sql.DB) error
}

func newConfig() *config {
	return &config{
		Image: "postgres",
		Tag:   "latest",
	}
}

func (c *config) apply(opts ...Option) *config {
	for _, o := range opts {
		o(c)
	}
	return c

}

type client struct {
	cfg   *config
	dsn   string
	close func()
	once  sync.Once
	txdb  string
}

func newClient(opts ...Option) (*client, error) {
	cfg := newConfig().apply(opts...)
	c := &client{cfg: cfg}
	if err := c.init(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *client) init() error {
	var (
		image = c.cfg.Image
		tag   = c.cfg.Tag
		fn    = c.cfg.Hook
	)

	pool, err := dockertest.NewPool("")
	if err != nil {
		return fmt.Errorf("could not construct pool: %w", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		return fmt.Errorf("could not connect to docker: %w", err)
	}

	// Pulls an image, creates a container based on it and run it.
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: image,
		Tag:        tag,
		Env: []string{
			"POSTGRES_PASSWORD=123456",
			"POSTGRES_USER=john",
			"POSTGRES_DB=test",
			"listen_addresses = '*'",
		},
	}, func(config *docker.HostConfig) {
		// Set AutoRemove to true so that stopped container goes away by itself.
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return fmt.Errorf("could not start resources: %w", err)
	}

	// https://www.postgresql.org/docs/current/non-durability.html
	code, err := resource.Exec([]string{"postgres",
		// No need to flush data to disk.
		"-c", "fsync=off",

		// No need to force WAL writes to disk on every commit.
		"-c", "synchronous_commit=off",

		// No need to guard against partial page writes.
		"-c", "full_page_writes=off",
	}, dockertest.ExecOptions{})
	if err != nil {
		return err
	}
	if code != 1 {
		return errors.New("exec code is not 1")
	}

	hostAndPort := resource.GetHostPort("5432/tcp")
	dsn := fmt.Sprintf("postgres://john:123456@%s/test?sslmode=disable", hostAndPort)

	resource.Expire(120) // Tell docker to kill the container in 120 seconds.

	// Exponential backoff-retry, because the application in the container might
	// not be ready to accept connections yet.
	pool.MaxWait = 120 * time.Second
	if err := pool.Retry(func() error {
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			return fmt.Errorf("failed to open: %w", err)
		}

		if err := db.Ping(); err != nil {
			return fmt.Errorf("failed to ping: %w", err)
		}

		// Run migrations, seed, fixtures etc.
		if err := fn(db); err != nil {
			return err
		}

		if err := db.Close(); err != nil {
			return fmt.Errorf("failed to close db: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("could not connect to docker: %w", err)
	}

	c.dsn = dsn
	c.close = func() {
		if err := pool.Purge(resource); err != nil {
			log.Fatal("could not purge resource:", err)
		}
	}

	return nil
}

func (c *client) DSN() string {
	return c.dsn
}

func (c *client) DB() *sql.DB {
	return pg.New(c.dsn)
}

func (c *client) Tx() *sql.DB {
	// Lazily initialize the txdb.
	c.once.Do(func() {
		c.txdb = fmt.Sprintf("txdb%d", nextID())
		txdb.Register(c.txdb, "postgres", c.dsn)
	})

	// Returns a new identifier for every open connection.
	db, err := sql.Open(c.txdb, uuid.New().String())
	if err != nil {
		panic(err)
	}

	return db
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

	return &testClient{
		t: t,
		c: c,
	}
}

func (tc *testClient) DB() *sql.DB {
	db := tc.c.DB()
	tc.t.Cleanup(func() {
		db.Close()
	})
	return db
}

func (tc *testClient) Tx() *sql.DB {
	db := tc.c.Tx()
	tc.t.Cleanup(func() {
		db.Close()
	})
	return db
}

func (tc *testClient) DSN() string {
	return tc.c.dsn
}
