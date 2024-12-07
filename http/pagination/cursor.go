package pagination

type Cursor[T any] struct {
	After T
	First int
}

// Limit converts the First into database limit, and fetches an additional row
// to check if there are more items.
func (c *Cursor[T]) Limit() int {
	return c.First + 1
}

type Pagination[T any] struct {
	Items   []T
	Cursor  *Cursor[T]
	HasNext bool
}

func Paginate[T any](items []T, cursor *Cursor[T]) *Pagination[T] {
	if len(items) > cursor.First {
		items = items[:cursor.First]

		return &Pagination[T]{
			Items: items,
			Cursor: &Cursor[T]{
				After: items[len(items)-1],
				First: cursor.First,
			},
			HasNext: true,
		}
	}

	return &Pagination[T]{
		Items: items,
		Cursor: &Cursor[T]{
			First: cursor.First,
		},
	}
}
