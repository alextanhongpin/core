package testutil_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/test/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestDumpJSONType(t *testing.T) {
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

	opts := []cmp.Option{
		// Ignore bornAt, which is dynamic.
		cmpopts.IgnoreFields(Person{}, "BornAt"),
	}

	testutil.DumpJSONType(t, p,
		testutil.CmpOptions(opts),
		testutil.JSONTypeInterceptor[Person](
			func(p Person) (Person, error) {
				// Modify the age to 42.
				p.Age = 42

				return p, nil
			},
		),
	)
}
