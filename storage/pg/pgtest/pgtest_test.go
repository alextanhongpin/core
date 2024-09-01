package pgtest_test

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"

	"github.com/alextanhongpin/core/storage/pg/pgtest"
	_ "github.com/lib/pq"
)

var opts = []pgtest.Option{
	pgtest.Image("postgres:15.1-alpine"),
	pgtest.Hook(migrate),
}

func TestMain(m *testing.M) {
	// Start the container once.
	stop := pgtest.Init(opts...)
	defer stop()

	m.Run() // Run tests.
}

func migrate(db *sql.DB) error {
	// Alternatively, you can also usej pgtest.DSN() to get
	// the connection string.
	_, err := db.Exec(`create table numbers(n int)`)
	return err
}

func TestDB(t *testing.T) {
	n := 3
	var wg sync.WaitGroup
	wg.Add(n)

	errs := make([]error, n)

	for i := range n {
		go func() {
			defer wg.Done()

			db := pgtest.DB(t)
			_, err := db.Exec(`insert into numbers(n) values ($1)`, i)
			if err != nil {
				errs[i] = err
			}
		}()
	}

	wg.Wait()

	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	var got int64
	db := pgtest.DB(t)
	if err := db.QueryRow(`select count(*) from numbers`).Scan(&got); err != nil {
		t.Fatal(err)
	}

	if want, got := int64(n), got; want != got {
		t.Fatalf("want %d, got %d", want, got)
	}
	_, err := db.Exec(`truncate table numbers`)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(pgtest.DSN())
}

func TestTx(t *testing.T) {
	for i := range 3 {
		t.Run(fmt.Sprintf("goroutine:%d", i+1), func(t *testing.T) {
			t.Parallel()

			db := pgtest.Tx(t)
			if err := testDB(db, i); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNew(t *testing.T) {
	for i, db := range []*sql.DB{
		pgtest.New(t, opts...).DB(),
		pgtest.New(t, opts...).DB(),
		pgtest.New(t, opts...).Tx(),
		pgtest.New(t, opts...).Tx(),
	} {
		if err := testDB(db, i); err != nil {
			t.Fatal(err)
		}
	}
}

func testDB(db *sql.DB, i int) error {
	_, err := db.Exec(`insert into numbers(n) values ($1)`, i)
	if err != nil {
		return err
	}

	var got int
	if err := db.QueryRow(`select count(*) from numbers`).Scan(&got); err != nil {
		return err
	}

	if want := 1; want != got {
		return fmt.Errorf("want %d, got %d", want, got)
	}
	return nil
}
