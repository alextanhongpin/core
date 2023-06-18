package testutil

import (
	"fmt"
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
			fileName := string(v)
			// Automatically suffix the filename with the extension ".json"
			// if no extension is provided.
			if ext := filepath.Ext(fileName); ext == "" {
				fileName = fmt.Sprintf("%s.json", fileName)
			}

			to.FileName = fileName
		}
	}

	return to
}

func (o *testOption) String() string {
	return filepath.Join(o.TestDir, o.TestName, o.FileName)
}

type IgnoreHeadersOption []string

func IgnoreHeaders(keys ...string) IgnoreHeadersOption {
	return IgnoreHeadersOption(keys)
}

func (o IgnoreHeadersOption) isHTTP() {}

type IgnoreFieldsOption []string

func IgnoreFields(keys ...string) IgnoreFieldsOption {
	return IgnoreFieldsOption(keys)
}

func (o IgnoreFieldsOption) isHTTP() {}
func (o IgnoreFieldsOption) isJSON() {}

type InspectBody func(body []byte)

func (o InspectBody) isHTTP() {}
func (o InspectBody) isJSON() {}

type InspectHeaders func(headers http.Header, isRequest bool)

func (o InspectHeaders) isHTTP() {}

type CmpOptionsOptions []cmp.Option

func CmpOptions(opts ...cmp.Option) CmpOptionsOptions {
	return CmpOptionsOptions(opts)
}

func (o CmpOptionsOptions) isJSON() {}

type HeaderCmpOptions CmpOptionsOptions

func (o HeaderCmpOptions) isHTTP() {}

type BodyCmpOptions CmpOptionsOptions

func (o BodyCmpOptions) isHTTP() {}

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
