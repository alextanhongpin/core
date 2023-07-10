package testdump

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/alextanhongpin/core/http/httputil"
	"github.com/alextanhongpin/core/types/maputil"
)

var ErrHeaderNotFound = errors.New("testdump: HTTP header not found")

func HTTPCompareHook(hook func(snap, recv *HTTPDump) error) Hook[*HTTPDump] {
	type T = *HTTPDump

	return func(s S[T]) S[T] {
		return &compareHook[T]{
			S:    s,
			hook: hook,
		}
	}
}

func HTTPMarshalHook(hook func(snap *HTTPDump) (*HTTPDump, error)) Hook[*HTTPDump] {
	type T = *HTTPDump

	return func(s S[T]) S[T] {
		return &marshalHook[T]{
			S:    s,
			hook: hook,
		}
	}
}

func MaskRequestHeaders(headers ...string) Hook[*HTTPDump] {
	type T = *HTTPDump

	return func(s S[T]) S[T] {
		return &marshalHook[T]{
			S: s,
			hook: func(t T) (T, error) {
				for _, h := range headers {
					v := t.R.Header.Get(h)
					if v == "" {
						return nil, fmt.Errorf("%w for Request: %q", ErrHeaderNotFound, h)
					}
					t.R.Header.Set(h, maputil.MaskValue)
				}

				return t, nil
			},
		}
	}
}

func MaskResponseHeaders(headers ...string) Hook[*HTTPDump] {
	type T = *HTTPDump

	return func(s S[T]) S[T] {
		return &marshalHook[T]{
			S: s,
			hook: func(t T) (T, error) {
				for _, h := range headers {
					v := t.W.Header.Get(h)
					if v == "" {
						return nil, fmt.Errorf("%w for Response: %q", ErrHeaderNotFound, h)
					}
					t.W.Header.Set(h, maputil.MaskValue)
				}

				return t, nil
			},
		}
	}
}

func MaskRequestBody(fields ...string) Hook[*HTTPDump] {
	type T = *HTTPDump

	return func(s S[T]) S[T] {
		return &marshalHook[T]{
			S: s,
			hook: func(t T) (T, error) {
				b, err := httputil.ReadRequest(t.R)
				if err != nil {
					return nil, err
				}

				if json.Valid(b) {
					b, err := maputil.MaskBytes(b, fields...)
					if err != nil {
						return nil, err
					}
					t.R.Body = io.NopCloser(bytes.NewReader(b))

				}

				return t, nil
			},
		}
	}
}

func MaskResponseBody(fields ...string) Hook[*HTTPDump] {
	type T = *HTTPDump

	return func(s S[T]) S[T] {
		return &marshalHook[T]{
			S: s,
			hook: func(t T) (T, error) {
				b, err := httputil.ReadResponse(t.W)
				if err != nil {
					return nil, err
				}

				if json.Valid(b) {
					b, err := maputil.MaskBytes(b, fields...)
					if err != nil {
						return nil, err
					}
					t.W.Body = io.NopCloser(bytes.NewReader(b))

				}

				return t, nil
			},
		}
	}
}
