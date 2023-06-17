package pg_test

import (
	"testing"

	"github.com/alextanhongpin/core/storage/pg"
)

func TestDSN(t *testing.T) {
	got := pg.Option{
		User:     "john",
		Password: "123456",
		Host:     "127.0.0.1",
		Port:     "5432",
		Database: "test",
		Params: pg.Params{
			"sslmode": "disable",
		},
	}.DSN()
	want := "postgres://john:123456@127.0.0.1:5432/test?sslmode=disable"
	if want != got {
		t.Fatalf("want %s, got %s", want, got)
	}
}
