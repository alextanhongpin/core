package testutil_test

import (
	"context"
	"database/sql"
	"math/rand"
	"testing"
	"time"

	"github.com/alextanhongpin/core/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDumpMySQL(t *testing.T) {
	type User struct {
		ID   int64
		Name string
	}

	simpleDump := &testutil.SQL{
		Query:  `select * from users where id = ? and deleted_at IS NULL`,
		Args:   []any{uuid.New()},
		Result: User{ID: 1, Name: "Alice"},
	}

	complexDump := &testutil.SQL{
		Query: `SELECT *
     FROM users
     WHERE email = 'john.doe@mail.com'
     AND deleted_at IS NULL
     AND last_logged_in_at > ?
		 AND created_at IN (?) 
     AND description = 'foo bar walks in a bar, h''a'
     AND subscription in ('freemium', 'premium')
		 AND age > 13
		 AND is_active = true
 		 AND name LIKE ANY('{Foo,bar,%oo%}')`,
		Args: []any{time.Now().Format("2006-01-02")},
		Result: []User{
			{ID: rand.Int63(), Name: "Alice"},
			{ID: rand.Int63(), Name: "Bob"},
		},
	}

	t.Run("simple", func(t *testing.T) {
		testutil.DumpMySQL(t, simpleDump,
			testutil.IgnoreArgs("v1"),
		)
	})

	t.Run("complex", func(t *testing.T) {
		testutil.DumpMySQL(t, complexDump,
			testutil.IgnoreArgs("v1"),
			testutil.IgnoreResultFields("id"),
		)
	})
}

func TestMySQLRepository(t *testing.T) {
	assert := assert.New(t)
	db := newMockDB(t)
	dbtx := &mysqlDBHook{
		t:    t,
		dbtx: db,
		opts: []testutil.SQLOption{
			testutil.SQLFileName("find_user"),
		},
	}
	repo := newMockUserRepository(dbtx, "mysql")
	user, err := repo.FindUser(context.Background(), "1")
	assert.Nil(err)
	assert.Equal("1", user.ID)
	assert.Equal("Alice", user.Name)
	testutil.DumpYAML(t, user)
}

type mysqlDBHook struct {
	dbtx
	t    *testing.T
	opts []testutil.SQLOption
}

func (h *mysqlDBHook) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	testutil.DumpMySQL(h.t, testutil.NewSQL(query, args, nil), h.opts...)

	return h.dbtx.QueryRowContext(ctx, query, args...)
}
