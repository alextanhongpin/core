package always_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/alextanhongpin/core/types/always"
)

type contextKey string

func ExampleVerifier() {
	type T = *User

	ctx := context.Background()
	err := always.Verify(ctx, new(UserSession), UserIsLoggedIn)
	fmt.Println(err)

	ctx = context.WithValue(ctx, contextKey("user_id"), "user-3173")
	err = always.Verify(ctx, new(UserSession), UserIsLoggedIn)
	fmt.Println(err)

	err = always.Verify(ctx, &UserSession{ID: "user-3173"}, UserIsLoggedIn)
	fmt.Println(err)

	// Output:
	// unauthorized
	// forbidden
	// <nil>
}

func UserIsLoggedIn(ctx context.Context, u *UserSession) error {
	id, ok := ctx.Value(contextKey("user_id")).(string)
	if !ok {
		return errors.New("unauthorized")
	}

	if u.ID != id {
		return errors.New("forbidden")
	}

	return nil
}

type UserSession struct {
	ID string
}
