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

type StringBuilder struct {
	fn func(string) error
}

func NewStringBuilder() *StringBuilder {
	return &StringBuilder{
		fn: func(s string) error {
			return nil
		},
	}
}

func (sb *StringBuilder) Parse(exprs string) *StringBuilder {
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
			panic(fmt.Sprintf("unknown expression %q", expr))
		}
	}

	return sb
}

func (sb *StringBuilder) Min(n int) *StringBuilder {
	sb.fn = seq(sb.fn, func(s string) error {
		if len([]rune(s)) < n {
			if n == 1 {
				return errors.New("min 1 character")
			}

			return fmt.Errorf("min %d characters", n)
		}
		return nil
	})
	return sb
}

func (sb *StringBuilder) Max(n int) *StringBuilder {
	sb.fn = seq(sb.fn, func(s string) error {
		if len([]rune(s)) > n {
			if n == 1 {
				return errors.New("max 1 character")
			}
			return fmt.Errorf("max %d characters", n)
		}
		return nil
	})
	return sb
}

func (sb *StringBuilder) Email() *StringBuilder {
	sb.fn = seq(sb.fn, func(s string) error {
		if !email.MatchString(s) {
			return errors.New("invalid email")
		}
		return nil
	})
	return sb
}

func (sb *StringBuilder) AlphaNumeric() *StringBuilder {
	sb.fn = seq(sb.fn, func(s string) error {
		if !alphaNumeric.MatchString(s) {
			return errors.New("must be alphanumeric only")
		}
		return nil
	})
	return sb
}

func (sb *StringBuilder) Alpha() *StringBuilder {
	sb.fn = seq(sb.fn, func(s string) error {
		if !alpha.MatchString(s) {
			return errors.New("must be alphabets only")
		}
		return nil
	})
	return sb
}

func (sb *StringBuilder) Numeric() *StringBuilder {
	sb.fn = seq(sb.fn, func(s string) error {
		if !numeric.MatchString(s) {
			return errors.New("must be numbers only")
		}
		return nil
	})
	return sb
}

func (sb *StringBuilder) IP() *StringBuilder {
	sb.fn = seq(sb.fn, func(s string) error {
		if net.ParseIP(s) == nil {
			return errors.New("invalid ip")
		}
		return nil
	})
	return sb
}

func (sb *StringBuilder) URL() *StringBuilder {
	sb.fn = seq(sb.fn, func(s string) error {
		_, err := url.Parse(s)
		if err != nil {
			return errors.New("invalid url")
		}
		return nil
	})
	return sb
}

func (sb *StringBuilder) Is(str string) *StringBuilder {
	sb.fn = seq(sb.fn, func(s string) error {
		if !strings.EqualFold(str, s) {
			return fmt.Errorf("must be %q", str)
		}
		return nil
	})
	return sb
}

func (sb *StringBuilder) Len(n int) *StringBuilder {
	sb.fn = seq(sb.fn, func(s string) error {
		if len([]rune(s)) != n {
			if n == 1 {
				return errors.New("must be 1 character")
			}

			return fmt.Errorf("must be %d characters", n)
		}
		return nil
	})
	return sb
}

func (sb *StringBuilder) Required() *StringBuilder {
	sb.fn = seq(sb.fn, func(s string) error {
		if s == "" {
			return errors.New("must not be empty")
		}
		return nil
	})
	return sb
}

func (sb *StringBuilder) StartsWith(prefix string) *StringBuilder {
	sb.fn = seq(sb.fn, func(s string) error {
		if !strings.HasPrefix(s, prefix) {
			return fmt.Errorf("must start with %q", prefix)
		}
		return nil
	})
	return sb
}

func (sb *StringBuilder) EndsWith(suffix string) *StringBuilder {
	sb.fn = seq(sb.fn, func(s string) error {
		if !strings.HasSuffix(s, suffix) {
			return fmt.Errorf("must end with %q", suffix)
		}
		return nil
	})
	return sb
}

func (sb *StringBuilder) Optional() *StringBuilder {
	sb.fn = seq(sb.fn, func(s string) error {
		if s == "" {
			return ErrSkip
		}
		return nil
	})
	return sb
}

func (sb *StringBuilder) OneOf(vals ...string) *StringBuilder {
	sb.fn = seq(sb.fn, func(s string) error {
		for _, v := range vals {
			if v == s {
				return nil
			}
		}
		return fmt.Errorf("must be one of %s", strings.Join(vals, ", "))
	})
	return sb
}

func (sb *StringBuilder) Regexp(name string, pattern string) *StringBuilder {
	re := regexp.MustCompile(pattern)
	sb.fn = seq(sb.fn, func(s string) error {
		if !re.MatchString(s) {
			return fmt.Errorf("pattern does not match %q", name)
		}
		return nil
	})
	return sb
}

func (sb *StringBuilder) Func(fn func(string) error) *StringBuilder {
	sb.fn = seq(sb.fn, fn)
	return sb
}

func (sb *StringBuilder) Build() func(s string) error {
	return func(s string) error {
		if err := sb.fn(s); err != nil {
			if errors.Is(err, ErrSkip) {
				return nil
			}
			return err
		}
		return nil
	}
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
}

func NewSliceBuilder[T any]() *SliceBuilder[T] {
	return &SliceBuilder[T]{
		fn: func(vs []T) error {
			return nil
		},
	}
}

func (sb *SliceBuilder[T]) Parse(exprs string) *SliceBuilder[T] {
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
			panic(fmt.Sprintf("unknown expression %q", expr))
		}
	}

	return sb
}

func (sb *SliceBuilder[T]) Required() *SliceBuilder[T] {
	sb.fn = seq(sb.fn, func(vs []T) error {
		if len(vs) == 0 {
			return errors.New("must not be empty")
		}
		return nil
	})
	return sb
}

func (sb *SliceBuilder[T]) Optional() *SliceBuilder[T] {
	sb.fn = seq(sb.fn, func(vs []T) error {
		if len(vs) == 0 {
			return ErrSkip
		}
		return nil
	})
	return sb
}

func (sb *SliceBuilder[T]) Min(n int) *SliceBuilder[T] {
	sb.fn = seq(sb.fn, func(vs []T) error {
		if len(vs) < n {
			if n == 1 {
				return errors.New("min 1 item")
			}
			return fmt.Errorf("min %d items", n)
		}
		return nil
	})
	return sb
}

func (sb *SliceBuilder[T]) Max(n int) *SliceBuilder[T] {
	sb.fn = seq(sb.fn, func(vs []T) error {
		if len(vs) > n {
			if n == 1 {
				return errors.New("max 1 item")
			}

			return fmt.Errorf("max %d items", n)
		}
		return nil
	})
	return sb
}

func (sb *SliceBuilder[T]) Len(n int) *SliceBuilder[T] {
	sb.fn = seq(sb.fn, func(vs []T) error {
		if len(vs) != n {
			if n == 1 {
				return errors.New("must have 1 item")
			}

			return fmt.Errorf("must have %d items", n)
		}
		return nil
	})
	return sb
}

func (sb *SliceBuilder[T]) Each(fn func(T) error) *SliceBuilder[T] {
	sb.fn = seq(sb.fn, func(vs []T) error {
		for _, v := range vs {
			if err := fn(v); err != nil {
				return err
			}
		}
		return nil
	})
	return sb
}

func (sb *SliceBuilder[T]) Func(fn func([]T) error) *SliceBuilder[T] {
	sb.fn = seq(sb.fn, fn)
	return sb
}

func (sb *SliceBuilder[T]) Build() func(vs []T) error {
	return func(vs []T) error {
		if err := sb.fn(vs); err != nil {
			if errors.Is(err, ErrSkip) {
				return nil
			}
			return err
		}
		return nil
	}
}

type Number interface {
	constraints.Integer | constraints.Float
}

type NumberBuilder[T Number] struct {
	fn func(T) error
}

func NewNumberBuilder[T Number]() *NumberBuilder[T] {
	return &NumberBuilder[T]{
		fn: func(v T) error {
			return nil
		},
	}
}

func (nb *NumberBuilder[T]) Parse(exprs string) *NumberBuilder[T] {
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
			panic(fmt.Sprintf("unknown expression %q", expr))
		}
	}

	return nb
}

func (nb *NumberBuilder[T]) Required() *NumberBuilder[T] {
	nb.fn = seq(nb.fn, func(v T) error {
		var zero T
		if v == zero {
			return errors.New("must not be zero")
		}
		return nil
	})
	return nb
}

func (nb *NumberBuilder[T]) Optional() *NumberBuilder[T] {
	nb.fn = seq(nb.fn, func(v T) error {
		var zero T
		if v == zero {
			return ErrSkip
		}
		return nil
	})
	return nb
}

func (nb *NumberBuilder[T]) Min(n T) *NumberBuilder[T] {
	nb.fn = seq(nb.fn, func(v T) error {
		if v < n {
			if n == 1 {
				return errors.New("min 1")
			}
			return fmt.Errorf("min %v", n)
		}
		return nil
	})
	return nb
}

func (nb *NumberBuilder[T]) Max(n T) *NumberBuilder[T] {
	nb.fn = seq(nb.fn, func(v T) error {
		if v > n {
			return fmt.Errorf("max %v", n)
		}
		return nil
	})
	return nb
}

func (nb *NumberBuilder[T]) Is(n T) *NumberBuilder[T] {
	nb.fn = seq(nb.fn, func(v T) error {
		if v != n {
			return fmt.Errorf("must be %v", n)
		}
		return nil
	})
	return nb
}

func (nb *NumberBuilder[T]) OneOf(ns ...T) *NumberBuilder[T] {
	nb.fn = seq(nb.fn, func(v T) error {
		for _, n := range ns {
			if n == v {
				return nil
			}
		}
		s := fmt.Sprint(ns)
		s = s[1 : len(s)-1]
		return fmt.Errorf("must be one of %v", strings.Join(strings.Fields(s), ", "))
	})
	return nb
}

func (nb *NumberBuilder[T]) Between(lo, hi T) *NumberBuilder[T] {
	nb.fn = seq(nb.fn, func(v T) error {
		if v < lo || v > hi {
			return fmt.Errorf("must be between %v and %v", lo, hi)
		}
		return nil
	})
	return nb
}

func (nb *NumberBuilder[T]) Positive() *NumberBuilder[T] {
	nb.fn = seq(nb.fn, func(v T) error {
		var zero T
		if v < zero {
			return fmt.Errorf("must be greater than 0")
		}
		return nil
	})
	return nb
}

func (nb *NumberBuilder[T]) Negative() *NumberBuilder[T] {
	nb.fn = seq(nb.fn, func(v T) error {
		var zero T
		if v > zero {
			return fmt.Errorf("must be less than 0")
		}
		return nil
	})
	return nb
}

func (nb *NumberBuilder[T]) Latitude() *NumberBuilder[T] {
	nb.fn = seq(nb.fn, func(v T) error {
		if math.Abs(float64(v)) > 90.0 {
			return errors.New("must be between -90 and 90")
		}
		return nil
	})
	return nb
}

func (nb *NumberBuilder[T]) Longitude() *NumberBuilder[T] {
	nb.fn = seq(nb.fn, func(v T) error {
		if math.Abs(float64(v)) > 180.0 {
			return errors.New("must be between -180 and 180")
		}
		return nil
	})
	return nb
}

func (nb *NumberBuilder[T]) Func(fn func(T) error) *NumberBuilder[T] {
	nb.fn = seq(nb.fn, fn)
	return nb
}

func (nb *NumberBuilder[T]) Build() func(T) error {
	return func(v T) error {
		if err := nb.fn(v); err != nil {
			if errors.Is(err, ErrSkip) {
				return nil
			}
			return err
		}
		return nil
	}
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
