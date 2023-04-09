package containers

import (
	"database/sql"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/DATA-DOG/go-txdb"
	"github.com/google/uuid"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

func PostgresBunDB(t *testing.T) *bun.DB {
	// Create a unique transaction for each connection.
	sqldb, err := sql.Open("bun_txdb", uuid.NewString())
	if err != nil {
		t.Errorf("failed to open tx: %v", err)
	}

	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
	))
	t.Cleanup(func() {
		db.Close()
	})

	return db
}

type postgresBunHook func(db *bun.DB) error

func StartPostgresBun(tag string, hooks ...postgresBunHook) func() {
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
	databaseURL := fmt.Sprintf("postgres://john:123456@%s/test?sslmode=disable", hostAndPort)

	log.Println("connecting to database on url:", databaseURL)

	resource.Expire(120) // Tell docker to kill the container in 120 seconds.

	var bunDB *bun.DB
	// Exponential backoff-retry, because the application in the container might
	// not be ready to accept connections yet.
	pool.MaxWait = 120 * time.Second
	if err = pool.Retry(func() error {
		sqlDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(databaseURL)))
		bunDB = bun.NewDB(sqlDB, pgdialect.New())
		if err := bunDB.Ping(); err != nil {
			return fmt.Errorf("failed to ping db: %w", err)
		}

		// Run db migrations, seed etc.
		for _, hook := range hooks {
			if err := hook(bunDB); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		log.Fatal("could not connect to docker:", err)
	}

	// Note the `pg` driver, which bun uses instead of `postgres`.
	txdb.Register("bun_txdb", "pg", databaseURL)

	return func() {
		if err := bunDB.Close(); err != nil {
			log.Println("could not close db:", err)
		}

		if err := pool.Purge(resource); err != nil {
			log.Fatal("could not purge resource:", err)
		}
	}
}
