package testutil_test

import (
	"context"
	"database/sql"
	"log"
	"math/rand"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alextanhongpin/core/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDumpPostgres(t *testing.T) {
	type User struct {
		ID   int64
		Name string
	}

	simpleDump := &testutil.SQL{
		Query:  `select * from users where id = $1 and deleted_at IS NULL`,
		Args:   []any{uuid.New()},
		Result: User{ID: 1, Name: "Alice"},
	}

	complexDump := &testutil.SQL{
		Query: `SELECT *
     FROM users
     WHERE email = 'john.doe@mail.com'
     AND deleted_at IS NULL
     AND last_logged_in_at > $1
		 AND created_at IN ($2) 
     AND description = 'foo bar walks in a bar, h''a'
     AND subscription in ('freemium', 'premium')
		 AND age > 13
		 AND is_active = true
 		 AND name LIKE ANY('{Foo,bar,%oo%}')
		 AND id <> ALL(ARRAY[1, 2])
		`,
		Args: []any{time.Now().Format("2006-01-02")},
		Result: []User{
			{ID: rand.Int63(), Name: "Alice"},
			{ID: rand.Int63(), Name: "Bob"},
		},
	}

	t.Run("simple", func(t *testing.T) {
		testutil.DumpPostgres(t, simpleDump,
			testutil.IgnoreArgs("$1"),
		)
	})

	t.Run("complex", func(t *testing.T) {
		testutil.DumpPostgres(t, complexDump,
			testutil.IgnoreArgs("$1"),
			testutil.IgnoreResultFields("id"),
		)
	})
}

func TestPostgresRepository(t *testing.T) {
	assert := assert.New(t)
	db := newMockDB(t)
	dbtx := &postgresDBHook{
		t:    t,
		dbtx: db,
		opts: []testutil.SQLOption{
			testutil.FileName("find_user"),
		},
	}
	repo := newMockUserRepository(dbtx, "postgres")
	user, err := repo.FindUser(context.Background(), "1")
	assert.Nil(err)
	assert.Equal("1", user.ID)
	assert.Equal("Alice", user.Name)
	testutil.DumpYAML(t, user)
}

func newMockDB(t *testing.T) *sql.DB {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
	})
	rows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow("1", "Alice")
	mock.ExpectQuery("select(.+)").WillReturnRows(rows)

	return db
}

type mockUserRepository struct {
	// Use this instead of *sql.DB or *sql.Tx to allow
	// interception etc.
	db      dbtx
	dialect string
}

func newMockUserRepository(db dbtx, dialect string) *mockUserRepository {
	return &mockUserRepository{
		db:      db,
		dialect: dialect,
	}
}

func (r *mockUserRepository) FindUser(ctx context.Context, id string) (*User, error) {
	var u User
	if err := r.db.QueryRowContext(ctx, r.query(), id).Scan(&u.ID, &u.Name); err != nil {
		return nil, err
	}

	return &u, nil
}

func (r *mockUserRepository) query() string {
	switch r.dialect {
	case "postgres":
		return `select * from users where id = $1`
	case "mysql":
		return `select * from users where id = ?`
	default:
		log.Fatalf("unknown dialect: %s", r.dialect)
		return ""
	}
}

type User struct {
	ID   string
	Name string
}

type dbtx interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type postgresDBHook struct {
	dbtx
	t    *testing.T
	opts []testutil.SQLOption
}

func (h *postgresDBHook) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	testutil.DumpPostgres(h.t, testutil.NewSQL(query, args, nil), h.opts...)

	return h.dbtx.QueryRowContext(ctx, query, args...)
}
