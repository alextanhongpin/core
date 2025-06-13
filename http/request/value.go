package request

import (
	"cmp"
	"encoding/base64"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
)

type Value string

func QueryValue(r *http.Request, name string) Value {
	return Value(r.URL.Query().Get(name))
}

func PathValue(r *http.Request, name string) Value {
	return Value(r.PathValue(name))
}

func FormValue(r *http.Request, name string) Value {
	return Value(r.FormValue(name))
}

func (v Value) String() string {
	return string(v)
}

func (v Value) Int64() int64 {
	return toInt64(v.String())
}

func (v Value) Int64N(n int64) int64 {
	return cmp.Or(v.Int64(), n)
}

func (v Value) Int32() int32 {
	return toInt32(v.String())
}

func (v Value) Int32N(n int32) int32 {
	return cmp.Or(v.Int32(), n)
}

func (v Value) Int() int {
	return toInt(v.String())
}

func (v Value) IntN(n int) int {
	return cmp.Or(v.Int(), n)
}

func (v Value) Bool() bool {
	return toBool(v.String())
}

func (v Value) FromBase64() Value {
	return Value(fromBase64(v.String()))
}

func (v Value) ToBase64() Value {
	return Value(toBase64(v.String()))
}

func toInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func toInt32(s string) int32 {
	i, _ := strconv.ParseInt(s, 10, 32)
	return int32(i)
}

func toInt64(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

func toBool(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}

func fromBase64(s string) string {
	b, _ := base64.URLEncoding.DecodeString(s)
	return string(b)
}

func toBase64(s any) string {
	// Skip zero values. We don't want the user to accidentally use the cursor.
	if isZero(s) {
		return ""
	}

	return base64.URLEncoding.EncodeToString(fmt.Appendf(nil, "%v", s))
}

func isZero(a any) bool {
	return reflect.ValueOf(a).IsZero()
}
