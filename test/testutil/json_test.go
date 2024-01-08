package testutil_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/test/testutil"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestDumpJSON(t *testing.T) {
	type Person struct {
		Name      string    `json:"name"`
		Age       int64     `json:"age"`
		IsMarried bool      `json:"isMarried"`
		BornAt    time.Time `json:"bornAt"`
	}

	p := Person{
		Name:      "John Appleseed",
		Age:       13,
		IsMarried: true,
		BornAt:    time.Now(),
	}

	testutil.DumpJSON(t, p,
		testutil.IgnoreFields("bornAt"))

	testutil.DumpJSON(t, p,
		testutil.IgnoreFields("bornAt"),
		testutil.FileName("person.json"),
	)

	testutil.DumpJSON(t, p,
		testutil.IgnoreFields("bornAt"),
		testutil.FileName("person_no_ext"),
	)

	testutil.DumpJSON(t, p,
		testutil.IgnoreFields("bornAt"),
		testutil.FileName("person_yaml.yaml"),
	)
	testutil.DumpJSON(t, p,
		testutil.CmpOpts(
			cmpopts.IgnoreMapEntries(func(key string, _ any) bool {
				return key == "bornAt"
			}),
		))
}

func TestDumpJSONIgnoreTag(t *testing.T) {
	type Person struct {
		Name      string    `json:"name"`
		Age       int64     `json:"age"`
		IsMarried bool      `json:"isMarried"`
		BornAt    time.Time `json:"bornAt" cmp:",ignore"`
	}

	p := Person{
		Name:      "John Appleseed",
		Age:       13,
		IsMarried: true,
		BornAt:    time.Now(),
	}

	testutil.DumpJSON(t, p)
}

func TestDumpJSONNonStruct(t *testing.T) {
	nums := []int{1, 2, 3}
	testutil.DumpJSON(t, nums)
}

func TestDumpJSONMaskField(t *testing.T) {
	type Credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	creds := Credentials{
		Email:    "john.appleseed@mail.com",
		Password: "s3cr3t",
	}

	testutil.DumpJSON(t, creds, testutil.MaskFields("password"))
}
