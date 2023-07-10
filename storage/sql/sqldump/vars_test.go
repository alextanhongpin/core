package sqldump_test

import (
	"sort"
	"testing"

	"github.com/alextanhongpin/core/storage/sql/sqldump"
)

func TestPostgresVars(t *testing.T) {
	q := `
SELECT *
  FROM users
  WHERE email = 'john.doe@mail.com'
    AND deleted_at IS NULL
    AND last_logged_in_at > $1
    AND created_at IN ($2)
    AND description = 'foo bar walks in a bar, h''a'
    AND subscription IN ('freemium',
                         'premium')
    AND age > 13
    AND is_active = TRUE
    AND name LIKE ANY('{Foo,bar,%oo%}')
    AND id <> ALL(ARRAY[1, 2])
`

	got, err := sqldump.PostgresVars(q)
	if err != nil {
		t.Fatal(err)
	}
	want := []sqldump.Var{
		{"$3", "john.doe@mail.com"},
		{"$1", "$1"},
		{"$2", "$2"},
		{"$4", "foo bar walks in a bar, h''a"},
		{"$5", "freemium"},
		{"$6", "premium"},
		{"$7", "13"},
		{"$8", "true"},
		{"$9", "{Foo,bar,%oo%}"},
		{"$10", "1"},
		{"$11", "2"},
	}

	if len(want) != len(got) {
		t.Fatalf("want %d vars, got %d vars", len(want), len(got))
	}
	for i := 0; i < len(want); i++ {
		if want[i] != got[i] {
			t.Fatalf("want %v, got %v", want[i], got[i])
		}
	}
}

func TestMySQL(t *testing.T) {
	q := `SELECT *
  FROM users
  WHERE email = 'john.doe@mail.com'
    AND deleted_at IS NULL
    AND last_logged_in_at > ?
    AND created_at IN (?)
    AND description = 'foo bar walks in a bar, h\'a'
    AND subscription IN ('freemium',
                         'premium')
    AND age > 13
    AND is_active = TRUE
    AND name like ANY('{Foo,bar,%oo%}')
`

	got, err := sqldump.MySQLVars(q)
	if err != nil {
		t.Fatal(err)
	}

	want := []sqldump.Var{
		{"age", "13"},
		{"description", "foo bar walks in a bar, h'a"},
		{"email", "john.doe@mail.com"},
		{"2", `"freemium","premium"`},
		{"1", "{Foo,bar,%oo%}"},
	}

	sort.Slice(want, func(i, j int) bool {
		return want[i].Name < want[j].Name
	})
	sort.Slice(got, func(i, j int) bool {
		return got[i].Name < got[j].Name
	})

	if len(want) != len(got) {
		t.Fatalf("want %d vars, got %d vars", len(want), len(got))
	}

	for i := 0; i < len(want); i++ {
		if want[i] != got[i] {
			t.Fatalf("want %v, got %v", want[i], got[i])
		}
	}
}
