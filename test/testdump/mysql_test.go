package testdump_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/test/testdump"
	"github.com/google/go-cmp/cmp"
)

func TestMySQL(t *testing.T) {
	fileName := fmt.Sprintf("testdata/%s.sql", t.Name())
	type User struct {
		ID   int64
		Name string
	}

	data := testdump.SQL{
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
			time.Now(),
			time.Now(),
		},
		Result: []User{
			{ID: rand.Int63(), Name: "Alice"},
			{ID: rand.Int63(), Name: "Bob"},
		},
	}

	opt := testdump.SQLOption{
		Args: []cmp.Option{
			internal.IgnoreMapEntries(":v1", ":v2"),
		},
		Result: []cmp.Option{
			internal.IgnoreMapEntries("id"),
		},
	}
	if err := testdump.MySQL(fileName, &data, &opt); err != nil {
		t.Fatal(err)
	}
}
