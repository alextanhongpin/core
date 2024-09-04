package batch_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/exp/batch"
)

func TestLoad(t *testing.T) {
	l := newUserLoader()

	ids := []int{1, 1, 2, 3, 4}

	n := len(ids)
	users := make([]User, n)
	for i := 0; i < n; i++ {
		l.Load(&users[i], ids[i])
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

func TestLoadMany(t *testing.T) {
	t.Run("load one-to-many copied", func(t *testing.T) {
		l := newBooksLoader()

		authorIds := []int{1, 1, 2, 3, 4}

		n := len(authorIds)
		books := make([][]PaperBook, n)
		for i := 0; i < n; i++ {
			l.LoadMany(&books[i], authorIds[i])
		}

		if err := l.Wait(); err != nil {
			t.Fatal(err)
		}

		// Tests that the books at index 0 and 1 are of the same ID.
		for i := 0; i < len(books[0]); i++ {
			if books[0][i].ID != books[1][i].ID {
				t.Fatal("book id does not match")
			}

			if books[0][i].AuthorID != books[1][i].AuthorID {
				t.Fatal("author id does not match")
			}
		}

		// Tests that the values are deep-copied, and not shared
		// by reference.
		// Modifying one User's Meta should not modify the other
		// User's Meta.
		books[0][0].Publication.Year = 2042

		if want, got := 2023, books[1][0].Publication.Year; want != got {
			t.Fatalf("want %d, got %d", want, got)
		}
	})

	for i := 0; i < 3; i++ {
		i := i
		t.Run(fmt.Sprintf("load %d-books", i), func(t *testing.T) {
			l := newBooksLoader()

			var b []PaperBook
			l.LoadMany(&b, i)
			if err := l.Wait(); err != nil {
				t.Fatalf("want nil error, got %v", err)
			}

			if want, got := i, len(b); want != got {
				t.Fatalf("want %d books, got %d", want, got)
			}
		})
	}

	t.Run("load one when has zero", func(t *testing.T) {
		l := newBooksLoader()

		var b PaperBook

		// ID 0 will load 0 books.
		l.Load(&b, 0)
		err := l.Wait()
		if !errors.Is(err, batch.ErrKeyNotFound) {
			t.Fatalf("want batch.ErrKeyNotFound, got %v", err)
		}
	})

	t.Run("load one when has one", func(t *testing.T) {
		l := newBooksLoader()

		var b PaperBook

		// ID 1 will load 1 book.
		l.Load(&b, 1)
		err := l.Wait()
		if err != nil {
			t.Fatalf("want nil error, got %v", err)
		}
	})

	t.Run("load one when has many", func(t *testing.T) {
		l := newBooksLoader()

		var b PaperBook
		// ID 2 will load 2 books.
		l.Load(&b, 2)
		err := l.Wait()
		if !errors.Is(err, batch.ErrMultipleValuesFound) {
			t.Fatalf("want batch.ErrMultipleValuesFound, got %v", err)
		}
	})
}

func TestLoadKeyNotFound(t *testing.T) {
	id := 100
	l := newUserLoader(id)

	var u User
	l.Load(&u, id)
	err := l.Wait()

	if !errors.Is(err, batch.ErrKeyNotFound) {
		t.Fatalf("want batch.ErrKeyNotFound, got %v", err)
	}

	if u != (User{}) {
		t.Fatalf("want zero user, got %v", u)
	}
}

func TestLoadClosed(t *testing.T) {
	l := newUserLoader()
	if err := l.Wait(); err != nil {
		t.Fatalf("want nil error, got %v", err)
	}

	err := l.Wait()
	if !errors.Is(err, batch.ErrClosed) {
		t.Fatalf("want batch.ErrClosed, got %v", err)
	}
}

func TestLoadNil(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			if !errors.Is(r.(error), batch.ErrNilReference) {
				t.Fatalf("want batch.ErrNilReference, got %v", r)
			}

		}
	}()

	type Data struct {
		User *User
	}

	l := newUserLoader()

	// Panics when a nil reference is passed in.
	var d Data
	l.Load(d.User, 1)
	if err := l.Wait(); err != nil {
		t.Fatalf("want nil error, got %v", err)
	}
}

type Meta struct {
	Age int
}

type User struct {
	ID   int
	Meta *Meta
}

// newUserLoader returns a one-to-one loader.
func newUserLoader(ignoreIds ...int) *batch.Loader[int, User] {
	ignore := make(map[int]bool)
	for _, id := range ignoreIds {
		ignore[id] = true
	}

	batchFn := func(ids ...int) ([]User, error) {
		users := make([]User, 0, len(ids))
		for _, id := range ids {
			if ignore[id] {
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

	return batch.New(batch.Option[int, User]{
		BatchFn: batchFn,
		KeyFn:   keyFn,
	})
}

type Publication struct {
	Year int
}

type PaperBook struct {
	ID          int
	AuthorID    int
	Publication *Publication
}

// newBooksLoader returns a one-to-many loader.
// one Author id will load multiple books
func newBooksLoader() *batch.Loader[int, PaperBook] {
	batchFn := func(ids ...int) ([]PaperBook, error) {
		var books []PaperBook
		// Number of books is proportional to id.
		// ID 0 will produce 0 books.
		// ID 1 will produce 2 books.
		for _, id := range ids {
			for j := 0; j < id; j++ {
				books = append(books, PaperBook{
					ID:       j,
					AuthorID: id,
					Publication: &Publication{
						Year: 2023,
					},
				})
			}
		}

		return books, nil
	}

	keyFn := func(b PaperBook) (int, error) {
		return b.AuthorID, nil
	}

	return batch.New(batch.Option[int, PaperBook]{
		BatchFn: batchFn,
		KeyFn:   keyFn,
	})
}
