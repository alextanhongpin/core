package testutil_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/test/testutil"
	"github.com/stretchr/testify/assert"
)

func TestSQL(t *testing.T) {
	db := newMockDB(t, "1", "Alice", "2", "Bob", "3", "Carol")
	dbi := testutil.DumpSQL(t, db, "postgres")

	for i := 0; i < 3; i++ {
		repo := newMockUserRepository(dbi, "postgres")
		_, err := repo.FindUser(context.Background(), fmt.Sprint(i+1))
		assert.Nil(t, err)
	}
}

func TestSQLCallOptions(t *testing.T) {
	db := newMockDB(t, "1", "Alice", "2", "Bob", "3", "Carol")

	dbi := testutil.DumpSQL(t, db, "postgres")
	dbi.SetCallOptions(0, testutil.FileName("a"))
	dbi.SetCallOptions(1, testutil.FileName("b"))
	dbi.SetCallOptions(2, testutil.FileName("c"))

	for i := 0; i < 3; i++ {
		repo := newMockUserRepository(dbi, "postgres")
		_, err := repo.FindUser(context.Background(), fmt.Sprint(i+1))
		assert.Nil(t, err)
	}
}

func TestSQLResult(t *testing.T) {
	db := newMockDB(t, "1", "Alice", "2", "Bob", "3", "Carol")

	dbi := testutil.DumpSQL(t, db, "postgres")
	for i := 0; i < 3; i++ {
		repo := newMockUserRepository(dbi, "postgres")
		user, err := repo.FindUser(context.Background(), fmt.Sprint(i+1))
		assert.Nil(t, err)
		dbi.SetResult(user)
	}
}
