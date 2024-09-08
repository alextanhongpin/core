package pg

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"time"

	_ "github.com/lib/pq"
)

type Params map[string]string

type Option struct {
	User     string
	Password string
	Host     string
	Port     string
	Database string
	Params   Params
}

func (o Option) DSN() string {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", o.User, o.Password, o.Host, o.Port, o.Database)

	u := url.Values{}
	for k, v := range o.Params {
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
