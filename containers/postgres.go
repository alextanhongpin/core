package containers

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/DATA-DOG/go-txdb"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

func DB() (*sql.DB, error) {
	return sql.Open("txdb", uuid.New().String())
}

func Postgres(tag string) func() {
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

	var db *sql.DB
	// Exponential backoff-retry, because the application in the container might
	// not be ready to accept connections yet.
	pool.MaxWait = 120 * time.Second
	if err = pool.Retry(func() error {
		db, err = sql.Open("postgres", databaseURL)
		if err != nil {
			return err
		}

		return db.Ping()
	}); err != nil {
		log.Fatal("could not connect to docker:", err)
	}
	txdb.Register("txdb", "postgres", databaseURL)

	return func() {
		if err := db.Close(); err != nil {
			log.Println("could not close db:", err)
		}

		if err := pool.Purge(resource); err != nil {
			log.Fatal("could not purge resource:", err)
		}
	}
}
