package validator_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/validator"
	"github.com/stretchr/testify/assert"
)

func TestStringValidator(t *testing.T) {
	test := func(name, expr, input string, want string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			validate := validator.StringExpr(expr)
			got := validate.Validate(input)
			is := assert.New(t)
			if want == "" {
				is.Nil(got)
			} else {
				is.Equal(got.Error(), want)
			}
			t.Log(expr, input, got)
		})
	}

	test("required", "required", "", "must not be empty")
	test("optional", "optional", "", "")
	test("is valid", "is=green", "green", "")
	test("is invalid", "is=green", "blue", `must be "green"`)
	test("min valid", "min=1", "a", "")
	test("min invalid", "min=1", "", "min 1 character")
	test("max valid", "max=1", "a", "")
	test("max invalid", "max=1", "ab", "max 1 character")
	test("len valid", "len=3", "abc", "")
	test("len invalid", "len=3", "abcd", "must be 3 characters")
	test("oneof valid", "oneof=a b c", "a", "")
	test("oneof invalid", "oneof=a b c", "ab", "must be one of a, b, c")
	test("alpha valid", "alpha", "a", "")
	test("alpha invalid", "alpha", "0", "must be alphabets only")
	test("numeric valid", "numeric", "0", "")
	test("numeric invalid", "numeric", "a", "must be numbers only")
	test("alphanumeric valid", "alphanumeric", "0aZ", "")
	test("alphanumeric invalid", "alphanumeric", "#$", "must be alphanumeric only")
	test("email valid", "email", "john.doe@mail.com", "")
	test("email invalid", "email", "johndoe", "invalid email")
	test("ip valid", "ip", "1.1.1.1", "")
	test("ip invalid", "ip", "johndoe", "invalid ip")
	test("url valid", "url", "http://localhost:8080", "")
	test("url invalid", "url", "https://www.percent-off.com/_20_%+off_60000_", "invalid url")
	test("starts_with valid", "starts_with=john", "john doe", "")
	test("starts_with invalid", "starts_with=john", "jane doe", `must start with "john"`)
	test("ends_with valid", "ends_with=doe", "john doe", "")
	test("ends_with invalid", "ends_with=doe", "john appleseed", `must end with "doe"`)
	test("chain valid", "required,email,starts_with=john,ends_with=@mail.com", "john.appleseed@mail.com", "")
	test("chain invalid", "required,email,starts_with=john,ends_with=@mail.com", "", "must not be empty")
	test("chain invalid", "required,email,starts_with=john,ends_with=@mail.com", "john doe", "invalid email")
	test("chain invalid", "required,email,starts_with=john,ends_with=@mail.com", "jane.doe@mail.com", `must start with "john"`)
	test("chain invalid", "required,email,starts_with=john,ends_with=@mail.com", "john.doe@yourmail.com", `must end with "@mail.com"`)
}

func TestStringFuncMap(t *testing.T) {
	toInt := func(s string) int {
		n, err := strconv.Atoi(s)
		if err != nil {
			panic(err)
		}
		return n
	}

	fm := validator.FuncMap[string]{
		"repeat": func(params string) validator.ValidatorFunc[string] {
			ch, ns, _ := strings.Cut(params, " ")
			n := toInt(ns)

			return func(v string) error {
				if strings.Repeat(ch, n) != v {
					return errors.New("does not repeat")
				}

				return nil
			}
		},
	}

	val := validator.StringExpr("repeat=# 10", fm)

	is := assert.New(t)
	is.Nil(val.Validate("##########"))
	is.Equal("does not repeat", val.Validate("1").Error())
}

func TestStringValidatorClone(t *testing.T) {
	val := validator.StringExpr("required,min=5")
	max10 := val.Clone().Max(10)
	max20 := val.Clone().Max(20)
	is := assert.New(t)
	is.Equal("max 10 characters", max10.Validate("abcdefghijklmnopqrsuvwxyz").Error())
	is.Equal("max 20 characters", max20.Validate("abcdefghijklmnopqrsuvwxyz").Error())
}

func TestNumberValidator(t *testing.T) {
	test := func(name, expr string, input int, want string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			validate := validator.NumberExpr[int](expr)
			got := validate.Validate(input)
			is := assert.New(t)
			if want == "" {
				is.Nil(got)
			} else {
				is.Equal(got.Error(), want)
			}
			t.Log(expr, input, got)
		})
	}
	test("required valid", "required", 1, "")
	test("required invalid", "required", 0, "must not be zero")
	test("optional valid", "optional", 0, "")
	test("min valid", "min=1", 1, "")
	test("min invalid", "min=1", 0, "min 1")
	test("max valid", "max=1", 1, "")
	test("max invalid", "max=1", 2, "max 1")
	test("is valid", "is=1", 1, "")
	test("is invalid", "is=1", 2, "must be 1")
	test("oneof valid", "oneof=1 2 3", 1, "")
	test("oneof invalid", "oneof=1 2 3", 4, "must be one of 1, 2, 3")
	test("between valid", "between=1 2", 1, "")
	test("between invalid", "between=1 2", 3, "must be between 1 and 2")
	test("positive valid", "positive", 0, "")
	test("positive invalid", "positive", -1, "must be greater than 0")
	test("negative valid", "negative", -1, "")
	test("negative invalid", "negative", 1, "must be less than 0")
	test("latitude valid", "latitude", -90, "")
	test("latitude invalid", "latitude", -91, "must be between -90 and 90")
	test("longitude valid", "longitude", -180, "")
	test("longitude invalid", "longitude", -181, "must be between -180 and 180")
	test("chain valid", "required,min=1,max=10", 5, "")
	test("chain invalid", "required,min=1,max=10", 0, "must not be zero")
	test("chain invalid", "required,min=1,max=10", -1, "min 1")
	test("chain invalid", "required,min=1,max=10", 11, "max 10")
}

func TestSliceValidator(t *testing.T) {
	test := func(name, expr string, input []string, want string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			validate := validator.SliceExpr[string](expr)
			got := validate.Validate(input)
			is := assert.New(t)
			if want == "" {
				is.Nil(got)
			} else {
				is.Equal(got.Error(), want)
			}
			t.Log(expr, input, got)
		})
	}
	test("required valid", "required", []string{"a"}, "")
	test("required invalid", "required", []string{}, "must not be empty")
	test("optional valid", "optional", []string{}, "")
	test("min valid", "min=1", []string{"a"}, "")
	test("min invalid", "min=1", []string{}, "min 1 item")
	test("max valid", "max=1", []string{"a"}, "")
	test("max invalid", "max=1", []string{"a", "b"}, "max 1 item")
	test("len valid", "len=1", []string{"a"}, "")
	test("len invalid", "len=1", []string{}, "must have 1 item")
	test("chain valid", "required,min=2,max=3", []string{"a", "b"}, "")
	test("chain invalid", "required,min=2,max=3", []string{"a"}, "min 2 items")
	test("chain invalid", "required,min=2,max=3", []string{"a", "b", "c", "d"}, "max 3 items")

	validateEmail := validator.NewStringBuilder().Parse("required,email").Build()
	validateSlice := validator.NewSliceBuilder[string]().Parse("required,min=2").Each(validateEmail).Build()
	assert.Nil(t, validateSlice([]string{"john.doe@mail.com", "jane.doe@mail.com"}))
	assert.Equal(t, validateSlice([]string{"john", "jane"}).Error(), "invalid email")
}
