package containers

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-txdb"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/uptrace/bun/driver/pgdriver"
)

var dsn string

var registerPgDB sync.Once

// PostgresTx runs everything as a single transaction.
// The operations will be rollbacked at the end, reducing the need to manually
// create transactions and rollbacking.
func PostgresTx(t *testing.T) *sql.DB {
	t.Helper()

	registerPgDB.Do(func() {
		txdb.Register("txdb", "postgres", dsn)
	})

	db, err := sql.Open("txdb", uuid.New().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// PostgresDB returns a non-transaction *sql.DB.
// The reason is there is a need for testing stuff that should not be in the
// same transactions, e.g. when generating current_timestamp, or locking in
// different connection.
func PostgresDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	return db
}

type postgresHook func(db *sql.DB) error

func StartPostgres(tag string, hooks ...postgresHook) func() {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatal("could not construct pool:", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		log.Fatal("could not connect to docker:", err)
	}

	// Pulls an image, creates a container based on it and run it.
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
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
		log.Fatal("could not start resources:", err)
	}
	hostAndPort := resource.GetHostPort("5432/tcp")
	dsn = fmt.Sprintf("postgres://john:123456@%s/test?sslmode=disable", hostAndPort)

	log.Println("connecting to database on url:", dsn)

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
		for _, hook := range hooks {
			if err := hook(db); err != nil {
				return err
			}
		}

		// NOTE: We need to run this once to register the sql driver `pg`.
		// Otherwise txdb will not be able to register this driver.
		bunDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		if err := bunDB.Ping(); err != nil {
			return fmt.Errorf("failed to ping: %w", err)
		}

		// NOTE: We can close this connection immediately, since we will be
		// creating a new one for every test.
		if err := bunDB.Close(); err != nil {
			return fmt.Errorf("failed to close bun: %w", err)
		}

		if err := db.Close(); err != nil {
			return fmt.Errorf("failed to close db: %w", err)
		}

		return nil
	}); err != nil {
		log.Fatal("could not connect to docker:", err)
	}

	return func() {
		if err := pool.Purge(resource); err != nil {
			log.Fatal("could not purge resource:", err)
		}
	}
}
