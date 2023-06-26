package testutil

import (
	"net/http"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type HTTPOption interface {
	isHTTP()
}

type JSONOption interface {
	isJSON()
}

type SQLOption interface {
	isSQL()
}

type FilePath string

func (FilePath) isJSON() {}
func (FilePath) isHTTP() {}
func (FilePath) isSQL()  {}

type FileName string

func (FileName) isJSON() {}
func (FileName) isHTTP() {}
func (FileName) isSQL()  {}

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
func (o IgnoreFieldsOption) isSQL()  {}

type IgnoreArgsOption []string

func (o IgnoreArgsOption) isSQL() {}
func IgnoreArgs(keys ...string) IgnoreArgsOption {
	return IgnoreArgsOption(keys)
}

type IgnoreRowsOption []string

func (o IgnoreRowsOption) isSQL() {}
func IgnoreRows(keys ...string) IgnoreRowsOption {
	return IgnoreRowsOption(keys)
}

type InspectBody func(body []byte)

func (o InspectBody) isHTTP() {}
func (o InspectBody) isJSON() {}

type InspectHeaders func(headers http.Header, isRequest bool)

func (o InspectHeaders) isHTTP() {}

type InspectQuery func(query string)

func (o InspectQuery) isSQL() {}

type CmpOptionsOptions []cmp.Option

func CmpOptions(opts ...cmp.Option) CmpOptionsOptions {
	return CmpOptionsOptions(opts)
}

func (o CmpOptionsOptions) isJSON() {}

type HeaderCmpOptions CmpOptionsOptions

func (o HeaderCmpOptions) isHTTP() {}

type BodyCmpOptions CmpOptionsOptions

func (o BodyCmpOptions) isHTTP() {}

type ArgsCmpOptions CmpOptionsOptions

func (o ArgsCmpOptions) isSQL() {}

type RowsCmpOptions CmpOptionsOptions

func (o RowsCmpOptions) isSQL() {}

type DialectOption string

func (o DialectOption) isSQL() {}

func Postgres() DialectOption {
	return DialectOption("postgres")
}

func MySQL() DialectOption {
	return DialectOption("mysql")
}

func IgnoreMapKeys(keys ...string) cmp.Option {
	return cmpopts.IgnoreMapEntries(func(key string, _ any) bool {
		for _, k := range keys {
			if k == key {
				return true
			}
		}

		return false
	})
}
