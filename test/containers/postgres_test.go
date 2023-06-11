package containers_test

import (
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/test/containers"
	_ "github.com/lib/pq"
)

func TestPostgresDB(t *testing.T) {
	db := containers.PostgresDB(t)

	var n int
	err := db.QueryRow("select 1 + 1").Scan(&n)
	if err != nil {
		t.Error(err)
	}

	want := 2
	got := n
	if want != got {
		t.Fatalf("sum: want %d, got %d", want, got)
	}
}

func TestPostgresDBIsolation(t *testing.T) {
	n := 3
	for i := 0; i < n; i++ {
		i := i
		t.Run(fmt.Sprintf("go:%d", i+1), func(t *testing.T) {
			t.Parallel()

			db := containers.PostgresDB(t)
			_, err := db.Exec(`insert into numbers(n) values ($1)`, i)
			if err != nil {
				t.Fatal(err)
			}
			var j int
			if err := db.QueryRow(`select count(*) from numbers`).Scan(&j); err != nil {
				t.Fatal(err)
			}
			t.Logf("j=%d", j)
		})
	}
}

func TestPostgresTx(t *testing.T) {
	db := containers.PostgresTx(t)

	var n int
	err := db.QueryRow("select 1 + 1").Scan(&n)
	if err != nil {
		t.Error(err)
	}

	want := 2
	got := n
	if want != got {
		t.Fatalf("sum: want %d, got %d", want, got)
	}
}

func TestPostgresTxIsolation(t *testing.T) {
	n := 3
	for i := 0; i < n; i++ {
		i := i
		t.Run(fmt.Sprintf("go:%d", i+1), func(t *testing.T) {
			t.Parallel()

			db := containers.PostgresTx(t)
			_, err := db.Exec(`insert into names(name) values ($1)`, fmt.Sprint(i))
			if err != nil {
				t.Fatal(err)
			}
			var j int
			if err := db.QueryRow(`select count(*) from names`).Scan(&j); err != nil {
				t.Fatal(err)
			}
			t.Logf("j=%d", j)
		})
	}
}
