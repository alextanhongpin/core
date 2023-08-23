package testdump_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/test/testdump"
	"github.com/alextanhongpin/core/types/maputil"
	"github.com/stretchr/testify/assert"
)

func TestJSON(t *testing.T) {
	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	fileName := fmt.Sprintf("testdata/%s.json", t.Name())
	data := User{
		Name:  "John Appleseed",
		Email: "john.appleseed@mail.com",
	}

	if err := testdump.JSON(testdump.NewFile(fileName), data, nil); err != nil {
		t.Fatal(err)
	}
}

func TestJSONHook(t *testing.T) {
	type Credentials struct {
		Email     string    `json:"email"`
		Password  string    `json:"password"`
		CreatedAt time.Time `json:"createdAt"`
	}

	fileName := fmt.Sprintf("testdata/%s.json", t.Name())
	data := Credentials{
		Email:     "John Appleseed",
		Password:  "$up3rS3cr3t", // To be masked.
		CreatedAt: time.Now(),    // Dynamic.
	}

	// Alias to shorten the types.
	type T = Credentials

	opt := new(testdump.JSONOption[T])
	opt.Body = append(opt.Body, internal.IgnoreMapEntries("createdAt"))
	opt.Hooks = append(opt.Hooks,
		// Mask the password value.
		testdump.MarshalHook(func(t T) (T, error) {
			t.Password = maputil.MaskValue
			return t, nil
		}),
		// Validate that the time is not zero.
		testdump.CompareHook(func(snap, recv T) error {
			if snap.CreatedAt.IsZero() || recv.CreatedAt.IsZero() {
				return errors.New("zero time")
			}

			return nil
		}),
	)

	if err := testdump.JSON(testdump.NewFile(fileName), data, opt); err != nil {
		t.Fatal(err)
	}
}

func TestJSONMap(t *testing.T) {
	fileName := fmt.Sprintf("testdata/%s.json", t.Name())
	data := map[string]any{
		"email":     "John Appleseed",
		"password":  "$up3rS3cr3t", // To be masked.
		"createdAt": time.Now(),    // Dynamic.
	}

	type T = map[string]any

	opt := new(testdump.JSONOption[T])
	opt.Body = append(opt.Body, internal.IgnoreMapEntries("createdAt"))
	opt.Hooks = append(opt.Hooks,
		testdump.MarshalHook(func(m T) (T, error) {
			// WARN: this will only add the field even if it is deleted.
			m["password"] = maputil.MaskValue
			return m, nil
		}),

		// Validate that the time is not zero.
		testdump.CompareHook(func(snap, recv T) error {
			return internal.AnyError(
				internal.IsNonZeroTime(snap, "createdAt"),
				internal.IsNonZeroTime(recv, "createdAt"),
			)
		}),
	)

	if err := testdump.JSON(testdump.NewFile(fileName), data, opt); err != nil {
		t.Fatal(err)
	}
}

func TestJSONDiff(t *testing.T) {
	type User struct {
		ID        int64     `json:"id"`
		Email     string    `json:"email"`
		Password  string    `json:"password"`
		CreatedAt time.Time `json:"createdAt"`
	}

	fileName := fmt.Sprintf("testdata/%s.json", t.Name())
	u := User{
		ID:        42,
		Email:     "John Appleseed",
		Password:  "$up3rS3cr3t", // To be masked.
		CreatedAt: time.Now(),    // Dynamic.
	}

	// Alias to shorten the types.
	type T = User

	opt := new(testdump.JSONOption[T])
	opt.Body = append(opt.Body, internal.IgnoreMapEntries("createdAt"))
	opt.Hooks = append(opt.Hooks,
		testdump.MarshalHook(func(t T) (T, error) {
			t.Password = maputil.MaskValue

			return t, nil
		}),
	)
	if err := testdump.JSON(testdump.NewFile(fileName), u, opt); err != nil {
		t.Fatal(err)
	}

	t.Run("add new field", func(t *testing.T) {
		type NewUser struct {
			User
			Hobbies []string `json:"hobbies"`
		}

		type T = NewUser

		u := NewUser{
			User:    u,
			Hobbies: []string{"coding"},
		}

		opt := new(testdump.JSONOption[T])
		opt.Body = append(opt.Body, internal.IgnoreMapEntries("createdAt"))
		opt.Hooks = append(opt.Hooks,
			testdump.MarshalHook(func(t T) (T, error) {
				t.Password = maputil.MaskValue

				return t, nil
			}),
		)

		assert := assert.New(t)
		err := testdump.JSON(testdump.NewFile(fileName), u, opt)
		assert.NotNil(err)

		var diffErr *internal.DiffError
		assert.True(errors.As(err, &diffErr))

		diffText := diffErr.Text()
		plus, minus := parseDiff(diffText)
		assert.Len(plus, 1)
		assert.Len(minus, 0)
		assert.Equal(`"hobbies":  []any{string("coding")},`, plus[0])
	})

	t.Run("remove existing field", func(t *testing.T) {
		type PartialUser struct {
			Email     string    `json:"email"`
			Password  string    `json:"password"`
			CreatedAt time.Time `json:"createdAt"`
		}

		type T = PartialUser

		u := T{
			Email:     u.Email,
			Password:  u.Password,
			CreatedAt: u.CreatedAt,
		}

		opt := new(testdump.JSONOption[T])
		opt.Body = append(opt.Body, internal.IgnoreMapEntries("createdAt"))
		opt.Hooks = append(opt.Hooks,
			testdump.MarshalHook(func(t T) (T, error) {
				t.Password = maputil.MaskValue

				return t, nil
			}),
		)

		assert := assert.New(t)
		err := testdump.JSON(testdump.NewFile(fileName), u, opt)
		assert.NotNil(err)

		var diffErr *internal.DiffError
		assert.True(errors.As(err, &diffErr))

		diffText := diffErr.Text()
		plus, minus := parseDiff(diffText)
		assert.Len(plus, 0)
		assert.Len(minus, 1)
		assert.Equal(`"id":       float64(42),`, minus[0])
	})

	t.Run("update existing field", func(t *testing.T) {
		u := User{
			ID:        42,
			Email:     "John Doe",
			Password:  "$up3rS3cr3t", // To be masked.
			CreatedAt: time.Now(),    // Dynamic.
		}

		assert := assert.New(t)
		err := testdump.JSON(testdump.NewFile(fileName), u, opt)
		assert.NotNil(err)

		var diffErr *internal.DiffError
		assert.True(errors.As(err, &diffErr))

		diffText := diffErr.Text()
		plus, minus := parseDiff(diffText)
		assert.Len(plus, 1)
		assert.Len(minus, 1)
		assert.Equal(`"email":    string("John Doe"),`, plus[0])
		assert.Equal(`"email":    string("John Appleseed"),`, minus[0])
	})
}

func parseDiff(diff string) ([]string, []string) {
	var plus, minus []string

	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		switch line[0] {
		case '+':
			plus = append(plus, strings.TrimSpace(line[1:]))
		case '-':
			minus = append(minus, strings.TrimSpace(line[1:]))
		}

	}

	return plus, minus
}
