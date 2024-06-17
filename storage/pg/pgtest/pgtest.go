package pgtest

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-txdb"
	"github.com/alextanhongpin/core/storage/pg"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var dsn string

var once sync.Once

func DSN() string {
	return dsn
}

// Tx runs everything as a single transaction.
// The operations will be rollbacked at the end, reducing the need to manually
// create transactions and rollbacking.
func Tx(t *testing.T) *sql.DB {
	t.Helper()

	// Lazily initialize the txdb.
	once.Do(func() {
		txdb.Register("txdb", "postgres", DSN())
	})

	db, err := sql.Open("txdb", uuid.New().String())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

// DB returns a non-transaction *sql.DB.
// The reason is there is a need for testing stuff that should not be in the
// same transactions, e.g. when generating current_timestamp, or locking in
// different connection.
func DB(t *testing.T) *sql.DB {
	t.Helper()

	db := pg.New(DSN())
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

type Option interface {
	isOption()
}

type Hook func(*sql.DB) error

func (h Hook) isOption() {}

type Image string

func (t Image) isOption() {}

func InitDB(opts ...Option) func() {
	stop, err := initDB(opts...)
	if err != nil {
		panic(err)
	}

	return stop
}

func initDB(opts ...Option) (func(), error) {
	var fn func(*sql.DB) error
	image := "postgres"
	tag := "latest"

	for _, opt := range opts {
		switch v := opt.(type) {
		case Image:
			var ok bool
			image, tag, ok = strings.Cut(string(v), ":")
			if !ok {
				tag = "latest"
			}
		case Hook:
			fn = v
		}
	}
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("could not construct pool: %w", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		return nil, fmt.Errorf("could not connect to docker: %w", err)
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
		return nil, fmt.Errorf("could not start resources: %w", err)
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
		return nil, err
	}
	if code != 1 {
		return nil, errors.New("exec code is not 1")
	}

	hostAndPort := resource.GetHostPort("5432/tcp")
	dsn = fmt.Sprintf("postgres://john:123456@%s/test?sslmode=disable", hostAndPort)

	resource.Expire(120) // Tell docker to kill the container in 120 seconds.

	var db *sql.DB
	// Exponential backoff-retry, because the application in the container might
	// not be ready to accept connections yet.
	pool.MaxWait = 120 * time.Second
	if err = pool.Retry(func() error {
		db, err = sql.Open("postgres", dsn)
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
		return nil, fmt.Errorf("could not connect to docker: %w", err)
	}

	return func() {
		if err := pool.Purge(resource); err != nil {
			log.Fatal("could not purge resource:", err)
		}
	}, nil
}
