package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"
)

func New() *sql.DB {
	user := mustEnv("DB_USER")
	pass := mustEnv("DB_PASS")
	host := mustEnv("DB_HOST")
	port := mustEnv("DB_PORT")
	name := mustEnv("DB_NAME")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, name)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("failed to open db:", err)
	}

	// https://www.alexedwards.net/blog/configuring-sqldb
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(1 * time.Hour)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatal("failed to ping db:", err)
	}

	return db
}

func mustEnv(name string) string {
	env, ok := os.LookupEnv(name)
	if !ok {
		log.Fatalf("%q not set", name)
	}

	return env
}
