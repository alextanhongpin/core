package containers_test

import (
	"context"
	"testing"

	"github.com/alextanhongpin/go-core-microservice/containers"
	_ "github.com/uptrace/bun/dialect/pgdialect"
)

func TestBunPostgres(t *testing.T) {
	db := containers.PostgresBunDB(t)
	var n int

	ctx := context.Background()
	err := db.NewRaw(`select 1 + 1`).Scan(ctx, &n)
	if err != nil {
		t.Error(err)
	}

	want := 2
	got := n
	if want != got {
		t.Fatalf("sum: want %d, got %d", want, got)
	}
}
