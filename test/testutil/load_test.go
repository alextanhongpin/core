package testutil_test

import (
	"testing"

	"github.com/alextanhongpin/core/test/testutil"
)

func TestLoadURLAndCompare(t *testing.T) {
	uri := "https://raw.githubusercontent.com/alextanhongpin/core/main/test/testutil/testdata/TestDumpJSON/person.json"
	path := testutil.NewJSONPath(testutil.FileName(t.Name()))
	cmp := testutil.NewJSONComparer()
	if err := testutil.LoadURLAndCompare(uri, path.String(), cmp); err != nil {
		t.Fatal(err)
	}
}
