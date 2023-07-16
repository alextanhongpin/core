package always_test

import (
	"errors"
	"fmt"

	"github.com/alextanhongpin/core/types/always"
)

func ExampleValid() {
	err := always.Valid(
		new(Author),
		new(Book),
		always.ValidFunc(func() error {
			fmt.Println("this is not called")
			return nil
		}),
	)
	fmt.Println(err)

	p := &Publication{
		Author: new(Author),
		Book:   new(Book),
	}
	fmt.Println(p.Valid())

	// Output:
	// author: name empty
	// book: ISBN empty
}

type Book struct {
	ISBN string
}

func (b *Book) Valid() error {
	if b.ISBN == "" {
		return errors.New("book: ISBN empty")
	}

	return nil
}

type Author struct {
	Name string
}

func (a *Author) Valid() error {
	if a.Name == "" {
		return errors.New("author: name empty")
	}

	return nil
}

type Publication struct {
	Book   *Book
	Author *Author
}

// Custom nest validate function.
func (p *Publication) Valid() error {
	return always.Valid(
		p.Book,
		p.Author,
	)
}
