package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/alextanhongpin/core/http/httputil"
	"github.com/alextanhongpin/core/types/maputil"
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

type MaskFn func(key string) bool

func (m MaskFn) isJSON() {}

func MaskFields(fields ...string) MaskFn {
	return maputil.MaskFields(fields...)
}

type HTTPInterceptor func(w *http.Response, r *http.Request) error

func (i HTTPInterceptor) isHTTP() {}

func MaskRequestBody(fields ...string) HTTPInterceptor {
	return func(w *http.Response, r *http.Request) error {
		b, err := httputil.ReadRequest(r)
		if err != nil {
			return err
		}

		if json.Valid(b) {
			b, err = maputil.MaskBytes(b, fields...)
			if err != nil {
				return err
			}

			r.Body = io.NopCloser(bytes.NewReader(b))
		}

		return nil
	}
}

func MaskResponseBody(fields ...string) HTTPInterceptor {
	return func(w *http.Response, r *http.Request) error {
		b, err := httputil.ReadResponse(w)
		if err != nil {
			return err
		}

		if json.Valid(b) {
			b, err = maputil.MaskBytes(b, fields...)
			if err != nil {
				return err
			}

			w.Body = io.NopCloser(bytes.NewReader(b))
		}

		return nil
	}
}

func InspectRequestBody(fn func([]byte) error) HTTPInterceptor {
	return func(w *http.Response, r *http.Request) error {
		b, err := httputil.ReadRequest(r)
		if err != nil {
			return err
		}

		return fn(b)
	}
}

func InspectResponseBody(fn func([]byte) error) HTTPInterceptor {
	return func(w *http.Response, r *http.Request) error {
		b, err := httputil.ReadResponse(w)
		if err != nil {
			return err
		}

		return fn(b)
	}
}

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
