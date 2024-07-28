package validator_test

import (
	"errors"
	"fmt"

	"github.com/alextanhongpin/core/validator"
)

func adminOnly(email string) error {
	if email == "john.doe@mail.com" {
		return nil
	}

	return errors.New("admin only")
}

var emailField = validator.StringExpr("required,email,ends_with=@mail.com", adminOnly)

type Account struct {
	Email string
}

func (u *Account) Valid() error {
	return validator.NewErrors(
		validator.Field("email", emailField.Validate(u.Email)),
	)
}

func ExampleStringExpr() {
	noemail := &Account{Email: ""}
	nonadmin := &Account{Email: "jane.doe@mail.com"}
	email := &Account{Email: "john.doe@mail.com"}

	fmt.Printf("%q => %v\n", noemail.Email, noemail.Valid())
	fmt.Printf("%q => %v\n", nonadmin.Email, nonadmin.Valid())
	fmt.Printf("%q => %v\n", email.Email, email.Valid())
	// Output:
	// "" => email: must not be empty
	// "jane.doe@mail.com" => email: admin only
	// "john.doe@mail.com" => <nil>
}
