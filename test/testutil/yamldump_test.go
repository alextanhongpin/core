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

	testutil.DumpYAML(t, p, testutil.IgnoreKeys("BornAt"))
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

	testutil.DumpYAML(t, creds,
		testutil.MaskKeys("password"),
		testutil.InspectYAML(func(b []byte) error {
			t.Log(string(b))
			return nil
		}),
	)
}
