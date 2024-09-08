package pgtest

import (
	"database/sql"
	"errors"
	"fmt"
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

var Error = errors.New("pgtest")

var id atomic.Int64

func nextID() int64 {
	return id.Add(1)
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
		if err := c.hook(); err != nil {
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
	return c.Tx(t)
}

// DB returns a non-transaction *sql.DB.
// The reason is there is a need for testing stuff that should not be in the
// same transactions, e.g. when generating current_timestamp, or locking in
// different connection.
func DB(t *testing.T) *sql.DB {
	return c.DB(t)
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

// Expire sets the duration for the docker to hard-kill the postgres image.
func Expire(duration time.Duration) func(*config) error {
	return func(cfg *config) error {
		cfg.Expire = duration

		return nil
	}
}

func Image(image string) func(*config) error {
	return func(cfg *config) error {
		repo, tag, ok := strings.Cut(image, ":")
		if !ok {
			return newError("invalid docker image format: %s", image)
		}
		cfg.Repository = repo
		cfg.Tag = tag

		return nil
	}
}

type config struct {
	Repository string
	Tag        string
	Expire     time.Duration
	Hook       func(*sql.DB) error
}

func newConfig() *config {
	return &config{
		Repository: "postgres",
		Tag:        "latest",
		Expire:     10 * time.Minute,
		Hook: func(*sql.DB) error {
			return nil
		},
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

type client struct {
	cfg   *config
	dsn   string
	close func()
	once  sync.Once
	txdb  string
}

func newClient(opts ...Option) (*client, error) {
	cfg := newConfig()
	if err := cfg.apply(opts...); err != nil {
		return nil, err
	}

	c := &client{cfg: cfg}
	if err := c.init(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *client) hook() error {
	return c.cfg.Hook(pg.New(c.dsn))
}

func (c *client) init() error {
	var (
		repo = c.cfg.Repository
		tag  = c.cfg.Tag
	)

	pool, err := dockertest.NewPool("")
	if err != nil {
		return newError("could not construct pool: %s", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		return newError("could not connect to docker: %s", err)
	}

	// Pulls an image, creates a container based on it and run it.
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: repo,
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
		return newError("could not start resources: %s", err)
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
		return newError("exec code is not 1")
	}

	hostAndPort := resource.GetHostPort("5432/tcp")
	c.dsn = fmt.Sprintf("postgres://john:123456@%s/test?sslmode=disable", hostAndPort)

	resource.Expire(uint(c.cfg.Expire.Seconds())) // Tell docker to kill the container in 120 seconds.

	// Exponential backoff-retry, because the application in the container might
	// not be ready to accept connections yet.
	pool.MaxWait = 120 * time.Second
	if err := pool.Retry(func() error {
		db, err := sql.Open("postgres", c.dsn)
		if err != nil {
			return newError("failed to open: %s", err)
		}

		if err := db.Ping(); err != nil {
			return newError("failed to ping: %s", err)
		}

		if err := db.Close(); err != nil {
			return newError("failed to close db: %s", err)
		}

		return nil
	}); err != nil {
		return newError("could not connect to docker: %s", err)
	}

	c.close = func() {
		if err := pool.Purge(resource); err != nil {
			panic(newError("could not purge resource: %s", err))
		}
	}

	return nil
}

func (c *client) DSN() string {
	return c.dsn
}

func (c *client) DB(t *testing.T) *sql.DB {
	db := pg.New(c.dsn)
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func (c *client) Tx(t *testing.T) *sql.DB {
	t.Helper()

	// Lazily initialize the txdb.
	c.once.Do(func() {
		c.txdb = fmt.Sprintf("txdb%d", nextID())
		txdb.Register(c.txdb, "postgres", c.dsn)
	})

	// Returns a new identifier for every open connection.
	db, err := sql.Open(c.txdb, uuid.New().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

type testClient struct {
	t *testing.T
	c *client
}

func newTestClient(t *testing.T, opts ...Option) *testClient {
	t.Helper()
	c, err := newClient(opts...)
	if err != nil {
		t.Fatal(err)
	}

	if err := c.hook(); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(c.close)

	return &testClient{
		t: t,
		c: c,
	}
}

func (tc *testClient) DB() *sql.DB {
	return tc.c.DB(tc.t)
}

func (tc *testClient) Tx() *sql.DB {
	return tc.c.Tx(tc.t)
}

func (tc *testClient) DSN() string {
	return tc.c.dsn
}

func newError(msg string, args ...any) error {
	return fmt.Errorf("%w: %s", Error, fmt.Sprintf(msg, args...))
}
