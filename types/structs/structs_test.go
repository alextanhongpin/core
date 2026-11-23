package structs_test

import (
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/types/structs"
	"github.com/go-openapi/testify/assert"
)

type Profile struct {
	Age   int
	Token string
}

type User struct {
	Name    string
	Token   string
	Profile *Profile
	Friends []User
	secret  string
}

func TestZeroField(t *testing.T) {
	is := assert.New(t)

	users := []User{{
		Name:    "alice",
		Token:   "t1",
		Profile: &Profile{Age: 10, Token: "t2"},
		Friends: []User{{Name: "bob", Token: "t3"}},
	}}
	is.NoError(structs.ZeroField(users, "Token"))
	is.Empty(users[0].Token)
	is.Empty(users[0].Profile.Token)
	is.Empty(users[0].Friends[0].Token)
	is.Equal("alice", users[0].Name)
	is.Equal(10, users[0].Profile.Age)

	is.Error(structs.ZeroField(&User{}))
	is.Error(structs.ZeroField(&User{secret: "x"}, "secret"))
	is.Error(structs.ZeroField(User{Name: "a"}, "Name"), "not addressable")
}

type Node struct {
	Value int
	Next  *Node
}

func TestZeroFieldCycle(t *testing.T) {
	a := &Node{Value: 1}
	b := &Node{Value: 2, Next: a}
	a.Next = b
	assert.NoError(t, structs.ZeroField(a, "Value"))
	assert.Zero(t, a.Value)
	assert.Zero(t, b.Value)
}

func TestZeroPath(t *testing.T) {
	is := assert.New(t)

	u := &User{Token: "t1", Profile: &Profile{Age: 10, Token: "t2"}}
	is.NoError(structs.ZeroPath(u, "Profile.Token"))
	is.Equal("t1", u.Token)
	is.Empty(u.Profile.Token)
	is.Equal(10, u.Profile.Age)

	is.NoError(structs.ZeroPath(&User{}, "Profile.Age"), "nil pointer is skipped")
	is.ErrorContains(structs.ZeroPath(u, "Missing"), "does not exist")
	is.ErrorContains(structs.ZeroPath(u, "Name.Length"), "not a struct")
	is.Error(structs.ZeroPath(u, ""))
	is.Error(structs.ZeroPath(u))
}

type svc struct{}

func (svc) B()  {}
func (*svc) A() {}
func (*svc) c() {}

func TestGetMethodNames(t *testing.T) {
	is := assert.New(t)

	for _, v := range []any{svc{}, &svc{}} {
		names, err := structs.GetMethodNames(v)
		is.NoError(err)
		is.Equal([]string{"A", "B"}, names)
	}
	_, err := structs.GetMethodNames(nil)
	is.Error(err)
}

func TestIsNilOrZero(t *testing.T) {
	var p *int
	var m map[string]int
	tests := []struct {
		in   any
		want bool
	}{
		{nil, true},
		{p, true},
		{m, true},
		{0, true},
		{"", true},
		{Profile{}, true},
		{1, false},
		{[]int{}, false},
		{Profile{Age: 1}, false},
	}
	for _, tt := range tests {
		if got := structs.IsNilOrZero(tt.in); got != tt.want {
			t.Errorf("IsNilOrZero(%#v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func ExampleZeroField() {
	type Account struct {
		Email    string
		Password string
	}
	accounts := []Account{{"a@example.com", "hunter2"}, {"b@example.com", "secret"}}
	if err := structs.ZeroField(accounts, "Password"); err != nil {
		panic(err)
	}
	fmt.Println(accounts)
	// Output: [{a@example.com } {b@example.com }]
}
