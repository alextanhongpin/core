package pgtest

import (
	"database/sql"
	"sync"
	"testing"

	"github.com/DATA-DOG/go-txdb"
	"github.com/alextanhongpin/core/storage/pg"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

var onceBun sync.Once

func BunTx(t *testing.T) *bun.DB {
	t.Helper()

	onceBun.Do(func() {
		// NOTE: We need to run this once to register the sql driver `pg`.
		// Otherwise txdb will not be able to register this driver.
		bunDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(DSN())))
		if err := bunDB.Ping(); err != nil {
			t.Fatalf("failed to ping: %v", err)
		}

		// NOTE: We can close this connection immediately, since we will be
		// creating a new one for every test.
		if err := bunDB.Close(); err != nil {
			t.Fatalf("failed to close bun: %v", err)
		}

		// NOTE: We use `pg` driver, which bun uses instead of `postgres`.
		txdb.Register("bun_txdb", "pg", dsn)
	})

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
		_ = db.Close()
	})

	return db
}

func BunDB(t *testing.T) *bun.DB {
	t.Helper()

	db := pg.NewBun(DSN())

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}
