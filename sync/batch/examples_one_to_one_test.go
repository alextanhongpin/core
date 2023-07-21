package batch_test

import (
	"fmt"

	"github.com/alextanhongpin/core/sync/batch"
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

	loader := batch.New(batchFn, keyFn)

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
		loader.Load(books[i].Author, books[i].AuthorID)
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
