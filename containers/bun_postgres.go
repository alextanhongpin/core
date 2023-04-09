package containers

import (
	"database/sql"
	"sync"
	"testing"

	"github.com/DATA-DOG/go-txdb"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/extra/bundebug"
)

var registerBunDB sync.Once

func PostgresBunDB(t *testing.T) *bun.DB {
	registerBunDB.Do(func() {
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
		db.Close()
	})

	return db
}
