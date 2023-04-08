package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"
)

func New() *sql.DB {
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	name := os.Getenv("DB_NAME")

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
