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
	"github.com/uptrace/bun/extra/bundebug"
)

var onceBun sync.Once

func BunTx(t *testing.T) *bun.DB {
	onceBun.Do(func() {
		// Note the `pg` driver, which bun uses instead of `postgres`.
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
	db := pg.NewBun(dsn)

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}
