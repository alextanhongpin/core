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
	d := &SQLDumper{
		t:             t,
		db:            db,
		driverName:    driverName,
		optsByCall:    make(map[int][]SQLOption),
		resultsByCall: make(map[int]any),
		opts:          opts,
		seen:          make(map[string]int),
	}
	t.Cleanup(d.dump)
	return d
}

var _ db = (*SQLDumper)(nil)

// SQLDumper logs the query and args.
type SQLDumper struct {
	t             *testing.T
	db            db
	driverName    string
	i             int
	optsByCall    map[int][]SQLOption
	resultsByCall map[int]any
	opts          []SQLOption
	seen          map[string]int
	calls         []SQLCall
}

type SQLCall struct {
	params     *SQL
	driverName string
}

// SetDB sets the db.
func (r *SQLDumper) SetDB(db db) {
	r.db = db
}

// SetDriverName sets the driver name.
func (r *SQLDumper) SetDriverName(driverName string) {
	r.driverName = driverName
}

func (r *SQLDumper) SetResult(v any) {
	r.resultsByCall[r.i-1] = v
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

func (r *SQLDumper) log(method, query string, args ...any) {
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

	r.optsByCall[r.i] = append(opts, FileName(fileName))
	r.calls = append(r.calls, SQLCall{
		params:     &SQL{Args: args, Query: query},
		driverName: r.driverName,
	})
}

func (r *SQLDumper) dump() {
	for i, call := range r.calls {
		call := call
		call.params.Result = r.resultsByCall[i]
		switch call.driverName {
		case "postgres":
			DumpPostgres(r.t, call.params, r.Options(i)...)
		case "mysql":
			DumpMySQL(r.t, call.params, r.Options(i)...)
		default:
			panic("unsupported driver: " + r.driverName)
		}
	}
}

func (r *SQLDumper) Exec(query string, args ...any) (sql.Result, error) {
	r.log("exec", query, args...)

	return r.db.Exec(query, args...)
}

func (r *SQLDumper) Prepare(query string) (*sql.Stmt, error) {
	r.log("prepare", query)

	return r.db.Prepare(query)
}

func (r *SQLDumper) Query(query string, args ...any) (*sql.Rows, error) {
	r.log("query", query, args...)

	return r.db.Query(query, args...)
}

func (r *SQLDumper) QueryRow(query string, args ...any) *sql.Row {
	r.log("query_row", query, args...)

	return r.db.QueryRow(query, args...)
}

func (r *SQLDumper) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	r.log("exec_context", query, args...)

	return r.db.ExecContext(ctx, query, args...)
}

func (r *SQLDumper) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	r.log("prepare_context", query)

	return r.db.PrepareContext(ctx, query)
}

func (r *SQLDumper) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	r.log("query_context", query, args...)

	return r.db.QueryContext(ctx, query, args...)
}

func (r *SQLDumper) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	r.log("query_row_context", query, args...)

	return r.db.QueryRowContext(ctx, query, args...)
}
