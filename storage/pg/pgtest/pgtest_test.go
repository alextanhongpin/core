package pgtest_test

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/alextanhongpin/core/storage/pg/pgtest"
	_ "github.com/lib/pq"
)

func TestMain(m *testing.M) {
	// Start the container.
	stop := pgtest.InitDB(pgtest.Hook(migrate), pgtest.Tag("15.1-alpine"))
	code := m.Run() // Run tests.
	stop()          // You can't defer this because os.Exit doesn't care for defer.
	os.Exit(code)
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`create table numbers(n int)`)
	return err
}

func TestDB(t *testing.T) {
	db := pgtest.DB(t)

	var got int
	if err := db.QueryRow("select 1 + 1").Scan(&got); err != nil {
		t.Fatal(err)
	}

	if want := 2; want != got {
		t.Fatalf("sum: want %d, got %d", want, got)
	}
}

func TestTx(t *testing.T) {
	n := 3
	for i := 0; i < n; i++ {
		i := i
		t.Run(fmt.Sprintf("goroutine:%d", i+1), func(t *testing.T) {
			t.Parallel()

			db := pgtest.Tx(t)
			_, err := db.Exec(`insert into numbers(n) values ($1)`, i)
			if err != nil {
				t.Fatal(err)
			}

			var got int
			if err := db.QueryRow(`select count(*) from numbers`).Scan(&got); err != nil {
				t.Fatal(err)
			}

			if want := 1; want != got {
				t.Fatalf("want %d, got %d", want, got)
			}
		})
	}
}
