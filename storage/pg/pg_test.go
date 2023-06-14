package pg_test

import (
	"testing"

	"github.com/alextanhongpin/core/storage/pg"
)

func TestDSN(t *testing.T) {
	got := pg.DSN("john", "123456", "127.0.0.1", "5432", "test", map[string]string{
		"sslmode": "disable",
	})
	want := "postgres://john:123456@127.0.0.1:5432/test?sslmode=disable"
	if want != got {
		t.Fatalf("want %s, got %s", want, got)
	}
}
