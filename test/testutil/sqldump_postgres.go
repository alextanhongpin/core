package testutil

import (
	"testing"

	"github.com/alextanhongpin/core/test/testdump"
)

type PostgresDumpOption = testdump.PostgresOption

type PostgresOption struct {
	Dump     *PostgresDumpOption
	FileName string
}

func DumpPostgres(t *testing.T, dump *testdump.SQL, opt *PostgresOption) {
	t.Helper()

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: opt.FileName,
		FileExt:  ".sql",
	}

	if err := testdump.Postgres(p.String(), dump, opt.Dump); err != nil {
		t.Fatal(err)
	}
}
