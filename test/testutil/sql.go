package testutil

import "github.com/google/go-cmp/cmp"

const (
	Postgres DialectOption = "postgres"
	MySQL    DialectOption = "mysql"
)

type IgnoreArgsOption []string

func (o IgnoreArgsOption) isSQL() {}
func IgnoreArgs(fields ...string) ArgsCmpOptions {
	return ArgsCmpOptions([]cmp.Option{IgnoreMapKeys(fields...)})
}

type IgnoreRowsOption []string

func (o IgnoreRowsOption) isSQL() {}

func IgnoreRows(fields ...string) RowsCmpOptions {
	return RowsCmpOptions([]cmp.Option{IgnoreMapKeys(fields...)})
}

type InspectQuery func(query string)

func (o InspectQuery) isSQL() {}

type ArgsCmpOptions []cmp.Option

func (o ArgsCmpOptions) isSQL() {}

type RowsCmpOptions []cmp.Option

func (o RowsCmpOptions) isSQL() {}

type DialectOption string

func (o DialectOption) isSQL() {}
