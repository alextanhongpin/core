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

	test("optional", "optional", "", "")
	test("is valid", "is=green", "green", "")
	test("is invalid", "is=green", "blue", `must be "green"`)
	test("min valid", "min=1", "a", "")
	test("min invalid", "min=1", "", "must not be empty")
	test("max valid", "max=1", "a", "")
	test("max invalid", "max=1", "ab", "max length is 1")
	test("len valid", "len=3", "abc", "")
	test("len invalid", "len=3", "abcd", "must have exact length of 3")
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
	test("chain valid", "email,starts_with=john,ends_with=@mail.com", "john.appleseed@mail.com", "")
	test("chain invalid", "email,starts_with=john,ends_with=@mail.com", "", "must not be empty")
	test("chain invalid", "email,starts_with=john,ends_with=@mail.com", "john doe", "invalid email")
	test("chain invalid", "email,starts_with=john,ends_with=@mail.com", "jane.doe@mail.com", `must start with "john"`)
	test("chain invalid", "email,starts_with=john,ends_with=@mail.com", "john.doe@yourmail.com", `must end with "@mail.com"`)
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
		"repeat": func(params string) func(string) error {
			ch, ns, _ := strings.Cut(params, " ")
			n := toInt(ns)

			return func(v string) error {
				if strings.Repeat(ch, n) != v {
					return errors.New("does not repeat")
				}

				return nil
			}
		},
		"min": func(params string) func(string) error {
			n := toInt(params)

			return func(v string) error {
				if len(v) < n {
					return errors.New("too short")
				}

				return nil
			}
		},
	}

	val := validator.StringExpr("repeat=# 10", fm)

	is := assert.New(t)
	is.Nil(val.Validate("##########"))
	is.Equal("does not repeat", val.Validate("1").Error())

	val = validator.StringExpr("min=3", fm)
	is.Nil(val.Validate("###"))
	is.Equal("too short", val.Validate("1").Error())
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
	test("optional valid", "optional", 0, "")
	test("min valid", "min=1", 1, "")
	test("min invalid", "min=1", 0, "must not be zero")
	test("max valid", "max=1", 1, "")
	test("max invalid", "max=1", 2, "max 1")
	test("is valid", "is=1", 1, "")
	test("is invalid", "is=1", 2, "must be 1")
	test("oneof valid", "oneof=1 2 3", 1, "")
	test("oneof invalid", "oneof=1 2 3", 4, "must be one of 1, 2, 3")
	test("between valid", "between=1 2", 1, "")
	test("between invalid", "between=1 2", 3, "must be between 1 and 2")
	test("positive valid", "positive", 0, "must not be zero")
	test("positive invalid", "positive", -1, "must be greater than 0")
	test("negative valid", "negative", -1, "")
	test("negative invalid", "negative", 1, "must be less than 0")
	test("latitude valid", "latitude", -90, "")
	test("latitude invalid", "latitude", -91, "must be between -90 and 90")
	test("longitude valid", "longitude", -180, "")
	test("longitude invalid", "longitude", -181, "must be between -180 and 180")
	test("chain valid", "min=1,max=10", 5, "")
	test("chain invalid", "min=1,max=10", 0, "must not be zero")
	test("chain invalid", "min=1,max=10", -1, "min 1")
	test("chain invalid", "min=1,max=10", 11, "max 10")
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
	test("optional valid", "optional", []string{}, "")
	test("min valid", "min=1", []string{"a"}, "")
	test("min invalid", "min=1", []string{}, "must not be empty")
	test("max valid", "max=1", []string{"a"}, "")
	test("max invalid", "max=1", []string{"a", "b"}, "max items is 1")
	test("len valid", "len=1", []string{"a"}, "")
	test("len invalid", "len=1", []string{}, "must not be empty")
	test("chain valid", "min=2,max=3", []string{"a", "b"}, "")
	test("chain invalid", "min=2,max=3", []string{"a"}, "min items is 2")
	test("chain invalid", "min=2,max=3", []string{"a", "b", "c", "d"}, "max items is 3")

	validateEmail := validator.String().Parse("email")
	validateSlice := validator.Slice[string]().Parse("optional,min=2").Each(validateEmail)
	assert.Nil(t, validateSlice.Validate([]string{}))
	assert.ErrorContains(t, validateSlice.Validate([]string{"john.doe@mail.com"}), "min items is 2")
	assert.Nil(t, validateSlice.Validate([]string{"john.doe@mail.com", "jane.doe@mail.com"}))
	assert.Equal(t, validateSlice.Validate([]string{"john", "jane"}).Error(), "invalid email")
}
