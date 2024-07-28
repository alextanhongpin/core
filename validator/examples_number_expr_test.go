package validator_test

import (
	"fmt"

	"github.com/alextanhongpin/core/validator"
)

var ageField = validator.NumberExpr[int]("required,min=13")

type User struct {
	Age int
}

func (u *User) Valid() error {
	return validator.NewErrors(map[string]error{
		"age": ageField.Validate(u.Age),
	})
}

func ExampleNumberExpr() {
	zero := &User{Age: 0}
	minor := &User{Age: 12}
	adult := &User{Age: 13}

	fmt.Printf("%v => %v\n", zero.Age, zero.Valid())
	fmt.Printf("%v => %v\n", minor.Age, minor.Valid())
	fmt.Printf("%v => %v\n", adult.Age, adult.Valid())

	// Output:
	// 0 => age: must not be zero
	// 12 => age: min 13
	// 13 => <nil>
}
