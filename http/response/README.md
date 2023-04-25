# HTTP Response

HTTP Response package provides a convenient way of marhshalling struct to JSON, and vice-versa.


## DecodeJSON

The `DecodeJSON` method decodes the json payload into struct, which is inferred through generic type:

```go
type LoginRequest struct {
	Email string `json:"email" validate:"required,email"`
}

func PostLoginHandler(w http.ResponseWriter, r *http.Request) {
	req, err := encoding.DecodeJSON[LoginRequest](w, r)
	// ...
}
```

Validation is done using the golang validator library. Note that the validation library is not customizable, since complex validation should be done in the domain layer and not the API layer anyway.


Additionally, this method also copies the response body to another buffer, so that we can read the response from the body multiple times later. This is particularly useful in the logging middleware, where we want to log the sample response body if an error occurs while decoding. The error can happen for example, when the client sends some payload that is not defined by the server.


## EncodeJSON

The `EncodeJSON` method encodes the struct to json representation:


```go
type LoginResponse struct {
	AccessToken string `json:"accessToken"`
}

func PostLoginHandler(w http.ResponseWriter, r *http.Request) {
	encoding.EncodeJSON(w, http.StatusOK, types.Result[LoginResponse]{
		Data: &LoginResponse{
			AccessToken: "token",
		},
	})
}
```


Similarly, the `EncodeJSONError` encodes the error into a JSON representation.

If an app error of type `*errors.Error` is passed in, this method will automatically infer the _status code_ as well as the message to be shown to the end user.

Any other error that is non-app error will be treated as unknown errors. See more on error handling.
