package testutil_test

import (
	"testing"

	"github.com/alextanhongpin/core/test/testutil"
)

func TestDumper(t *testing.T) {
	type data struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	d := testutil.New(t)
	d.DumpJSON(data{
		Name: "John Appleseed",
		Age:  18,
	})
	d.DumpYAML(data{
		Name: "John Appleseed",
		Age:  20,
	})
	d.DumpText("hello world")
}
