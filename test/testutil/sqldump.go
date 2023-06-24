package testutil

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/google/go-cmp/cmp"
	pg_query "github.com/pganalyze/pg_query_go/v4"
)

const queryStmtSection = "-- Query"
const queryNormalizedStmtSection = "-- Query Normalized"
const argsStmtSection = "-- Args"
const rowsStmtSection = "-- Rows"

type sqlOption struct {
	queryFn   InspectQuery
	argsOpts  []cmp.Option
	rowsOpts  []cmp.Option
	normalize bool
}

func NewSQLOption(opts ...SQLOption) *sqlOption {
	s := &sqlOption{}
	for _, opt := range opts {
		switch o := opt.(type) {
		case InspectQuery:
			s.queryFn = o
		case IgnoreFieldsOption:
			// We share the same options, with the assumptions that there are no
			// keys-collision - args are using keys numbered from $1 to $n.
			s.argsOpts = append(s.argsOpts, IgnoreMapKeys(o...))
			s.rowsOpts = append(s.rowsOpts, IgnoreMapKeys(o...))
		case ArgsCmpOptions:
			s.argsOpts = append(s.argsOpts, o...)
		case RowsCmpOptions:
			s.rowsOpts = append(s.rowsOpts, o...)
		case FilePath, FileName:
		// Do nothing.
		case *NormalizeOption:
			s.normalize = true
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
	Stmt string
	Args []any
	Rows any
}

func NewSQLDump(stmt string, args []any, rows any) *SQLDump {
	return &SQLDump{
		Stmt: strings.TrimSpace(stmt),
		Args: args,
		Rows: rows,
	}
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

	if err := cmpDiff(l.Stmt, r.Stmt); err != nil {
		return err
	}

	if err := cmpDiff(toArgsMap(l.Args), toArgsMap(r.Args), c.opt.argsOpts...); err != nil {
		return err
	}

	if err := cmpDiff(l.Rows, r.Rows, c.opt.rowsOpts...); err != nil {
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
		case queryStmtSection:
			var tmp [][]byte

			for s.Scan() {
				line := s.Bytes()
				if len(line) == 0 {
					break
				}

				tmp = append(tmp, line)
			}

			dump.Stmt = string(bytes.Join(tmp, LineBreak))
		case queryNormalizedStmtSection:
			var tmp [][]byte

			for s.Scan() {
				line := s.Bytes()
				if len(line) == 0 {
					break
				}
				tmp = append(tmp, line)
			}

			// If normalized section is present, we compare that instead of the
			// non-normalized query.
			if len(tmp) > 0 {
				dump.Stmt = string(bytes.Join(tmp, LineBreak))
			}
		case argsStmtSection:
			var tmp [][]byte

			for s.Scan() {
				line := s.Bytes()
				if len(line) == 0 {
					break
				}

				tmp = append(tmp, line)
			}

			jsonBytes := bytes.Join(tmp, LineBreak)

			var m map[string]any
			if err := json.Unmarshal(jsonBytes, &m); err != nil {
				return nil, err
			}

			dump.Args = fromArgsMap(m)
		case rowsStmtSection:
			var tmp [][]byte

			for s.Scan() {
				line := s.Bytes()
				if len(line) == 0 {
					break
				}

				tmp = append(tmp, line)
			}

			jsonBytes := bytes.Join(tmp, LineBreak)
			if json.Valid(jsonBytes) {
				switch jsonBytes[0] {
				case '{':
					var m map[string]any
					if err := json.Unmarshal(jsonBytes, &m); err != nil {
						return nil, err
					}
					dump.Rows = m
				case '[':
					var m []map[string]any
					if err := json.Unmarshal(jsonBytes, &m); err != nil {
						return nil, err
					}
					dump.Rows = m
				default:
					var m any
					if err := json.Unmarshal(jsonBytes, &m); err != nil {
						return nil, err
					}
					dump.Rows = m
				}
			} else {
				dump.Rows = string(jsonBytes)
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

func extractSQLVariables(orig, norm string) (string, string) {
	normr := []rune(norm)
	origr := []rune(orig)

	for i := 0; i < len(normr); i++ {
		if norm[i] == '$' {
			normr = normr[i:]
			origr = origr[i:]
			break
		}
	}

	for i := len(normr); i > -1; i-- {
		if unicode.IsDigit(normr[i-1]) {
			n := len(normr) - i
			normr = normr[:len(normr)-n]
			origr = origr[:len(origr)-n]
			break
		}
	}

	orig = string(origr)
	norm = string(normr)

	return orig, norm
}

func normalizePostgres(query string) (norm string, args map[string]any, err error) {
	norm, err = pg_query.Normalize(query)
	if err != nil {
		return
	}

	origTokens := tokenizeSQL(query)
	normTokens := tokenizeSQL(norm)
	if len(origTokens) != len(normTokens) {
		return "", nil, errors.New("sql not normalized")
	}

	args = make(map[string]any)

	for i := 0; i < len(origTokens); i++ {
		o, n := origTokens[i], normTokens[i]
		if o != n {
			orig, norm := extractSQLVariables(o, n)
			args[norm] = parseSQLType(orig)
		}
	}

	return
}

func parseSQLType(v string) any {
	// Remove null characters.
	v = strings.Trim(v, "\x00")

	isQuote := v[0] == '\''
	isString := isQuote && len(v) > 1 && v[0] == v[len(v)-1]
	if isString {
		return v[1 : len(v)-1]
	}

	n, err := strconv.ParseInt(v, 10, 64)
	if err == nil {
		return n
	}

	b, err := strconv.ParseBool(v)
	if err == nil {
		return b
	}

	return v
}

func toArgsMap(args []any) map[string]any {
	res := make(map[string]any)
	for i, arg := range args {
		// For Postgres, it starts from 1.
		key := fmt.Sprintf("$%d", i+1)
		res[key] = arg
	}

	return res
}

func fromArgsMap(m map[string]any) []any {
	args := make([]any, len(m))
	for i := 0; i < len(m); i++ {
		key := fmt.Sprintf("$%d", i+1)
		args[i] = m[key]
	}

	return args
}
