package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
)

type db interface {
	Exec(query string, args ...any) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row

	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// DumpSQL ...
func DumpSQL(t *testing.T, db db, driverName string, opts ...SQLOption) *SQLDumper {
	return &SQLDumper{
		t:          t,
		db:         db,
		driverName: driverName,
		optsByCall: make(map[int][]SQLOption),
		opts:       opts,
		seen:       make(map[string]int),
	}
}

var _ db = (*SQLDumper)(nil)

// SQLDumper logs the query and args.
type SQLDumper struct {
	t          *testing.T
	db         db
	driverName string
	i          int
	optsByCall map[int][]SQLOption
	opts       []SQLOption
	seen       map[string]int
}

// SetDB sets the db.
func (r *SQLDumper) SetDB(db db) {
	r.db = db
}

// SetDriverName sets the driver name.
func (r *SQLDumper) SetDriverName(driverName string) {
	r.driverName = driverName
}

// Calls returns the number of calls made to the db.
func (r *SQLDumper) Calls() int {
	return r.i
}

// SetCallOptions sets the options for the i-th call.
func (r *SQLDumper) SetCallOptions(i int, optsByCall ...SQLOption) {
	r.optsByCall[i] = append(r.optsByCall[i], optsByCall...)
}

// Options returns the options for the i-th call.
func (r *SQLDumper) Options(i int) []SQLOption {
	return append(r.opts, r.optsByCall[i]...)
}

func (r *SQLDumper) dump(method, query string, args ...any) {
	defer func() {
		r.i++
	}()

	var fileName string
	opts := r.Options(r.i)
	for _, o := range opts {
		switch v := o.(type) {
		case FileName:
			fileName = string(v)
		}
	}

	if fileName == "" {
		fileName = method
	}

	r.seen[fileName]++
	if n := r.seen[fileName]; n > 1 {
		fileName = fmt.Sprintf("%s#%d", fileName, n-1)
	}

	opts = append(opts, FileName(fileName))
	params := &SQL{Args: args, Query: query}

	switch r.driverName {
	case "postgres":
		DumpPostgres(r.t, params, opts...)
	case "mysql":
		DumpMySQL(r.t, params, opts...)
	default:
		panic("unsupported driver: " + r.driverName)
	}
}

func (r *SQLDumper) Exec(query string, args ...any) (sql.Result, error) {
	r.dump("exec", query, args...)

	return r.db.Exec(query, args...)
}

func (r *SQLDumper) Prepare(query string) (*sql.Stmt, error) {
	r.dump("prepare", query)

	return r.db.Prepare(query)
}

func (r *SQLDumper) Query(query string, args ...any) (*sql.Rows, error) {
	r.dump("query", query, args...)

	return r.db.Query(query, args...)
}

func (r *SQLDumper) QueryRow(query string, args ...any) *sql.Row {
	r.dump("query_row", query, args...)

	return r.db.QueryRow(query, args...)
}

func (r *SQLDumper) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	r.dump("exec_context", query, args...)

	return r.db.ExecContext(ctx, query, args...)
}

func (r *SQLDumper) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	r.dump("prepare_context", query)

	return r.db.PrepareContext(ctx, query)
}

func (r *SQLDumper) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	r.dump("query_context", query, args...)

	return r.db.QueryContext(ctx, query, args...)
}

func (r *SQLDumper) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	r.dump("query_row_context", query, args...)

	return r.db.QueryRowContext(ctx, query, args...)
}
