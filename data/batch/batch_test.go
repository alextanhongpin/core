package batch_test

import (
	"errors"
	"math/rand"
	"testing"

	"github.com/alextanhongpin/core/data/batch"
)

func TestBatch(t *testing.T) {
	// Ensure test is reproducible.
	rand.Seed(0)

	l := newUserLoader()
	n := 5

	var users []*User
	for i := 0; i < n; i++ {
		users = append(users, l.Load(rand.Intn(n)))
	}

	if err := l.Wait(); err != nil {
		t.Fatal(err)
	}

	// Tests that the user at index 0 and 1 are of the same ID.
	if users[0].ID != users[1].ID {
		t.Fatal("user id does not match")
	}

	// Tests that the values are deep-copied, and not shared
	// by reference.
	// Modifying one User's Meta should not modify the other
	// User's Meta.
	users[0].Meta.Age = 42

	if want, got := 13, users[1].Meta.Age; want != got {
		t.Fatalf("want %d, got %d", want, got)
	}
}

func TestBatchKeyNotFound(t *testing.T) {
	l := newUserLoader()
	user := l.Load(100)
	err := l.Wait()
	if !errors.Is(err, batch.ErrKeyNotFound) {
		t.Fatalf("want batch.ErrKeyNotFound, got %v", err)
	}

	if *user != (User{}) {
		t.Fatalf("want zero user, got %v", user)
	}
}

func TestBatchClosed(t *testing.T) {
	l := newUserLoader()
	if err := l.Wait(); err != nil {
		t.Fatalf("want nil error, got %v", err)
	}

	err := l.Wait()
	if !errors.Is(err, batch.ErrClosed) {
		t.Fatalf("want batch.ErrClosed, got %v", err)
	}
}

type Meta struct {
	Age int
}

type User struct {
	ID   int
	Meta *Meta
}

func newUserLoader() *batch.Loader[int, User] {
	batchFn := func(ids ...int) ([]User, error) {
		users := make([]User, 0, len(ids))
		for _, id := range ids {
			if id > 99 {
				continue
			}
			users = append(users, User{
				ID:   id,
				Meta: &Meta{Age: 13},
			})
		}

		return users, nil
	}
	keyFn := func(u User) (int, error) {
		return u.ID, nil
	}

	return batch.NewLoader(batchFn, keyFn)
}
