package containers_test

import (
	"testing"

	"github.com/alextanhongpin/core/containers"
)

func TestPostgres(t *testing.T) {
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
