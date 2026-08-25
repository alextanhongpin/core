package specification_test

import (
	"testing"

	"github.com/alextanhongpin/core/types/specification"
)

func TestSpecification(t *testing.T) {
	type User struct {
		Age  int
		Name string
	}

	legalAge := specification.Func[*User](func(u *User) bool {
		return u.Age >= 13
	})
	hasName := specification.Func[*User](func(u *User) bool {
		return u.Name != ""
	})

	rule := specification.And(legalAge, hasName)
	var tests = []struct {
		scenario string
		user     *User
		want     bool
	}{
		{
			scenario: "legal age",
			user:     &User{Age: 13, Name: "John"},
			want:     true,
		},
		{
			scenario: "no name",
			user:     &User{Age: 13, Name: ""},
		},
		{
			scenario: "underage",
			user:     &User{Age: 12, Name: "John"},
		},
		{
			scenario: "zero",
			user:     &User{},
		},
	}
	for _, test := range tests {
		t.Run(test.scenario, func(t *testing.T) {
			got := rule.IsSatisfiedBy(test.user)
			if test.want != got {
				t.Fatalf("want %t, got %t", test.want, got)
			}
		})
	}
}
