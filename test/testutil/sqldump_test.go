package testutil_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/test/testutil"
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

		testutil.DumpSQL(t, dump)
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
 		 LIKE ANY('{Foo,bar,%oo%}')`

		dump := &testutil.SQLDump{
			Stmt: stmt,
			Args: []any{time.Now().Format("2006-01-02")},
		}

		testutil.DumpSQL(t, dump)
	})
}

func TestSQLDumpParameterize(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		type User struct {
			ID   int64
			Name string
		}

		dump := &testutil.SQLDump{
			Stmt: `select * from users where name = 'John' and age = 13`,
			Args: nil,
			Rows: []User{{ID: 1, Name: "John"}},
		}

		testutil.DumpSQL(t, dump, testutil.Parameterize())
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
 		 LIKE ANY('{Foo,bar,%oo%}')`

		dump := &testutil.SQLDump{
			Stmt: stmt,
			Args: []any{time.Now().Format("2006-01-02")},
		}

		testutil.DumpSQL(t, dump, testutil.Parameterize())
	})
}
