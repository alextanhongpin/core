package validator

import (
	"errors"
	"fmt"
	"maps"
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

type ValidateFunc[T any] func(T) error

func (val ValidateFunc[T]) Validate(v T) error {
	err := val(v)
	if errors.Is(err, ErrSkip) {
		return nil
	}
	return err
}

func StringExpr(expr string, fms ...FuncMap[string]) *StringBuilder {
	sb := NewStringBuilder()
	for _, fm := range fms {
		sb = sb.FuncMap(fm)
	}

	return sb.Parse(expr)
}

func SliceExpr[T any](expr string, fms ...FuncMap[[]T]) *SliceBuilder[T] {
	sb := NewSliceBuilder[T]()
	for _, fm := range fms {
		sb = sb.FuncMap(fm)
	}

	return sb.Parse(expr)
}

func NumberExpr[T Number](expr string, fms ...FuncMap[T]) *NumberBuilder[T] {
	nb := NewNumberBuilder[T]()
	for _, fm := range fms {
		nb = nb.FuncMap(fm)
	}

	return nb.Parse(expr)
}

type ParserFunc[T any] func(params string) func(T) error

type FuncMap[T any] map[string]ParserFunc[T]

type StringBuilder struct {
	fn func(string) error
	fm FuncMap[string]
}

func NewStringBuilder() *StringBuilder {
	return &StringBuilder{
		fn: func(s string) error {
			return nil
		},
		fm: make(FuncMap[string]),
	}
}

func (sbo *StringBuilder) Parse(exprs string) *StringBuilder {
	sb := sbo.Clone()
	for _, expr := range strings.Split(exprs, ",") {
		k, v, _ := strings.Cut(expr, "=")
		switch k {
		case "required":
			sb = sb.Required()
		case "optional":
			sb = sb.Optional()
		case "min":
			sb = sb.Min(toInt(v))
		case "max":
			sb = sb.Max(toInt(v))
		case "len":
			sb = sb.Len(toInt(v))
		case "oneof":
			sb = sb.OneOf(strings.Fields(v)...)
		case "is":
			sb = sb.Is(v)
		case "alpha":
			sb = sb.Alpha()
		case "numeric":
			sb = sb.Numeric()
		case "alphanumeric":
			sb = sb.AlphaNumeric()
		case "email":
			sb = sb.Email()
		case "ip":
			sb = sb.IP()
		case "url":
			sb = sb.URL()
		case "starts_with":
			sb = sb.StartsWith(v)
		case "ends_with":
			sb = sb.EndsWith(v)
		default:
			fn, ok := sb.fm[k]
			if !ok {
				panic(fmt.Sprintf("unknown expression %q", expr))
			}
			sb = sb.Func(fn(v))
		}
	}

	return sb
}

func (sb *StringBuilder) Min(n int) *StringBuilder {
	return sb.clone(func(s string) error {
		if len([]rune(s)) < n {
			if n == 1 {
				return errors.New("min 1 character")
			}

			return fmt.Errorf("min %d characters", n)
		}
		return nil
	})
}

func (sb *StringBuilder) Max(n int) *StringBuilder {
	return sb.clone(func(s string) error {
		if len([]rune(s)) > n {
			if n == 1 {
				return errors.New("max 1 character")
			}
			return fmt.Errorf("max %d characters", n)
		}
		return nil
	})
}

func (sb *StringBuilder) Email() *StringBuilder {
	return sb.clone(func(s string) error {
		if !email.MatchString(s) {
			return errors.New("invalid email")
		}
		return nil
	})
}

func (sb *StringBuilder) AlphaNumeric() *StringBuilder {
	return sb.clone(func(s string) error {
		if !alphaNumeric.MatchString(s) {
			return errors.New("must be alphanumeric only")
		}
		return nil
	})
}

func (sb *StringBuilder) Alpha() *StringBuilder {
	return sb.clone(func(s string) error {
		if !alpha.MatchString(s) {
			return errors.New("must be alphabets only")
		}
		return nil
	})
}

func (sb *StringBuilder) Numeric() *StringBuilder {
	return sb.clone(func(s string) error {
		if !numeric.MatchString(s) {
			return errors.New("must be numbers only")
		}
		return nil
	})
}

func (sb *StringBuilder) IP() *StringBuilder {
	return sb.clone(func(s string) error {
		if net.ParseIP(s) == nil {
			return errors.New("invalid ip")
		}
		return nil
	})
}

func (sb *StringBuilder) URL() *StringBuilder {
	return sb.clone(func(s string) error {
		_, err := url.Parse(s)
		if err != nil {
			return errors.New("invalid url")
		}
		return nil
	})
}

func (sb *StringBuilder) Is(str string) *StringBuilder {
	return sb.clone(func(s string) error {
		if !strings.EqualFold(str, s) {
			return fmt.Errorf("must be %q", str)
		}
		return nil
	})
}

func (sb *StringBuilder) Len(n int) *StringBuilder {
	return sb.clone(func(s string) error {
		if len([]rune(s)) != n {
			if n == 1 {
				return errors.New("must be 1 character")
			}

			return fmt.Errorf("must be %d characters", n)
		}
		return nil
	})
}

func (sb *StringBuilder) Required() *StringBuilder {
	return sb.clone(func(s string) error {
		if s == "" {
			return errors.New("must not be empty")
		}
		return nil
	})
}

func (sb *StringBuilder) StartsWith(prefix string) *StringBuilder {
	return sb.clone(func(s string) error {
		if !strings.HasPrefix(s, prefix) {
			return fmt.Errorf("must start with %q", prefix)
		}
		return nil
	})
}

func (sb *StringBuilder) EndsWith(suffix string) *StringBuilder {
	return sb.clone(func(s string) error {
		if !strings.HasSuffix(s, suffix) {
			return fmt.Errorf("must end with %q", suffix)
		}
		return nil
	})
}

func (sb *StringBuilder) Optional() *StringBuilder {
	return sb.clone(func(s string) error {
		if s == "" {
			return ErrSkip
		}
		return nil
	})
}

func (sb *StringBuilder) OneOf(vals ...string) *StringBuilder {
	return sb.clone(func(s string) error {
		for _, v := range vals {
			if v == s {
				return nil
			}
		}
		return fmt.Errorf("must be one of %s", strings.Join(vals, ", "))
	})
}

func (sb *StringBuilder) Regexp(name string, pattern string) *StringBuilder {
	re := regexp.MustCompile(pattern)
	return sb.clone(func(s string) error {
		if !re.MatchString(s) {
			return fmt.Errorf("pattern does not match %q", name)
		}
		return nil
	})
}

func (sb *StringBuilder) Func(fn func(string) error) *StringBuilder {
	return sb.Funcs(fn)
}

func (sbo *StringBuilder) Funcs(fns ...func(string) error) *StringBuilder {
	sb := sbo.Clone()
	for _, fn := range fns {
		sb = sb.clone(fn)
	}

	return sb
}

func (sbo *StringBuilder) FuncMap(fm FuncMap[string]) *StringBuilder {
	sb := sbo.Clone()
	for k, fn := range fm {
		sb.fm[k] = fn
	}

	return sb
}

func (sb *StringBuilder) Validate(s string) error {
	//return sb.Build().Validate(s)
	return ValidateFunc[string](sb.fn).Validate(s)
}

func (sb *StringBuilder) Clone() *StringBuilder {
	sbc := NewStringBuilder()
	sbc.fn = sb.fn
	sbc.fm = maps.Clone(sb.fm)
	return sbc
}

func (sb *StringBuilder) clone(fn func(string) error) *StringBuilder {
	sbc := sb.Clone()
	sbc.fn = seq(sb.fn, fn)
	return sbc
}

func seq[T any](fn func(T) error, fns ...func(T) error) func(T) error {
	if len(fns) == 0 {
		return fn
	}

	return seq(func(v T) error {
		if err := fn(v); err != nil {
			return err
		}
		return fns[0](v)
	}, fns[1:]...)
}

type SliceBuilder[T any] struct {
	fn func([]T) error
	fm FuncMap[[]T]
}

func NewSliceBuilder[T any]() *SliceBuilder[T] {
	return &SliceBuilder[T]{
		fn: func(vs []T) error {
			return nil
		},
		fm: make(FuncMap[[]T]),
	}
}

func (sb *SliceBuilder[T]) Clone() *SliceBuilder[T] {
	sbc := NewSliceBuilder[T]()
	sbc.fn = sb.fn
	sbc.fm = maps.Clone(sb.fm)
	return sbc
}

func (sb *SliceBuilder[T]) clone(fn func([]T) error) *SliceBuilder[T] {
	sbc := sb.Clone()
	sbc.fn = seq(sb.fn, fn)
	return sbc
}

func (sbo *SliceBuilder[T]) Parse(exprs string) *SliceBuilder[T] {
	sb := sbo.Clone()
	for _, expr := range strings.Split(exprs, ",") {
		k, v, _ := strings.Cut(expr, "=")
		switch k {
		case "required":
			sb = sb.Required()
		case "optional":
			sb = sb.Optional()
		case "min":
			sb = sb.Min(toInt(v))
		case "max":
			sb = sb.Max(toInt(v))
		case "len":
			sb = sb.Len(toInt(v))
		default:
			fn, ok := sb.fm[k]
			if !ok {
				panic(fmt.Sprintf("unknown expression %q", expr))
			}
			sb = sb.Func(fn(v))
		}
	}

	return sb
}

func (sb *SliceBuilder[T]) Required() *SliceBuilder[T] {
	return sb.clone(func(vs []T) error {
		if len(vs) == 0 {
			return errors.New("must not be empty")
		}
		return nil
	})
}

func (sb *SliceBuilder[T]) Optional() *SliceBuilder[T] {
	return sb.clone(func(vs []T) error {
		if len(vs) == 0 {
			return ErrSkip
		}
		return nil
	})
}

func (sb *SliceBuilder[T]) Min(n int) *SliceBuilder[T] {
	return sb.clone(func(vs []T) error {
		if len(vs) < n {
			if n == 1 {
				return errors.New("min 1 item")
			}
			return fmt.Errorf("min %d items", n)
		}
		return nil
	})
}

func (sb *SliceBuilder[T]) Max(n int) *SliceBuilder[T] {
	return sb.clone(func(vs []T) error {
		if len(vs) > n {
			if n == 1 {
				return errors.New("max 1 item")
			}

			return fmt.Errorf("max %d items", n)
		}
		return nil
	})
}

func (sb *SliceBuilder[T]) Len(n int) *SliceBuilder[T] {
	return sb.clone(func(vs []T) error {
		if len(vs) != n {
			if n == 1 {
				return errors.New("must have 1 item")
			}

			return fmt.Errorf("must have %d items", n)
		}
		return nil
	})
}

func (sb *SliceBuilder[T]) EachFunc(fn func(T) error) *SliceBuilder[T] {
	return sb.clone(func(vs []T) error {
		for _, v := range vs {
			if err := fn(v); err != nil {
				return err
			}
		}
		return nil
	})
}

func (sb *SliceBuilder[T]) Each(fn Validator[T]) *SliceBuilder[T] {
	return sb.clone(func(vs []T) error {
		for _, v := range vs {
			if err := fn.Validate(v); err != nil {
				return err
			}
		}
		return nil
	})
}

func (sb *SliceBuilder[T]) Func(fn func([]T) error) *SliceBuilder[T] {
	return sb.Funcs(fn)
}

func (sbo *SliceBuilder[T]) Funcs(fns ...func([]T) error) *SliceBuilder[T] {
	sb := sbo.Clone()
	for _, fn := range fns {
		sb.fn = seq(sb.fn, fn)
	}

	return sb
}

func (sbo *SliceBuilder[T]) FuncMap(fm FuncMap[[]T]) *SliceBuilder[T] {
	sb := sbo.Clone()
	for k, fn := range fm {
		sb.fm[k] = fn
	}

	return sb
}

func (sb *SliceBuilder[T]) Validate(vs []T) error {
	return ValidateFunc[[]T](sb.fn).Validate(vs)
}

type Number interface {
	constraints.Integer | constraints.Float
}

type NumberBuilder[T Number] struct {
	fn func(T) error
	fm FuncMap[T]
}

func NewNumberBuilder[T Number]() *NumberBuilder[T] {
	return &NumberBuilder[T]{
		fn: func(v T) error {
			return nil
		},
		fm: make(FuncMap[T]),
	}
}

func (nbo *NumberBuilder[T]) Parse(exprs string) *NumberBuilder[T] {
	nb := nbo.Clone()
	for _, expr := range strings.Split(exprs, ",") {
		k, v, _ := strings.Cut(expr, "=")
		switch k {
		case "required":
			nb = nb.Required()
		case "optional":
			nb = nb.Optional()
		case "min":
			nb = nb.Min(T(toFloat64(v)))
		case "max":
			nb = nb.Max(T(toFloat64(v)))
		case "is":
			nb = nb.Is(T(toFloat64(v)))
		case "oneof":
			vs := strings.Fields(v)
			fs := make([]T, len(vs))
			for i, v := range vs {
				fs[i] = T(toFloat64(v))
			}
			nb = nb.OneOf(fs...)
		case "between":
			lo, hi, _ := strings.Cut(v, " ")
			nb = nb.Between(T(toFloat64(lo)), T(toFloat64(hi)))
		case "positive":
			nb = nb.Positive()
		case "negative":
			nb = nb.Negative()
		case "latitude":
			nb = nb.Latitude()
		case "longitude":
			nb = nb.Longitude()
		default:
			fn, ok := nb.fm[k]
			if !ok {
				panic(fmt.Sprintf("unknown expression %q", expr))
			}
			nb = nb.Func(fn(v))
		}
	}

	return nb
}

func (nb *NumberBuilder[T]) Required() *NumberBuilder[T] {
	return nb.clone(func(v T) error {
		var zero T
		if v == zero {
			return errors.New("must not be zero")
		}
		return nil
	})
}

func (nb *NumberBuilder[T]) Optional() *NumberBuilder[T] {
	return nb.clone(func(v T) error {
		var zero T
		if v == zero {
			return ErrSkip
		}
		return nil
	})
}

func (nb *NumberBuilder[T]) Min(n T) *NumberBuilder[T] {
	return nb.clone(func(v T) error {
		if v < n {
			if n == 1 {
				return errors.New("min 1")
			}
			return fmt.Errorf("min %v", n)
		}
		return nil
	})
}

func (nb *NumberBuilder[T]) Max(n T) *NumberBuilder[T] {
	return nb.clone(func(v T) error {
		if v > n {
			return fmt.Errorf("max %v", n)
		}
		return nil
	})
}

func (nb *NumberBuilder[T]) Is(n T) *NumberBuilder[T] {
	return nb.clone(func(v T) error {
		if v != n {
			return fmt.Errorf("must be %v", n)
		}
		return nil
	})
}

func (nb *NumberBuilder[T]) OneOf(ns ...T) *NumberBuilder[T] {
	return nb.clone(func(v T) error {
		for _, n := range ns {
			if n == v {
				return nil
			}
		}
		s := fmt.Sprint(ns)
		s = s[1 : len(s)-1]
		return fmt.Errorf("must be one of %v", strings.Join(strings.Fields(s), ", "))
	})
}

func (nb *NumberBuilder[T]) Between(lo, hi T) *NumberBuilder[T] {
	return nb.clone(func(v T) error {
		if v < lo || v > hi {
			return fmt.Errorf("must be between %v and %v", lo, hi)
		}
		return nil
	})
}

func (nb *NumberBuilder[T]) Positive() *NumberBuilder[T] {
	return nb.clone(func(v T) error {
		var zero T
		if v < zero {
			return fmt.Errorf("must be greater than 0")
		}
		return nil
	})
}

func (nb *NumberBuilder[T]) Negative() *NumberBuilder[T] {
	return nb.clone(func(v T) error {
		var zero T
		if v > zero {
			return fmt.Errorf("must be less than 0")
		}
		return nil
	})
}

func (nb *NumberBuilder[T]) Latitude() *NumberBuilder[T] {
	return nb.clone(func(v T) error {
		if math.Abs(float64(v)) > 90.0 {
			return errors.New("must be between -90 and 90")
		}
		return nil
	})
}

func (nb *NumberBuilder[T]) Longitude() *NumberBuilder[T] {
	return nb.clone(func(v T) error {
		if math.Abs(float64(v)) > 180.0 {
			return errors.New("must be between -180 and 180")
		}
		return nil
	})
}

func (nb *NumberBuilder[T]) Func(fn func(T) error) *NumberBuilder[T] {
	return nb.Funcs(fn)
}

func (nbo *NumberBuilder[T]) Funcs(fns ...func(T) error) *NumberBuilder[T] {
	nb := nbo.Clone()
	for _, fn := range fns {
		nb.fn = seq(nb.fn, fn)
	}
	return nb
}

func (nbo *NumberBuilder[T]) FuncMap(fm FuncMap[T]) *NumberBuilder[T] {
	nb := nbo.Clone()
	for k, fn := range fm {
		nb.fm[k] = fn
	}

	return nb
}

func (nb *NumberBuilder[T]) Clone() *NumberBuilder[T] {
	nbc := NewNumberBuilder[T]()
	nbc.fn = nb.fn
	nbc.fm = maps.Clone(nb.fm)
	return nbc
}

func (nb *NumberBuilder[T]) clone(fn func(T) error) *NumberBuilder[T] {
	nbc := nb.Clone()
	nbc.fn = seq(nb.fn, fn)
	return nbc
}

func (nb *NumberBuilder[T]) Validate(v T) error {
	return ValidateFunc[T](nb.fn).Validate(v)
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
