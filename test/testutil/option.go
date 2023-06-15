package testutil

import (
	"net/http"
	"path/filepath"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type HTTPOption interface {
	isHTTP()
}

type JSONOption interface {
	isJSON()
}

type TestDir string

func (TestDir) isJSON() {}
func (TestDir) isHTTP() {}

type TestName string

func (TestName) isJSON() {}
func (TestName) isHTTP() {}

type FileName string

func (FileName) isJSON() {}
func (FileName) isHTTP() {}

type testOption struct {
	TestDir  string
	TestName string
	FileName string
}

func newTestOption(opts ...JSONOption) *testOption {
	to := &testOption{
		TestDir:  "./testdata",
		TestName: "",
		FileName: "",
	}

	for _, o := range opts {
		switch v := o.(type) {
		case TestDir:
			to.TestDir = string(v)
		case TestName:
			to.TestName = string(v)
		case FileName:
			to.FileName = string(v)
		}
	}

	return to
}

func (o *testOption) String() string {
	return filepath.Join(o.TestDir, o.TestName, o.FileName)
}

type IgnoreHeadersOption struct {
	keys []string
}

func IgnoreHeaders(keys ...string) *IgnoreHeadersOption {
	return &IgnoreHeadersOption{
		keys: keys,
	}
}

func (o *IgnoreHeadersOption) isHTTP() {}

type IgnoreFieldsOption struct {
	keys []string
}

func IgnoreFields(keys ...string) *IgnoreFieldsOption {
	return &IgnoreFieldsOption{
		keys: keys,
	}
}

func (o *IgnoreFieldsOption) isHTTP() {}
func (o *IgnoreFieldsOption) isJSON() {}

type InspectBodyOption struct {
	fn func(body []byte)
}

func InspectBody(fn func(body []byte)) *InspectBodyOption {
	return &InspectBodyOption{
		fn: fn,
	}
}
func (o *InspectBodyOption) isHTTP() {}
func (o *InspectBodyOption) isJSON() {}

type InspectHeadersOption struct {
	fn func(http.Header)
}

func InspectHeaders(fn func(http.Header)) *InspectHeadersOption {
	return &InspectHeadersOption{
		fn: fn,
	}
}

func (o *InspectHeadersOption) isHTTP() {}

func ignoreMapKeys(keys ...string) cmp.Option {
	return cmpopts.IgnoreMapEntries(func(key string, _ any) bool {
		for _, k := range keys {
			if k == key {
				return true
			}
		}

		return false
	})
}
