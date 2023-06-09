package batch_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/data/batch"
)

func ExampleOneToOne() {
	// Book belongs to an Author.
	type Author struct {
		ID   int
		Name string
	}

	type Book struct {
		ID       int
		AuthorID int
		Author   *Author
	}

	batchFn := func(authorIds ...int) ([]Author, error) {
		authors := make([]Author, len(authorIds))
		for i, id := range authorIds {
			authors[i] = Author{
				ID:   id,
				Name: fmt.Sprintf("author of book %d", id),
			}
		}
		return authors, nil
	}

	keyFn := func(a Author) (authorID int, err error) {
		authorID = a.ID
		return
	}

	loader := batch.NewLoader(batchFn, keyFn)

	// We have a bunch of books, and we want to load the author.
	books := []Book{
		{ID: 1, AuthorID: 1},
		{ID: 2, AuthorID: 1}, // Same author as Book ID 1.
		{ID: 3, AuthorID: 2},
	}

	for i := 0; i < len(books); i++ {
		// Create a non-nil Author.
		books[i].Author = new(Author)

		// Load and assign Author to Book.
		loader.Load(books[i].AuthorID, books[i].Author)
	}

	// Initiate the fetch.
	if err := loader.Wait(); err != nil {
		panic(err)
	}

	fmt.Println(books[0].Author.Name)
	fmt.Println(books[1].Author.Name)
	fmt.Println(books[2].Author.Name)
	// Output:
	// author of book 1
	// author of book 1
	// author of book 2
}

func ExampleOneToMany() {
	// An Author has many Books.
	type Book struct {
		ID       int
		AuthorID int
	}

	type Author struct {
		ID    int
		Books []Book
	}

	batchFn := func(authorIds ...int) ([]Book, error) {
		var books []Book
		for _, id := range authorIds {
			// The number of books is proportional to the AuthorID.
			// AuthorID 0 will 0 books.
			// AuthorID 1 will 1 book.
			// AuthorID n will n books.
			for j := 0; j < id; j++ {
				books = append(books, Book{
					ID:       j,
					AuthorID: id,
				})
			}
		}

		return books, nil
	}

	keyFn := func(b Book) (authorID int, err error) {
		authorID = b.AuthorID
		return
	}

	loader := batch.NewLoader(batchFn, keyFn)

	// We have a bunch of books, and we want to load the author.
	authors := make([]Author, 3)
	for i := 0; i < len(authors); i++ {
		authors[i].ID = i

		// Load and assign Books to Author.
		loader.LoadMany(authors[i].ID, &authors[i].Books)
	}

	// Initiate the fetch.
	if err := loader.Wait(); err != nil {
		panic(err)
	}

	for i := range authors {
		fmt.Printf("author %d has %d books\n", authors[i].ID, len(authors[i].Books))
	}
	// Output:
	// author 0 has 0 books
	// author 1 has 1 books
	// author 2 has 2 books
}

func TestLoad(t *testing.T) {
	l := newUserLoader()

	ids := []int{1, 1, 2, 3, 4}

	n := len(ids)
	users := make([]User, n)
	for i := 0; i < n; i++ {
		l.Load(ids[i], &users[i])
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
		books := make([][]Book, n)
		for i := 0; i < n; i++ {
			l.LoadMany(authorIds[i], &books[i])
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

			var b []Book
			l.LoadMany(i, &b)
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

		var b Book

		// ID 0 will load 0 books.
		l.Load(0, &b)
		err := l.Wait()
		if !errors.Is(err, batch.ErrKeyNotFound) {
			t.Fatalf("want batch.ErrKeyNotFound, got %v", err)
		}
	})

	t.Run("load one when has one", func(t *testing.T) {
		l := newBooksLoader()

		var b Book

		// ID 0 will load 0 books.
		l.Load(1, &b)
		err := l.Wait()
		if err != nil {
			t.Fatalf("want nil error, got %v", err)
		}
	})

	t.Run("load one when has many", func(t *testing.T) {
		l := newBooksLoader()

		var b Book
		// ID 2 will load 2 books.
		l.Load(2, &b)
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
	l.Load(id, &u)
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
	l.Load(1, d.User)
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

	return batch.NewLoader(batchFn, keyFn)
}

type Publication struct {
	Year int
}

type Book struct {
	ID          int
	AuthorID    int
	Publication *Publication
}

// newBooksLoader returns a one-to-many loader.
// one Author id will load multiple books
func newBooksLoader() *batch.Loader[int, Book] {
	batchFn := func(ids ...int) ([]Book, error) {
		var books []Book
		// Number of books is proportional to id.
		// ID 0 will produce 0 books.
		// ID 1 will produce 2 books.
		for _, id := range ids {
			for j := 0; j < id; j++ {
				books = append(books, Book{
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

	keyFn := func(b Book) (int, error) {
		return b.AuthorID, nil
	}

	return batch.NewLoader(batchFn, keyFn)
}
