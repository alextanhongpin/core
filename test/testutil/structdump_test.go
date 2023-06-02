package testutil_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/test/testutil"
)

func TestDumpStruct(t *testing.T) {
	type Person struct {
		name      string
		age       int64
		isMarried bool
		bornAt    time.Time
	}

	p := Person{
		name:      "John Appleseed",
		age:       13,
		isMarried: true,
		bornAt:    time.Now(),
	}
	// Due to limitations in the library for diffing, we
	// need to fix the value here.
	p.bornAt = time.Time{}

	testutil.DumpStruct(t, p)
}
