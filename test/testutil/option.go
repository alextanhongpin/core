package testutil

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

const TestData = "testdata"

type HTTPOption interface {
	isHTTP()
}

type JSONOption interface {
	isJSON()
}

type TestDir string

func (TestDir) isJSON() {}
func (TestDir) isHTTP() {}

type FilePath string

func (FilePath) isJSON() {}
func (FilePath) isHTTP() {}

type FileName string

func (FileName) isJSON() {}
func (FileName) isHTTP() {}

type FileExt string

func (FileExt) isJSON() {}
func (FileExt) isHTTP() {}

type PathOption struct {
	TestDir  TestDir
	FilePath FilePath
	FileName FileName
	FileExt  FileExt
}

func NewPathOption(opts ...JSONOption) *PathOption {
	opt := &PathOption{
		TestDir:  TestData,
		FilePath: "",
		FileExt:  "",
		FileName: "",
	}

	for _, o := range opts {
		switch v := o.(type) {
		case TestDir:
			opt.TestDir = v
		case FilePath:
			opt.FilePath = v
		case FileName:
			opt.FileName = v
		case FileExt:
			if len(v) == 0 {
				continue
			}

			if v[0] == '.' {
				v = v[1:]
			}

			opt.FileExt = v
		}
	}

	return opt
}

func (o *PathOption) String() string {
	// Get the file extension.
	fileExt := filepath.Ext(string(o.FileName))

	// Filename without extension.
	fileName := o.FileName[:len(o.FileName)-len(fileExt)]
	if len(o.FileExt) > 0 {
		fileExt = string(o.FileExt)
	} else if len(fileExt) > 0 {
		fileExt = fileExt[1:]
	}

	return filepath.Join(
		string(o.TestDir),
		string(o.FilePath),
		fmt.Sprintf("%s.%s", fileName, fileExt),
	)
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
