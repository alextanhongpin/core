package testdump_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/test/testdump"
	"github.com/google/uuid"
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

func TestJSONMap(t *testing.T) {
	fileName := fmt.Sprintf("testdata/%s.json", t.Name())
	data := map[string]any{
		"email":     "John Appleseed",
		"password":  "$up3rS3cr3t", // To be masked.
		"createdAt": time.Now(),    // Dynamic.
	}

	type T = map[string]any

	opt := new(testdump.JSONOption)
	opt.IgnoreFields = []string{"createdAt"}
	opt.MaskFields = []string{"password"}

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

	opt := new(testdump.JSONOption)
	opt.IgnoreFields = []string{"createdAt"}
	opt.MaskFields = []string{"password"}
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

		opt := new(testdump.JSONOption)
		opt.IgnoreFields = []string{"createdAt"}

		assert := assert.New(t)
		err := testdump.JSON(testdump.NewFile(fileName), u, opt)
		assert.NotNil(err)

		var diffErr *internal.DiffError
		assert.True(errors.As(err, &diffErr))

		testdump.Text(testdump.NewFile(fmt.Sprintf("testdata/%s.txt", t.Name())), diffErr.Text())
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

		opt := new(testdump.JSONOption)
		opt.IgnoreFields = []string{"createdAt"}
		opt.MaskFields = []string{"password"}
		assert := assert.New(t)
		err := testdump.JSON(testdump.NewFile(fileName), u, opt)
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
		err := testdump.JSON(testdump.NewFile(fileName), u, opt)
		assert.NotNil(err)

		var diffErr *internal.DiffError
		assert.True(errors.As(err, &diffErr))

		testdump.Text(testdump.NewFile(fmt.Sprintf("testdata/%s.txt", t.Name())), diffErr.Text())
	})
}

func TestJSONIgnoreTag(t *testing.T) {
	type User struct {
		Name      string    `json:"name"`
		Email     string    `json:"email"`
		CreatedAt time.Time `json:"createdAt" cmp:",ignore"`
	}

	fileName := fmt.Sprintf("testdata/%s.json", t.Name())
	data := User{
		Name:      "John Appleseed",
		Email:     "john.appleseed@mail.com",
		CreatedAt: time.Now(),
	}

	if err := testdump.JSON(testdump.NewFile(fileName), data, nil); err != nil {
		t.Fatal(err)
	}
}

func TestJSONMaskField(t *testing.T) {
	type LoginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password" mask:"true"`
	}

	fileName := fmt.Sprintf("testdata/%s.json", t.Name())
	data := LoginRequest{
		Email:    "john.appleseed@mail.com",
		Password: "super secret",
	}

	if err := testdump.JSON(testdump.NewFile(fileName), data, nil); err != nil {
		t.Fatal(err)
	}
}

func TestJSONMaskNestedField(t *testing.T) {
	type Credential struct {
		Provider string `json:"provider"`
		Token    string `json:"token" mask:"true"` // This is masked.
	}

	type LoginRequest struct {
		Email       string       `json:"email"`
		Token       string       `json:"token"` // This is not masked.
		Password    string       `json:"password" mask:"true"`
		Credentials []Credential `json:"credentials"`
	}

	fileName := fmt.Sprintf("testdata/%s.json", t.Name())
	data := LoginRequest{
		Email:    "john.appleseed@mail.com",
		Token:    "1234",
		Password: "super secret",
		Credentials: []Credential{
			{
				Provider: "google",
				Token:    "google-token",
			},
			{
				Provider: "facebook",
				Token:    "facebook-token",
			},
		},
	}

	if err := testdump.JSON(testdump.NewFile(fileName), data, nil); err != nil {
		t.Fatal(err)
	}
}

func TestJSONTxTar(t *testing.T) {
	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	fileName := fmt.Sprintf("testdata/%s", t.Name())
	data := User{
		Name:  "John Appleseed",
		Email: "john.appleseed@mail.com",
	}

	if err := testdump.JSON(testdump.NewTxTar(fileName, "user.json"), data, nil); err != nil {
		t.Fatal(err)
	}
}

func TestJSONCue(t *testing.T) {
	type User struct {
		ID       uuid.UUID `json:"id"`
		Name     string    `json:"name"`
		Age      int       `json:"age"`
		Birthday time.Time `json:"birthday"`
	}

	fileName := fmt.Sprintf("testdata/%s.json", t.Name())
	data := User{
		ID:       uuid.New(),
		Name:     "John Appleseed",
		Age:      13,
		Birthday: time.Now(),
	}

	if err := testdump.JSON(testdump.NewFile(fileName), data, &testdump.JSONOption{
		IgnoreFields: []string{"id", "birthday"},
		CUEOption: &testdump.CUEOption{
			Schema: `import (
	"time",
	"uuid"
)

close({
	id: uuid.Valid()
	name: string
	age: >=13
	birthday: string & time.Format(time.RFC3339)
})`,
		},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestJSONCuePath(t *testing.T) {
	type User struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	fileName := fmt.Sprintf("testdata/%s.json", t.Name())
	data := User{
		Name: "John Appleseed",
		Age:  13,
	}

	if err := testdump.JSON(testdump.NewFile(fileName), data, &testdump.JSONOption{
		CUEOption: &testdump.CUEOption{
			SchemaPath: "testdata/user.cue",
		},
	}); err != nil {
		t.Fatal(err)
	}
}
