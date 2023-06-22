package testutil_test

import (
	"testing"

	"github.com/alextanhongpin/core/test/testutil"
)

func TestSQLDump(t *testing.T) {
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
}
