package testutil_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/test/testutil"
)

func TestDumpYAML(t *testing.T) {
	type Person struct {
		Name      string    `yaml:"name"`
		Age       int64     `yaml:"age"`
		IsMarried bool      `yaml:"isMarried"`
		BornAt    time.Time `yaml:"bornAt"`
	}

	p := Person{
		Name:      "John Appleseed",
		Age:       13,
		IsMarried: true,
		BornAt:    time.Now(),
	}

	testutil.DumpYAML(t, p, testutil.IgnoreKeys("bornAt"))
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
