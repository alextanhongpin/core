package sqltest_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/alextanhongpin/core/storage/sql/sqltest"
)

type mockDB struct {
	sqltest.DBTX
}

func (m *mockDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, nil
}

type db interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type userRepository struct {
	db db
}

func newUserRepository(db db) *userRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, name string) error {
	_, err := r.db.ExecContext(ctx, "insert into users (name) values ($1)", name)
	return err
}

func TestRecorder(t *testing.T) {
	dbr := sqltest.NewRecorder(&mockDB{})
	repo := newUserRepository(dbr)
	err := repo.Create(context.Background(), "Alice")
	if err != nil {
		t.Fatal(err)
	}

	q, args := dbr.Reset()
	{
		want := "insert into users (name) values ($1)"
		if got := q; got != want {
			t.Fatalf("want %s, got %s", want, got)
		}
	}

	{
		want := []any{"Alice"}
		if got := args[0]; got != want[0] {
			t.Fatalf("want %v, got %v", want, got)
		}
	}
}
