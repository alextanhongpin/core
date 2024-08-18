package batch_test

import (
	"fmt"

	"github.com/alextanhongpin/core/sync/batch"
)

// An Author has many Books.
type Book struct {
	ID       int
	AuthorID int
}

type Author struct {
	ID    int
	Books []Book
}

func ExampleOneToMany() {
	l := newBookLoader()

	// We have a bunch of books, and we want to load the author.
	authors := make([]Author, 3)
	for i := 0; i < len(authors); i++ {
		authors[i].ID = i

		// Load and assign Books to Author.
		if err := l.LoadMany(&authors[i].Books, authors[i].ID); err != nil {
			panic(err)
		}
	}

	// Initiate the fetch.
	if err := l.Wait(); err != nil {
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

func newBookLoader() *batch.Loader[int, Book] {
	batchFn := func(authorIds ...int) ([]Book, error) {
		var books []Book
		for _, id := range authorIds {
			// The number of books is proportional to the AuthorID.
			// AuthorID 0 have 0 books.
			// AuthorID 1 have 1 book.
			// AuthorID n have n books.
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

	return batch.New(batch.Option[int, Book]{
		BatchFn: batchFn,
		KeyFn:   keyFn,
	})
}
