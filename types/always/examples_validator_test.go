package always_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/alextanhongpin/core/types/always"
)

var ErrUserNameRequired = errors.New("user: name required")

func ExampleValidator() {
	type T = *User

	ctx := context.Background()

	err := always.Validate(new(User),
		UserIsAuthorized(ctx),
		UserIsInitialized,
		(*User).Validate, // Applies the default user validation.
		UserMustBeOfLegalAge,
	)
	fmt.Println(err)

	// Output:
	// checking context ...
	// user: name required
}

type User struct {
	Name string
	Age  int
}

func (u *User) Validate() error {
	if u.Name == "" {
		return ErrUserNameRequired
	}

	return nil
}

func UserMustBeOfLegalAge(u *User) error {
	if u.Age < 13 {
		return errors.New("user: illegal age")
	}

	return nil
}

func UserIsInitialized(u *User) error {
	if u == nil {
		return errors.New("user: zero")
	}

	return nil
}

func UserIsAuthorized(ctx context.Context) func(u *User) error {
	return func(u *User) error {
		fmt.Println("checking context ...")

		return nil
	}
}
