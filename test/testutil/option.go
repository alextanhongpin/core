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

type IgnoreArgsOption []string

func (o IgnoreArgsOption) isSQL() {}
func IgnoreArgs(fields ...string) ArgsCmpOptions {
	return ArgsCmpOptions([]cmp.Option{IgnoreMapKeys(fields...)})
}

type IgnoreRowsOption []string

func (o IgnoreRowsOption) isSQL() {}

func IgnoreRows(fields ...string) RowsCmpOptions {
	return RowsCmpOptions([]cmp.Option{IgnoreMapKeys(fields...)})
}

type InspectQuery func(query string)

func (o InspectQuery) isSQL() {}

type JSONCmpOptions []cmp.Option

func JSONCmpOption(opts ...cmp.Option) JSONCmpOptions {
	return JSONCmpOptions(opts)
}

func IgnoreFields(fields ...string) JSONCmpOptions {
	return JSONCmpOptions([]cmp.Option{IgnoreMapKeys(fields...)})
}

func (o JSONCmpOptions) isJSON() {}

type HeaderCmpOptions []cmp.Option

func (o HeaderCmpOptions) isHTTP() {}

func IgnoreHeaders(keys ...string) HeaderCmpOptions {
	return HeaderCmpOptions([]cmp.Option{IgnoreMapKeys(keys...)})
}

type BodyCmpOptions []cmp.Option

func (o BodyCmpOptions) isHTTP() {}

func IgnoreBodyFields(fields ...string) BodyCmpOptions {
	return BodyCmpOptions([]cmp.Option{IgnoreMapKeys(fields...)})
}

type ArgsCmpOptions []cmp.Option

func (o ArgsCmpOptions) isSQL() {}

type RowsCmpOptions []cmp.Option

func (o RowsCmpOptions) isSQL() {}

type DialectOption string

func (o DialectOption) isSQL() {}

type JSONInspector func([]byte) error

func (i JSONInspector) isJSON() {}

func InspectJSON(fn func([]byte) error) JSONInspector {
	return fn
}

type JSONInterceptor func([]byte) ([]byte, error)

func (i JSONInterceptor) isJSON() {}

func MaskJSON(fields ...string) JSONInterceptor {
	return func(b []byte) ([]byte, error) {
		return maputil.MaskBytes(b, fields...)
	}
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

func MaskRequestHeaders(keys ...string) HTTPInterceptor {
	return func(w *http.Response, r *http.Request) error {
		for _, k := range keys {
			if v := r.Header.Get(k); len(v) > 0 {
				r.Header.Set(k, maputil.MaskValue)
			}
		}

		return nil
	}
}

func MaskResponseHeaders(keys ...string) HTTPInterceptor {
	return func(w *http.Response, r *http.Request) error {
		for _, k := range keys {
			if v := w.Header.Get(k); len(v) > 0 {
				w.Header.Set(k, maputil.MaskValue)
			}
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

func InspectRequestHeaders(fn func(http.Header)) HTTPInterceptor {
	return func(w *http.Response, r *http.Request) error {
		fn(r.Header.Clone())
		return nil
	}
}

func InspectResponseHeaders(fn func(http.Header)) HTTPInterceptor {
	return func(w *http.Response, r *http.Request) error {
		fn(w.Header.Clone())
		return nil
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
