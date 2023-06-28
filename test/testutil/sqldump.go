package testutil

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"testing"
	"unicode"

	"github.com/google/go-cmp/cmp"
	pg_query "github.com/pganalyze/pg_query_go/v4"
)

const querySection = "-- Query"
const queryNormalizedSection = "-- Query Normalized"
const argsSection = "-- Args"
const resultSection = "-- Result"

type sqlOption struct {
	queryFn    InspectQuery
	argsOpts   []cmp.Option
	resultOpts []cmp.Option
}

func NewSQLOption(opts ...SQLOption) *sqlOption {
	s := &sqlOption{}
	for _, opt := range opts {
		switch o := opt.(type) {
		case InspectQuery:
			s.queryFn = o
		case ArgsCmpOptions:
			s.argsOpts = append(s.argsOpts, o...)
		case RowsCmpOptions:
			s.resultOpts = append(s.resultOpts, o...)
		case FilePath, FileName:
		// Do nothing.
		default:
			panic("option not implemented")
		}
	}

	return s
}

func DumpSQL(t *testing.T, dump *SQLDump, dialect DialectOption, opts ...SQLOption) {
	t.Helper()
	opt := NewSQLPath(opts...)
	if opt.FilePath == "" {
		opt.FilePath = t.Name()
	}

	fileName := opt.String()
	if err := DumpSQLFile(fileName, dump, dialect, opts...); err != nil {
		t.Fatal(err)
	}
}

func DumpMySQL(t *testing.T, dump *SQLDump, opts ...SQLOption) {
	t.Helper()
	opt := NewSQLPath(opts...)
	if opt.FilePath == "" {
		opt.FilePath = t.Name()
	}

	fileName := opt.String()
	if err := DumpSQLFile(fileName, dump, MySQL(), opts...); err != nil {
		t.Fatal(err)
	}
}

func DumpPostgres(t *testing.T, dump *SQLDump, opts ...SQLOption) {
	t.Helper()
	opt := NewSQLPath(opts...)
	if opt.FilePath == "" {
		opt.FilePath = t.Name()
	}

	fileName := opt.String()
	if err := DumpSQLFile(fileName, dump, Postgres(), opts...); err != nil {
		t.Fatal(err)
	}
}

func DumpSQLFile(fileName string, dump *SQLDump, dialect DialectOption, opts ...SQLOption) error {
	type dumpAndCompare struct {
		dumper
		comparer
	}

	var d dumper
	switch dialect {
	case Postgres():
		d = NewPostgresSQLDumper(dump, opts...)
	case MySQL():
		d = NewMySQLDumper(dump, opts...)
	default:
		log.Fatalf(`sqldump: dialect must be one of "postgres" or "mysql", got %q`, d)
	}

	dnc := dumpAndCompare{
		dumper:   d,
		comparer: NewSQLComparer(opts...),
	}

	return Dump(fileName, dnc)
}

type SQLDump struct {
	Stmt   string
	Args   []any
	Result any

	// The original mapping, used by Comparer for
	// comparison.
	args any
}

func NewSQLDump(stmt string, args []any, res any) *SQLDump {
	return &SQLDump{
		Stmt:   strings.TrimSpace(stmt),
		Args:   args,
		Result: res,
	}
}

func (d *SQLDump) WithResult(res any) *SQLDump {
	d.Result = res
	return d
}

func dynamicLineWidth(query string) int {
	// Conditionally determine the line width so that the text is not a single
	// line when it is too short.
	n := len(query)
	if n < 120 {
		n = n / 2
	} else {
		n = 80
	}
	if n < 32 {
		n = 32
	}

	return n
}

type SQLComparer struct {
	opt *sqlOption
}

func NewSQLComparer(opts ...SQLOption) *SQLComparer {
	return &SQLComparer{
		opt: NewSQLOption(opts...),
	}
}

func (c *SQLComparer) Compare(a, b []byte) error {
	a = bytes.TrimLeft(a, " \t\r\n")
	b = bytes.TrimLeft(b, " \t\r\n")

	l, err := parseSQLDump(a)
	if err != nil {
		return err
	}

	r, err := parseSQLDump(b)
	if err != nil {
		return err
	}

	if c.opt.queryFn != nil {
		c.opt.queryFn(r.Stmt)
	}

	if err := ansiDiff(l.Stmt, r.Stmt); err != nil {
		return err
	}

	if err := ansiDiff(l.args, r.args, c.opt.argsOpts...); err != nil {
		return err
	}

	if err := ansiDiff(l.Result, r.Result, c.opt.resultOpts...); err != nil {
		return err
	}

	return nil
}

func parseSQLDump(b []byte) (*SQLDump, error) {
	br := bytes.NewReader(b)
	s := bufio.NewScanner(br)

	dump := new(SQLDump)

	for s.Scan() {
		line := s.Text()

		switch line {
		case querySection:
			// This is not used for comparison.
			for s.Scan() {
				line := s.Bytes()
				if len(line) == 0 {
					break
				}
			}
		case queryNormalizedSection:
			var tmp [][]byte

			for s.Scan() {
				line := s.Bytes()
				if len(line) == 0 {
					break
				}

				tmp = append(tmp, line)
			}

			dump.Stmt = string(bytes.Join(tmp, LineBreak))
		case argsSection:
			var tmp [][]byte

			for s.Scan() {
				line := s.Bytes()
				if len(line) == 0 {
					break
				}

				tmp = append(tmp, line)
			}

			b := bytes.Join(tmp, LineBreak)
			a, err := unmarshal(b)
			if err != nil {
				return nil, err
			}

			dump.args = a
		case resultSection:
			var tmp [][]byte

			for s.Scan() {
				line := s.Bytes()
				if len(line) == 0 {
					break
				}

				tmp = append(tmp, line)
			}

			b := bytes.Join(tmp, LineBreak)
			if json.Valid(b) {
				a, err := unmarshal(b)
				if err != nil {
					return nil, err
				}
				dump.Result = a
			} else {
				dump.Result = string(b)
			}
		}
	}

	return dump, nil
}

func tokenizeSQL(s string) []string {
	quoted := false
	a := strings.FieldsFunc(s, func(r rune) bool {
		if r == '\'' {
			quoted = !quoted
		}
		return !quoted && r == ' '
	})

	return a
}

// cleanParameters cleans the placeholder and constants.
// Due to the naive tokenization, we might end up with such results:
//
// Normalized query tokens: ["($1", "$2)"]
// Original query tokens: ["('foo'", "'bar')"]
//
// The placeholder token might not start or end with the dollar sign.
// To fix this, we need to shift trim the start and end for both the placholder
// and constant values.
func cleanParameters(p, c string) (string, string) {
	rp := []rune(p)
	rc := []rune(c)

	// Move to the right until we find the first $.
	for i := 0; i < len(rp); i++ {
		if rp[i] == '$' {
			rp = rp[i:]
			rc = rc[i:]
			break
		}
	}

	// Move to the left until we find the first digit.
	for i := len(rp); i > -1; i-- {
		if unicode.IsDigit(rp[i-1]) {
			n := len(rp) - i
			rp = rp[:len(rp)-n]
			rc = rc[:len(rc)-n]
			break
		}
	}

	p = string(rp)
	c = string(rc)

	return p, c
}

// normalizePostgres extracts the constant values from the statement and
// replaces it with placeholders.
func normalizePostgres(query string) (norm string, args map[string]any, err error) {
	// After normalization, the constant variables will be replaced with
	// placeholders, e.g. $1, $2 etc.
	norm, err = pg_query.Normalize(query)
	if err != nil {
		return
	}

	// Tokenize the query to find out the position that has been substituted with
	// the placeholder.
	origTokens := tokenizeSQL(query)
	normTokens := tokenizeSQL(norm)
	if len(origTokens) != len(normTokens) {
		return "", nil, errors.New("sql not normalized")
	}

	safeUnmarshal := func(s string) any {
		// Return "'foo'" as string "foo".
		if len(s) >= 2 {
			if s[0] == '\'' && s[len(s)-1] == '\'' {
				return s[1 : len(s)-1]
			}
		}

		a, err := unmarshal([]byte(s))
		if err != nil {
			return s
		}

		return a
	}

	args = make(map[string]any)
	for i := 0; i < len(origTokens); i++ {
		c, p := origTokens[i], normTokens[i]
		// If the token does not match, then
		if c != p {
			placholder, constant := cleanParameters(p, c)
			args[placholder] = safeUnmarshal(constant)
		}
	}

	return
}

func unmarshal(b []byte) (any, error) {
	var m any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	return m, nil
}
