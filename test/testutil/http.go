package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/alextanhongpin/core/http/httputil"
	"github.com/alextanhongpin/core/types/maputil"
	"github.com/google/go-cmp/cmp"
)

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
