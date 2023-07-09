package testdump_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/test/testdump"
	"github.com/alextanhongpin/core/types/maputil"
	"github.com/google/go-cmp/cmp"
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

	if err := testdump.YAML(fileName, data, nil); err != nil {
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

	opt := testdump.YAMLOption{
		Body: []cmp.Option{
			// Ignore CreatedAt field for comparison.
			internal.IgnoreMapEntries("CreatedAt"),
		},
		Hooks: []testdump.Hook[any]{
			// Mask the password value.
			testdump.MarshalHook(func(a any) (any, error) {
				c := a.(Credentials)
				c.Password = maputil.MaskValue
				return c, nil
			}),

			// Validate that the time is not zero.
			testdump.CompareHook(func(snap, recv any) error {
				x := snap.(map[string]any)
				y := snap.(map[string]any)
				if err := internal.IsNonZeroTime(x, "CreatedAt"); err != nil {
					return err
				}
				if err := internal.IsNonZeroTime(y, "CreatedAt"); err != nil {
					return err
				}

				return nil
			}),
		},
	}

	if err := testdump.YAML(fileName, data, &opt); err != nil {
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

	opt := testdump.YAMLOption{
		Body: []cmp.Option{
			// Ignore CreatedAt field for comparison.
			internal.IgnoreMapEntries("createdAt"),
		},
		Hooks: []testdump.Hook[any]{
			// Mask the password value.
			testdump.MarshalHook(func(a any) (any, error) {
				m := a.(map[string]any)
				m["password"] = maputil.MaskValue
				return m, nil
			}),

			// Validate that the time is not zero.
			testdump.CompareHook(func(snap, recv any) error {
				x := snap.(map[string]any)
				y := snap.(map[string]any)
				if err := internal.IsNonZeroTime(x, "createdAt"); err != nil {
					return err
				}
				if err := internal.IsNonZeroTime(y, "createdAt"); err != nil {
					return err
				}

				return nil
			}),
		},
	}

	if err := testdump.YAML(fileName, data, &opt); err != nil {
		t.Fatal(err)
	}
}
