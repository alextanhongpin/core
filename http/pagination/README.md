# pagination

Cursor (keyset) and offset pagination helpers.

```go
cursor, err := pagination.CursorFromString(q.Get("after"), q.Get("before"), q.Get("first"), q.Get("last"))
if err == nil {
	err = cursor.Validate(100)
}
if err != nil { /* 400 */ }

// Forward:  WHERE id > $after  ORDER BY id ASC  LIMIT cursor.Limit()
// Backward: WHERE id < $before ORDER BY id DESC LIMIT cursor.Limit()
rows := query(cursor)

page := pagination.Paginate(rows, cursor)
info := pagination.BuildPageInfo(page, func(id string) string {
	return pagination.EncodeCursor(id)
})
```

## Cursor

- `Cursor[T]{After, Before, First, Last}`; `Limit()` is `First+1` (or
  `Last+1`): fetch one extra row to detect another page.
- `Validate(max)` requires exactly one of `First`/`Last`, non-negative and at
  most `max`.
- `Paginate(items, cursor)` trims the extra row and returns a copy of the
  page. Forward input is ascending and sets `HasNext`; backward input is
  descending (nearest the cursor first), sets `HasPrev`, and is returned in
  ascending order. The opposite flag is always `false`, since it needs
  another query. `T` is both the item and the cursor type.
- `EncodeCursor` / `DecodeCursor[T]` wrap a JSON value in base64url. Cursors
  are opaque, not tamper-proof: validate decoded values like any input.

## Offset

- `NewOffsetPagination(page, limit)` defaults invalid values (page 1,
  limit 10) and clamps the page so the offset cannot overflow.
- `Validate(max)`, `TotalPages()`, `HasNext()`, `HasPrev()` use `Total`.
- `PaginateOffset(items, p)` slices an in-memory result set and sets
  `p.Total`.
