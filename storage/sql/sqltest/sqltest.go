package sqltest

import (
	"context"
	"database/sql"
)

// DBTX represents the common db operations for both *sql.DB and *sql.Tx.
type DBTX interface {
	Exec(query string, args ...any) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row

	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

var _ DBTX = (*Recorder)(nil)

type Recorder struct {
	dbtx  DBTX
	query string
	args  []any
}

func NewRecorder(dbtx DBTX) *Recorder {
	return &Recorder{dbtx: dbtx}
}

func (r *Recorder) Reset() (query string, args []any) {
	query = r.query
	args = r.args

	r.query = ""
	r.args = nil

	return
}

func (r *Recorder) Exec(query string, args ...any) (sql.Result, error) {
	r.query = query
	r.args = args

	return r.dbtx.Exec(query, args...)
}

func (r *Recorder) Prepare(query string) (*sql.Stmt, error) {
	r.query = query

	return r.dbtx.Prepare(query)
}

func (r *Recorder) Query(query string, args ...any) (*sql.Rows, error) {
	r.query = query
	r.args = args

	return r.dbtx.Query(query, args...)
}

func (r *Recorder) QueryRow(query string, args ...any) *sql.Row {
	r.query = query
	r.args = args

	return r.dbtx.QueryRow(query, args...)
}

func (r *Recorder) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	r.query = query
	r.args = args

	return r.dbtx.ExecContext(ctx, query, args...)
}

func (r *Recorder) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	r.query = query

	return r.dbtx.PrepareContext(ctx, query)
}

func (r *Recorder) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	r.query = query
	r.args = args

	return r.dbtx.QueryContext(ctx, query, args...)
}

func (r *Recorder) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	r.query = query
	r.args = args

	return r.dbtx.QueryRowContext(ctx, query, args...)
}
