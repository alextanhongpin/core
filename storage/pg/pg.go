package pg

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"time"
)

func DSN(user, pass, host, port, name string, opts map[string]string) string {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, pass, host, port, name)
	u := url.Values{}
	for k, v := range opts {
		u.Set(k, v)
	}

	if q := u.Encode(); q != "" {
		dsn = fmt.Sprintf("%s?%s", dsn, q)
	}

	return dsn
}

func New(dsn string) *sql.DB {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping db: %v", err)
	}

	applyDefaults(db)

	return db
}

func applyDefaults(db *sql.DB) {
	// https://www.alexedwards.net/blog/configuring-sqldb
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(1 * time.Hour)
	db.SetConnMaxIdleTime(5 * time.Minute)
}
