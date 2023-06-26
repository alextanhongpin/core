package maputil_test

import (
	"testing"

	"github.com/alextanhongpin/core/test/testutil"
	"github.com/alextanhongpin/core/types/maputil"
)

func TestMask(t *testing.T) {
	type Data struct {
		Token string `json:"token"`
	}
	type credentials struct {
		Password string `json:"password"`
		Email    string `json:"email"`
		Data     Data   `json:"data"`
		Tokens   []Data `json:"tokens"`
	}

	creds := credentials{
		Password: "123456",
		Email:    "john.doe@mail.com",
		Data:     Data{Token: "jwt-123456"},
		Tokens:   []Data{{Token: "abc-123"}, {Token: "xyz-987"}},
	}

	credsMap, err := maputil.StructToMap(creds)
	if err != nil {
		t.Fatal(err)
	}

	credsMask := maputil.MaskFunc(credsMap,
		maputil.MaskFields("password", "data.token", "tokens[_].token"),
	)
	testutil.DumpJSON(t, credsMask)
}
