package okay_test

import (
	"context"
	"fmt"

	"github.com/alextanhongpin/core/types/okay"
)

type contextKey string
type Document struct {
	UserID string
	Public bool
}

func DocumentOwnerOK() okay.OK[Document] {
	fn := func(ctx context.Context, doc Document) okay.Response {
		userID, ok := ctx.Value(contextKey("user_id")).(string)
		if !ok || userID != doc.UserID {
			return okay.Errorf("unauthorized")
		}

		return okay.Allow(true)
	}

	return okay.Func[Document](fn)
}

func DocumentPublicOK() okay.OK[Document] {
	fn := func(ctx context.Context, doc Document) okay.Response {
		if doc.Public {
			return okay.Allow(true)
		}

		return okay.Errorf("private document")
	}

	return okay.Func[Document](fn)
}

func ExampleNew() {
	ok := okay.New(
		DocumentPublicOK(),
		DocumentOwnerOK(),
	)

	publicDoc := Document{
		UserID: "user-xyz",
		Public: true,
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, contextKey("user_id"), "user-xyz")
	res := okay.Check[Document](ctx, publicDoc, ok)

	fmt.Println(res.OK())
	fmt.Println(res.Err())

	res = ok.Allows(context.Background(), publicDoc)
	fmt.Println(res.OK())
	fmt.Println(res.Err())

	privateDoc := Document{
		UserID: "user-xyz",
		Public: false,
	}

	res = ok.Allows(ctx, privateDoc)

	fmt.Println(res.OK())
	fmt.Println(res.Err())
	// Output:
	// true
	// <nil>
	// false
	// unauthorized
	// false
	// private document
}
