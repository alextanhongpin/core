package pgtest_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/storage/pg/pgtest"
)

func TestBunDB(t *testing.T) {
	ctx := context.Background()

	db := pgtest.BunDB(t)

	var got int
	if err := db.NewRaw("select 1 + 1").Scan(ctx, &got); err != nil {
		t.Fatal(err)
	}

	if want := 2; want != got {
		t.Fatalf("sum: want %d, got %d", want, got)
	}
}

func TestBunTx(t *testing.T) {
	ctx := context.Background()

	n := 3
	for i := 0; i < n; i++ {
		i := i
		t.Run(fmt.Sprintf("goroutine:%d", i+1), func(t *testing.T) {
			t.Parallel()

			db := pgtest.BunTx(t)
			_, err := db.NewRaw(`insert into numbers(n) values (?)`, i).Exec(ctx)
			if err != nil {
				t.Fatal(err)
			}

			var got int
			if err := db.NewRaw(`select count(*) from numbers`).Scan(ctx, &got); err != nil {
				t.Fatal(err)
			}

			if want := 1; want != got {
				t.Fatalf("want %d, got %d", want, got)
			}
		})
	}
}
