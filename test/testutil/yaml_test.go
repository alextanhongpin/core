package testutil_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/test/testutil"
)

func TestDumpYAML(t *testing.T) {
	type Person struct {
		Name      string
		Age       int64
		IsMarried bool
		BornAt    time.Time
		Biography string
	}

	p := Person{
		Name:      "John Appleseed",
		Age:       13,
		IsMarried: true,
		BornAt:    time.Now(),
		Biography: `You see, tonight, it could go either way
Hearts balanced on a razor blade
We are designed to love and break
And to rinse and repeat it all again`,
	}

	type T = Person
	testutil.DumpYAML(t, p, testutil.IgnoreKeys[T]("BornAt"))
}

func TestDumpYAMLNonStruct(t *testing.T) {
	nums := []int{1, 2, 3}
	testutil.DumpYAML(t, nums)
}

func TestDumpYAMLMaskField(t *testing.T) {
	type Credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	creds := Credentials{
		Email:    "john.appleseed@mail.com",
		Password: "s3cr3t",
	}

	type T = Credentials
	testutil.DumpYAML(t, creds,
		testutil.MaskKeys[T]("password"),
	)
}

func TestDumpYAMLIntercept(t *testing.T) {
	nums := []int{1, 2, 3}

	type T = []int

	testutil.DumpYAML(t, nums,
		testutil.InterceptYAML(func(t T) (T, error) {
			// Double the value
			for i, v := range t {
				t[i] = v * 2
			}

			return t, nil
		}),
	)
}
