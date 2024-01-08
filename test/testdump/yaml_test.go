package testdump_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/test/testdump"
	"github.com/alextanhongpin/core/types/maputil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func TestYAML(t *testing.T) {
	type User struct {
		Name         string `yaml:"username"`
		EmailAddress string
	}

	fileName := fmt.Sprintf("testdata/%s.yaml", t.Name())
	data := User{
		Name:         "John Appleseed",
		EmailAddress: "john.appleseed@mail.com",
	}

	if err := testdump.YAML(testdump.NewFile(fileName), data, nil); err != nil {
		t.Fatal(err)
	}
}

func TestYAMLHook(t *testing.T) {
	type Credentials struct {
		Email     string
		Password  string
		CreatedAt time.Time
	}

	fileName := fmt.Sprintf("testdata/%s.yaml", t.Name())
	data := Credentials{
		Email:     "John Appleseed",
		Password:  "$up3rS3cr3t", // To be masked.
		CreatedAt: time.Now(),    // Dynamic.
	}

	// Alias to shorten the types.
	type T = Credentials

	opt := testdump.YAMLOption{
		Body: []cmp.Option{
			// Ignore CreatedAt field for comparison.
			internal.IgnoreMapEntries("CreatedAt"),
		},
	}

	hooks := []testdump.Hook[T]{
		// Mask the password value.
		testdump.MarshalHook(func(t T) (T, error) {
			t.Password = maputil.MaskValue
			return t, nil
		}),

		// Validate that the time is not zero.
		testdump.CompareHook(func(snap, recv T) error {
			// You can access the concrete type.
			if snap.CreatedAt.IsZero() || recv.CreatedAt.IsZero() {
				return errors.New("zero time")
			}
			return nil
		}),
	}

	if err := testdump.YAML(testdump.NewFile(fileName), data, &opt, hooks...); err != nil {
		t.Fatal(err)
	}
}

func TestYAMLMap(t *testing.T) {
	fileName := fmt.Sprintf("testdata/%s.yaml", t.Name())
	data := map[string]any{
		"email":     "John Appleseed",
		"password":  "$up3rS3cr3t", // To be masked.
		"createdAt": time.Now(),    // Dynamic.
	}

	type T = map[string]any

	opt := testdump.YAMLOption{
		Body: []cmp.Option{
			// Ignore CreatedAt field for comparison.
			internal.IgnoreMapEntries("createdAt"),
		},
	}
	hooks := []testdump.Hook[T]{
		// Mask the password value.
		testdump.MarshalHook(func(t T) (T, error) {
			// WARN: This will set the field even if it doesn't exists.
			t["password"] = maputil.MaskValue
			return t, nil
		}),

		// Validate that the time is not zero.
		testdump.CompareHook(func(snap, recv T) error {
			x := snap
			y := recv
			if err := internal.IsNonZeroTime(x, "createdAt"); err != nil {
				return err
			}
			if err := internal.IsNonZeroTime(y, "createdAt"); err != nil {
				return err
			}

			return nil
		}),
	}

	if err := testdump.YAML(testdump.NewFile(fileName), data, &opt, hooks...); err != nil {
		t.Fatal(err)
	}
}

func TestYAMLDiff(t *testing.T) {
	type User struct {
		ID        int64
		Email     string
		Password  string
		CreatedAt time.Time
	}

	fileName := fmt.Sprintf("testdata/%s.yaml", t.Name())
	u := User{
		ID:        42,
		Email:     "John Appleseed",
		Password:  "$up3rS3cr3t", // To be masked.
		CreatedAt: time.Now(),    // Dynamic.
	}

	// Alias to shorten the types.
	type T = User

	opt := new(testdump.YAMLOption)
	opt.Body = append(opt.Body, internal.IgnoreMapEntries("CreatedAt"))
	hooks := []testdump.Hook[T]{
		testdump.MarshalHook(func(t T) (T, error) {
			t.Password = maputil.MaskValue

			return t, nil
		}),
	}
	if err := testdump.YAML(testdump.NewFile(fileName), u, opt, hooks...); err != nil {
		t.Fatal(err)
	}

	t.Run("add new field", func(t *testing.T) {
		type NewUser struct {
			User
			Hobbies []string
		}

		type T = NewUser

		u := NewUser{
			User:    u,
			Hobbies: []string{"coding"},
		}

		opt := new(testdump.YAMLOption)
		opt.Body = append(opt.Body, internal.IgnoreMapEntries("CreatedAt"))
		hooks := []testdump.Hook[T]{
			testdump.MarshalHook(func(t T) (T, error) {
				t.Password = maputil.MaskValue

				return t, nil
			}),
		}

		assert := assert.New(t)
		err := testdump.YAML(testdump.NewFile(fileName), u, opt, hooks...)
		assert.NotNil(err)

		var diffErr *internal.DiffError
		assert.True(errors.As(err, &diffErr))
		testdump.Text(testdump.NewFile(fmt.Sprintf("testdata/%s.txt", t.Name())), diffErr.Text())
	})

	t.Run("remove existing field", func(t *testing.T) {
		type PartialUser struct {
			Email     string
			Password  string
			CreatedAt time.Time
		}

		type T = PartialUser

		u := T{
			Email:     u.Email,
			Password:  u.Password,
			CreatedAt: u.CreatedAt,
		}

		opt := new(testdump.YAMLOption)
		opt.Body = append(opt.Body, internal.IgnoreMapEntries("CreatedAt"))
		hooks := []testdump.Hook[T]{
			testdump.MarshalHook(func(t T) (T, error) {
				t.Password = maputil.MaskValue

				return t, nil
			}),
		}

		assert := assert.New(t)
		err := testdump.YAML(testdump.NewFile(fileName), u, opt, hooks...)
		assert.NotNil(err)

		var diffErr *internal.DiffError
		assert.True(errors.As(err, &diffErr))

		testdump.Text(testdump.NewFile(fmt.Sprintf("testdata/%s.txt", t.Name())), diffErr.Text())
	})

	t.Run("update existing field", func(t *testing.T) {
		u := User{
			ID:        42,
			Email:     "John Doe",
			Password:  "$up3rS3cr3t", // To be masked.
			CreatedAt: time.Now(),    // Dynamic.
		}

		assert := assert.New(t)
		err := testdump.YAML(testdump.NewFile(fileName), u, opt, hooks...)
		assert.NotNil(err)

		var diffErr *internal.DiffError
		assert.True(errors.As(err, &diffErr))

		testdump.Text(testdump.NewFile(fmt.Sprintf("testdata/%s.txt", t.Name())), diffErr.Text())
	})
}

func TestYAMLMaskField(t *testing.T) {
	type LoginRequest struct {
		Email    string
		Password string `cmp:",mask"`
	}

	fileName := fmt.Sprintf("testdata/%s.yaml", t.Name())
	data := LoginRequest{
		Email:    "john.appleseed@mail.com",
		Password: "super secret",
	}

	if err := testdump.YAML(testdump.NewFile(fileName), data, nil); err != nil {
		t.Fatal(err)
	}
}

func TestYAMLIgnoreTag(t *testing.T) {
	type User struct {
		Name      string
		Email     string
		CreatedAt time.Time `cmp:",ignore"`
	}

	fileName := fmt.Sprintf("testdata/%s.yaml", t.Name())
	data := User{
		Name:      "John Appleseed",
		Email:     "john.appleseed@mail.com",
		CreatedAt: time.Now(),
	}

	if err := testdump.YAML(testdump.NewFile(fileName), data, nil); err != nil {
		t.Fatal(err)
	}
}
