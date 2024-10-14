package validator

import (
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/exp/constraints"
)

// Allow names with spaces.
var (
	alpha        = regexp.MustCompile(`^[a-zA-Z]+$`)
	alphaNumeric = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	numeric      = regexp.MustCompile(`^[0-9]+$`)
	email        = regexp.MustCompile("^(?:(?:(?:(?:[a-zA-Z]|\\d|[!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])+(?:\\.([a-zA-Z]|\\d|[!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])+)*)|(?:(?:\\x22)(?:(?:(?:(?:\\x20|\\x09)*(?:\\x0d\\x0a))?(?:\\x20|\\x09)+)?(?:(?:[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x7f]|\\x21|[\\x23-\\x5b]|[\\x5d-\\x7e]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(?:(?:[\\x01-\\x09\\x0b\\x0c\\x0d-\\x7f]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}]))))*(?:(?:(?:\\x20|\\x09)*(?:\\x0d\\x0a))?(\\x20|\\x09)+)?(?:\\x22))))@(?:(?:(?:[a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(?:(?:[a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])(?:[a-zA-Z]|\\d|-|\\.|~|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])*(?:[a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])))\\.)+(?:(?:[a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(?:(?:[a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])(?:[a-zA-Z]|\\d|-|\\.|~|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])*(?:[a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])))\\.?$")
)

var ErrSkip = errors.New("skip")

type Validator[T any] interface {
	Validate(T) error
}

func StringExpr(expr string, fms ...FuncMap[string]) *StringValidator {
	sv := String()
	for _, fm := range fms {
		sv = sv.FuncMap(fm)
	}

	return sv.Parse(expr)
}

func SliceExpr[T any](expr string, fms ...FuncMap[[]T]) *SliceValidator[T] {
	sv := Slice[T]()
	for _, fm := range fms {
		sv = sv.FuncMap(fm)
	}

	return sv.Parse(expr)
}

func NumberExpr[T Numeric](expr string, fms ...FuncMap[T]) *NumberValidator[T] {
	nv := Number[T]()
	for _, fm := range fms {
		nv = nv.FuncMap(fm)
	}

	return nv.Parse(expr)
}

type ParserFunc[T any] func(params string) func(T) error

type FuncMap[T any] map[string]ParserFunc[T]

type StringValidator struct {
	optional bool
	fns      []func(string) error
	fm       FuncMap[string]
}

func String() *StringValidator {
	return &StringValidator{
		fm: make(FuncMap[string]),
	}
}

func (sv *StringValidator) Parse(exprs string) *StringValidator {
	for _, expr := range strings.Split(exprs, ",") {
		k, v, _ := strings.Cut(expr, "=")

		// Custom FuncMap takes precedence.
		if fn, ok := sv.fm[k]; ok {
			sv = sv.Func(fn(v))

			continue
		}

		switch k {
		case "optional":
			sv = sv.Optional()
		case "min":
			sv = sv.Min(toInt(v))
		case "max":
			sv = sv.Max(toInt(v))
		case "len":
			sv = sv.Len(toInt(v))
		case "oneof":
			sv = sv.OneOf(strings.Fields(v)...)
		case "is":
			sv = sv.Is(v)
		case "alpha":
			sv = sv.Alpha()
		case "numeric":
			sv = sv.Numeric()
		case "alphanumeric":
			sv = sv.AlphaNumeric()
		case "email":
			sv = sv.Email()
		case "ip":
			sv = sv.IP()
		case "url":
			sv = sv.URL()
		case "starts_with":
			sv = sv.StartsWith(v)
		case "ends_with":
			sv = sv.EndsWith(v)
		default:
			panic(fmt.Sprintf("unknown expression %q", expr))
		}
	}

	return sv
}

func (sv *StringValidator) Min(n int) *StringValidator {
	sv.fns = append(sv.fns, func(s string) error {
		if len([]rune(s)) < n {
			return fmt.Errorf("min length is %d", n)
		}
		return nil
	})

	return sv
}

func (sv *StringValidator) Max(n int) *StringValidator {
	sv.fns = append(sv.fns, func(s string) error {
		if len([]rune(s)) > n {
			return fmt.Errorf("max length is %d", n)
		}
		return nil
	})

	return sv
}

func (sv *StringValidator) Email() *StringValidator {
	sv.fns = append(sv.fns, func(s string) error {
		if !email.MatchString(s) {
			return errors.New("invalid email")
		}
		return nil
	})

	return sv
}

func (sv *StringValidator) AlphaNumeric() *StringValidator {
	sv.fns = append(sv.fns, func(s string) error {
		if !alphaNumeric.MatchString(s) {
			return errors.New("must be alphanumeric only")
		}
		return nil
	})

	return sv
}

func (sv *StringValidator) Alpha() *StringValidator {
	sv.fns = append(sv.fns, func(s string) error {
		if !alpha.MatchString(s) {
			return errors.New("must be alphabets only")
		}
		return nil
	})

	return sv
}

func (sv *StringValidator) Numeric() *StringValidator {
	sv.fns = append(sv.fns, func(s string) error {
		if !numeric.MatchString(s) {
			return errors.New("must be numbers only")
		}
		return nil
	})

	return sv
}

func (sv *StringValidator) IP() *StringValidator {
	sv.fns = append(sv.fns, func(s string) error {
		if net.ParseIP(s) == nil {
			return errors.New("invalid ip")
		}
		return nil
	})

	return sv
}

func (sv *StringValidator) URL() *StringValidator {
	sv.fns = append(sv.fns, func(s string) error {
		_, err := url.Parse(s)
		if err != nil {
			return errors.New("invalid url")
		}
		return nil
	})

	return sv
}

func (sv *StringValidator) Is(str string) *StringValidator {
	sv.fns = append(sv.fns, func(s string) error {
		if !strings.EqualFold(str, s) {
			return fmt.Errorf("must be %q", str)
		}
		return nil
	})

	return sv
}

func (sv *StringValidator) Len(n int) *StringValidator {
	sv.fns = append(sv.fns, func(s string) error {
		if len([]rune(s)) != n {
			return fmt.Errorf("must have exact length of %d", n)
		}
		return nil
	})

	return sv
}

func (sv *StringValidator) StartsWith(prefix string) *StringValidator {
	sv.fns = append(sv.fns, func(s string) error {
		if !strings.HasPrefix(s, prefix) {
			return fmt.Errorf("must start with %q", prefix)
		}
		return nil
	})

	return sv
}

func (sv *StringValidator) EndsWith(suffix string) *StringValidator {
	sv.fns = append(sv.fns, func(s string) error {
		if !strings.HasSuffix(s, suffix) {
			return fmt.Errorf("must end with %q", suffix)
		}
		return nil
	})

	return sv
}

func (sv *StringValidator) Optional() *StringValidator {
	sv.optional = true

	return sv
}

func (sv *StringValidator) OneOf(vals ...string) *StringValidator {
	sv.fns = append(sv.fns, func(s string) error {
		for _, v := range vals {
			if v == s {
				return nil
			}
		}
		return fmt.Errorf("must be one of %s", strings.Join(vals, ", "))
	})

	return sv
}

func (sv *StringValidator) Regexp(name string, pattern string) *StringValidator {
	// TODO: Store in registry?
	re := regexp.MustCompile(pattern)
	sv.fns = append(sv.fns, func(s string) error {
		if !re.MatchString(s) {
			return fmt.Errorf("pattern does not match %q", name)
		}
		return nil
	})

	return sv
}

func (sv *StringValidator) Func(fn func(string) error) *StringValidator {
	return sv.Funcs(fn)
}

func (sv *StringValidator) Funcs(fns ...func(string) error) *StringValidator {
	sv.fns = append(sv.fns, fns...)

	return sv
}

func (sv *StringValidator) FuncMap(fm FuncMap[string]) *StringValidator {
	for k, fn := range fm {
		sv.fm[k] = fn
	}

	return sv
}

func (sv *StringValidator) Validate(s string) error {
	if s == "" {
		if sv.optional {
			return nil
		}

		return errors.New("must not be empty")
	}

	for _, fn := range sv.fns {
		if err := fn(s); err != nil {
			return err
		}
	}

	return nil
}

type SliceValidator[T any] struct {
	optional bool
	fns      []func([]T) error
	fm       FuncMap[[]T]
}

func Slice[T any]() *SliceValidator[T] {
	return &SliceValidator[T]{
		fm: make(FuncMap[[]T]),
	}
}

func (sv *SliceValidator[T]) Parse(exprs string) *SliceValidator[T] {
	for _, expr := range strings.Split(exprs, ",") {
		k, v, _ := strings.Cut(expr, "=")

		if fn, ok := sv.fm[k]; ok {
			sv = sv.Func(fn(v))

			continue
		}

		switch k {
		case "optional":
			sv = sv.Optional()
		case "min":
			sv = sv.Min(toInt(v))
		case "max":
			sv = sv.Max(toInt(v))
		case "len":
			sv = sv.Len(toInt(v))
		default:
			panic(fmt.Sprintf("unknown expression %q", expr))
		}
	}

	return sv
}

func (sv *SliceValidator[T]) Optional() *SliceValidator[T] {
	sv.optional = true

	return sv
}

func (sv *SliceValidator[T]) Min(n int) *SliceValidator[T] {
	sv.fns = append(sv.fns, func(vs []T) error {
		if len(vs) < n {
			return fmt.Errorf("min items is %d", n)
		}
		return nil
	})

	return sv
}

func (sv *SliceValidator[T]) Max(n int) *SliceValidator[T] {
	sv.fns = append(sv.fns, func(vs []T) error {
		if len(vs) > n {
			return fmt.Errorf("max items is %d", n)
		}
		return nil
	})

	return sv
}

func (sv *SliceValidator[T]) Len(n int) *SliceValidator[T] {
	sv.fns = append(sv.fns, func(vs []T) error {
		if len(vs) != n {
			return fmt.Errorf("number of items must be %d", n)
		}
		return nil
	})

	return sv
}

func (sv *SliceValidator[T]) EachFunc(fn func(T) error) *SliceValidator[T] {
	sv.fns = append(sv.fns, func(vs []T) error {
		for _, v := range vs {
			if err := fn(v); err != nil {
				return err
			}
		}
		return nil
	})

	return sv
}

func (sv *SliceValidator[T]) Each(fn Validator[T]) *SliceValidator[T] {
	sv.fns = append(sv.fns, func(vs []T) error {
		for _, v := range vs {
			if err := fn.Validate(v); err != nil {
				return err
			}
		}
		return nil
	})

	return sv
}

func (sv *SliceValidator[T]) Func(fn func([]T) error) *SliceValidator[T] {
	return sv.Funcs(fn)
}

func (sv *SliceValidator[T]) Funcs(fns ...func([]T) error) *SliceValidator[T] {
	sv.fns = append(sv.fns, fns...)

	return sv
}

func (sv *SliceValidator[T]) FuncMap(fm FuncMap[[]T]) *SliceValidator[T] {
	for k, fn := range fm {
		sv.fm[k] = fn
	}

	return sv
}

func (sv *SliceValidator[T]) Validate(vs []T) error {
	if len(vs) == 0 {
		if sv.optional {
			return nil
		}

		return errors.New("must not be empty")
	}

	for _, fn := range sv.fns {
		if err := fn(vs); err != nil {
			return err
		}
	}

	return nil
}

type Numeric interface {
	constraints.Integer | constraints.Float
}

type NumberValidator[T Numeric] struct {
	optional bool
	fns      []func(T) error
	fm       FuncMap[T]
}

func Number[T Numeric]() *NumberValidator[T] {
	return &NumberValidator[T]{
		fm: make(FuncMap[T]),
	}
}

func (nv *NumberValidator[T]) Parse(exprs string) *NumberValidator[T] {
	for _, expr := range strings.Split(exprs, ",") {
		k, v, _ := strings.Cut(expr, "=")

		if fn, ok := nv.fm[k]; ok {
			nv = nv.Func(fn(v))

			continue
		}

		switch k {
		case "optional":
			nv = nv.Optional()
		case "min":
			nv = nv.Min(T(toFloat64(v)))
		case "max":
			nv = nv.Max(T(toFloat64(v)))
		case "is":
			nv = nv.Is(T(toFloat64(v)))
		case "oneof":
			vs := strings.Fields(v)
			fs := make([]T, len(vs))
			for i, v := range vs {
				fs[i] = T(toFloat64(v))
			}
			nv = nv.OneOf(fs...)
		case "between":
			lo, hi, _ := strings.Cut(v, " ")
			nv = nv.Between(T(toFloat64(lo)), T(toFloat64(hi)))
		case "positive":
			nv = nv.Positive()
		case "negative":
			nv = nv.Negative()
		case "latitude":
			nv = nv.Latitude()
		case "longitude":
			nv = nv.Longitude()
		default:
			panic(fmt.Sprintf("unknown expression %q", expr))
		}
	}

	return nv
}

func (nv *NumberValidator[T]) Optional() *NumberValidator[T] {
	nv.optional = true

	return nv
}

func (nv *NumberValidator[T]) Min(n T) *NumberValidator[T] {
	nv.fns = append(nv.fns, func(v T) error {
		if v < n {
			if n == 1 {
				return errors.New("min 1")
			}
			return fmt.Errorf("min %v", n)
		}
		return nil
	})

	return nv
}

func (nv *NumberValidator[T]) Max(n T) *NumberValidator[T] {
	nv.fns = append(nv.fns, func(v T) error {
		if v > n {
			return fmt.Errorf("max %v", n)
		}
		return nil
	})

	return nv
}

func (nv *NumberValidator[T]) Is(n T) *NumberValidator[T] {
	nv.fns = append(nv.fns, func(v T) error {
		if v != n {
			return fmt.Errorf("must be %v", n)
		}
		return nil
	})

	return nv
}

func (nv *NumberValidator[T]) OneOf(ns ...T) *NumberValidator[T] {
	nv.fns = append(nv.fns, func(v T) error {
		for _, n := range ns {
			if n == v {
				return nil
			}
		}
		s := fmt.Sprint(ns)
		s = s[1 : len(s)-1]
		return fmt.Errorf("must be one of %v", strings.Join(strings.Fields(s), ", "))
	})

	return nv
}

func (nv *NumberValidator[T]) Between(lo, hi T) *NumberValidator[T] {
	nv.fns = append(nv.fns, func(v T) error {
		if v < lo || v > hi {
			return fmt.Errorf("must be between %v and %v", lo, hi)
		}
		return nil
	})

	return nv
}

func (nv *NumberValidator[T]) Positive() *NumberValidator[T] {
	nv.fns = append(nv.fns, func(v T) error {
		var zero T
		if v < zero {
			return fmt.Errorf("must be greater than 0")
		}
		return nil
	})

	return nv
}

func (nv *NumberValidator[T]) Negative() *NumberValidator[T] {
	nv.fns = append(nv.fns, func(v T) error {
		var zero T
		if v > zero {
			return fmt.Errorf("must be less than 0")
		}
		return nil
	})

	return nv
}

func (nv *NumberValidator[T]) Latitude() *NumberValidator[T] {
	nv.fns = append(nv.fns, func(v T) error {
		if math.Abs(float64(v)) > 90.0 {
			return errors.New("must be between -90 and 90")
		}
		return nil
	})

	return nv
}

func (nv *NumberValidator[T]) Longitude() *NumberValidator[T] {
	nv.fns = append(nv.fns, func(v T) error {
		if math.Abs(float64(v)) > 180.0 {
			return errors.New("must be between -180 and 180")
		}
		return nil
	})

	return nv
}

func (nv *NumberValidator[T]) Func(fn func(T) error) *NumberValidator[T] {
	return nv.Funcs(fn)
}

func (nv *NumberValidator[T]) Funcs(fns ...func(T) error) *NumberValidator[T] {
	nv.fns = append(nv.fns, fns...)
	return nv
}

func (nv *NumberValidator[T]) FuncMap(fm FuncMap[T]) *NumberValidator[T] {
	for k, fn := range fm {
		nv.fm[k] = fn
	}

	return nv
}

func (nv *NumberValidator[T]) Validate(v T) error {
	if v == 0 {
		if nv.optional {
			return nil
		}

		return errors.New("must not be zero")
	}

	for _, fn := range nv.fns {
		if err := fn(v); err != nil {
			return err
		}
	}

	return nil
}

func Value[T any](v *T) (t T) {
	if v == nil {
		return
	}
	t = *v

	return
}

func toInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}

	return n
}

func toFloat64(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(err)
	}

	return f
}
