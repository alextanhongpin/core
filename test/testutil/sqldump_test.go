package testutil_test

import (
	"testing"

	"github.com/alextanhongpin/core/test/testutil"
	"github.com/google/uuid"
)

func TestSQLDump(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		type User struct {
			ID   int64
			Name string
		}

		dump := &testutil.SQLDump{
			Stmt: `select * from users where name = $1 and age = $2`,
			Args: []any{"John", 13},
			Rows: []User{{ID: 1, Name: "John"}},
		}

		testutil.DumpSQL(t, dump, testutil.Postgres())
	})

	t.Run("complex", func(t *testing.T) {
		stmt := `SELECT *
     FROM users
     WHERE email = 'john.doe@mail.com'
     AND deleted_at IS NULL
     AND last_logged_in_at > $1
     AND description = 'foo bar walks in a bar, h''a'
     AND subscription in ('freemium', 'premium')
		 AND age > 13
		 AND is_active = true
 		 AND name LIKE ANY('{Foo,bar,%oo%}')`

		dump := &testutil.SQLDump{
			Stmt: stmt,
			Args: []any{"2023-06-23"},
		}

		testutil.DumpSQL(t, dump, testutil.Postgres())
	})
}

func TestSQLDumpParameterize(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		type User struct {
			ID   int64
			Name string
		}

		dump := testutil.NewSQLDump(
			`select * from users where name = 'John' and age = 13`,
			nil,
			[]User{{ID: 1, Name: "John"}},
		)

		testutil.DumpSQL(t, dump, testutil.Postgres(), testutil.Normalize())
	})

	t.Run("complex", func(t *testing.T) {
		stmt := `SELECT *
     FROM users
     WHERE email = 'john.doe@mail.com'
     AND deleted_at IS NULL
     AND last_logged_in_at > $1
     AND description = 'foo bar walks in a bar, h''a'
     AND subscription in ('freemium', 'premium')
		 AND age > 13
		 AND is_active = true
 		 AND name LIKE ANY('{Foo,bar,%oo%}')`

		dump := &testutil.SQLDump{
			Stmt: stmt,
			Args: []any{"2023-06-23"},
		}

		testutil.DumpSQL(t, dump, testutil.Postgres(), testutil.Normalize())
	})

	t.Run("skip comparison", func(t *testing.T) {
		stmt := `select * from users where id = $1`
		dump := testutil.NewSQLDump(
			stmt,
			[]any{uuid.New()},
			nil,
		)

		// Args are mapped to $1, $2 ...
		testutil.DumpSQL(t, dump, testutil.Postgres(), testutil.IgnoreFields("$1"))
	})
}
