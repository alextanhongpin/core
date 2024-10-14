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

var (
	emailField         = validator.StringExpr("email,ends_with=@mail.com").Func(adminOnly)
	maritalStatusField = validator.StringExpr("optional,oneof=single married divorced")
)

type Account struct {
	Email         string
	MaritalStatus *string
}

func (a *Account) String() string {
	email := "n/a"
	status := "n/a"
	if a.Email != "" {
		email = a.Email
	}
	if a.MaritalStatus != nil {
		status = *a.MaritalStatus
	}
	return fmt.Sprintf("email: %s, marital_status: %s", email, status)
}

func (u *Account) Valid() error {
	return validator.NewErrors(map[string]error{
		"email":          emailField.Validate(u.Email),
		"marital_status": maritalStatusField.Validate(validator.Value(u.MaritalStatus)),
	})
}

func ExampleStringExpr() {
	noemail := &Account{Email: ""}
	nonadmin := &Account{Email: "jane.doe@mail.com"}
	email := &Account{Email: "john.doe@mail.com"}
	unknownStatus := "unknown"
	unknown := &Account{Email: "john.doe@mail.com", MaritalStatus: &unknownStatus}
	invalidUnknown := &Account{Email: "jane.doe@mail.com", MaritalStatus: &unknownStatus}

	fmt.Printf("%s => %v\n", noemail, noemail.Valid())
	fmt.Printf("%s => %v\n", nonadmin, nonadmin.Valid())
	fmt.Printf("%s => %v\n", email, email.Valid())
	fmt.Printf("%s => %v\n", unknown, unknown.Valid())
	fmt.Printf("%s => %v\n", invalidUnknown, invalidUnknown.Valid())

	var ve validator.Errors
	fmt.Println(errors.As(invalidUnknown.Valid(), &ve))
	fmt.Println(ve["email"])
	fmt.Println(ve["marital_status"])
	// Output:
	// email: n/a, marital_status: n/a => email: must not be empty
	// email: jane.doe@mail.com, marital_status: n/a => email: admin only
	// email: john.doe@mail.com, marital_status: n/a => <nil>
	// email: john.doe@mail.com, marital_status: unknown => marital_status: must be one of single, married, divorced
	// email: jane.doe@mail.com, marital_status: unknown => email: admin only
	// marital_status: must be one of single, married, divorced
	// true
	// admin only
	// must be one of single, married, divorced
}
