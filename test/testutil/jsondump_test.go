package testutil_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/test/testutil"
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

	testutil.DumpJSON(t, p, testutil.IgnoreFields("bornAt"))
}
