package sqldump_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/storage/sql/sqldump"
)

func TestDumpMySQL(t *testing.T) {
	type User struct {
		ID   int64
		Name string
	}

	type T = []User

	dump := sqldump.SQL{
		Query: `SELECT *
     FROM users
     WHERE email = 'john.doe@mail.com'
     AND deleted_at IS NULL
     AND last_logged_in_at > ?
		 AND created_at IN (?) 
     AND description = 'foo bar walks in a bar, h''a'
     AND subscription in ('freemium', 'premium')
		 AND age > 13
		 AND is_active = true
 		 AND name LIKE ANY('{Foo,bar,%oo%}')`,
		Args: []any{
			time.Now().Format("2006-01-02"),
			time.Now().Format("2006-01-02"),
		},
		Result: []User{
			{ID: rand.Int63(), Name: "Alice"},
			{ID: rand.Int63(), Name: "Bob"},
		},
	}

	b, err := sqldump.DumpMySQL(&dump, internal.MarshalYAMLPreserveKeysOrder)
	if err != nil {
		t.Fatal(err)
	}

	fileName := fmt.Sprintf("testdata/%s.sql", t.Name())
	if err := internal.WriteIfNotExists(fileName, b); err != nil {
		t.Fatal(err)
	}
}

func TestMatchMySQLQuery(t *testing.T) {
	a := `select * from users where name = 'john' and id = 1`
	b := `
	SELECT *
	FROM users
	WHERE id = 1
	AND name = 'Jane'
	`
	ok, err := sqldump.MatchMySQLQuery(a, b)
	if err != nil {
		t.Fatalf("got error %v", err)
	}
	if !ok {
		t.Fatal("mysql query does not match")
	}
}
