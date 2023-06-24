package testutil_test

import (
	"testing"

	"github.com/alextanhongpin/core/test/testutil"
)

func TestMySQLDumper(t *testing.T) {
	stmt := `
		select * 
		from users 
		where id = 1 
		and age = ? 
		and status in ('pending', 'success') 
		and subscription in ('gold', 'silver') 
		and created_at > '2023-01-01' 
		order by age desc 
		limit ?
	`

	testutil.DumpSQL(t,
		testutil.NewSQLDump(stmt, []any{13, 10}, nil),
		testutil.MySQL(),
		testutil.Parameterize(),
	)
}
