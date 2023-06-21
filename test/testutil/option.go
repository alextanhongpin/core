package testutil

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

const TestData = "testdata"
const ExtJSON = ".json"
const ExtHTTP = ".http"

type HTTPOption interface {
	isHTTP()
}

type JSONOption interface {
	isJSON()
}

type FilePath string

func (FilePath) isJSON() {}
func (FilePath) isHTTP() {}

type FileName string

func (FileName) isJSON() {}
func (FileName) isHTTP() {}

type PathOption struct {
	TestDir  string
	FilePath string
	FileName string
	FileExt  string
}

func (o *PathOption) String() string {
	if len(o.FileName) == 0 {
		return filepath.Join(
			o.TestDir,
			o.FilePath+o.FileExt,
		)
	}

	// Get the file extension.
	fileName := string(o.FileName)
	fileExt := filepath.Ext(fileName)
	if fileExt != o.FileExt {
		fileName = fileName + o.FileExt
	}

	return filepath.Join(
		o.TestDir,
		o.FilePath,
		fileName,
	)
}

func NewJSONPath(opts ...JSONOption) *PathOption {
	opt := &PathOption{
		TestDir:  TestData,
		FilePath: "",
		FileName: "",
		FileExt:  ExtJSON,
	}

	for _, o := range opts {
		switch v := o.(type) {
		case FilePath:
			opt.FilePath = strings.TrimSuffix(string(v), "/")
		case FileName:
			opt.FileName = string(v)
		}
	}

	return opt
}

func NewHTTPPath(opts ...HTTPOption) *PathOption {
	opt := &PathOption{
		TestDir:  TestData,
		FilePath: "",
		FileName: "",
		FileExt:  ExtHTTP,
	}

	for _, o := range opts {
		switch v := o.(type) {
		case FilePath:
			opt.FilePath = strings.TrimSuffix(string(v), "/")
		case FileName:
			opt.FileName = string(v)
		}
	}

	return opt
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
