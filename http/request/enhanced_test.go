package request_test

import (
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/alextanhongpin/core/http/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestStruct struct {
	Name  string   `json:"name" form:"name"`
	Age   int      `json:"age" form:"age"`
	Email string   `json:"email" form:"email"`
	Tags  []string `json:"tags" form:"tags"`
}

func (ts *TestStruct) Validate() error {
	if ts.Name == "" {
		return errors.New("name is required")
	}
	if ts.Age < 0 || ts.Age > 150 {
		return errors.New("age must be between 0 and 150")
	}
	return nil
}

func TestDecodeJSONWithOptions(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		opts    []request.DecodeOption
		wantErr bool
		errType error
	}{
		{
			name:    "valid JSON",
			body:    `{"name":"John","age":30,"email":"john@example.com"}`,
			opts:    nil,
			wantErr: false,
		},
		{
			name:    "empty body with required option",
			body:    "",
			opts:    []request.DecodeOption{request.WithRequired()},
			wantErr: true,
			errType: request.ErrEmptyBody,
		},
		{
			name:    "body too large",
			body:    strings.Repeat("a", 1000),
			opts:    []request.DecodeOption{request.WithMaxBodySize(500)},
			wantErr: true,
			errType: request.ErrBodyTooLarge,
		},
		{
			name:    "invalid JSON",
			body:    `{"name":"John","age":}`,
			opts:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			var result TestStruct

			err := request.DecodeJSON(req, &result, tt.opts...)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					var bodyErr *request.BodyError
					if assert.ErrorAs(t, err, &bodyErr) {
						assert.ErrorIs(t, bodyErr, tt.errType)
					}
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "John", result.Name)
				assert.Equal(t, 30, result.Age)
			}
		})
	}
}

func TestDecodeForm(t *testing.T) {
	form := url.Values{
		"name":  []string{"John Doe"},
		"age":   []string{"30"},
		"email": []string{"john@example.com"},
		"tags":  []string{"go,web,api"},
	}

	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var result TestStruct
	err := request.DecodeForm(req, &result)

	require.NoError(t, err)
	assert.Equal(t, "John Doe", result.Name)
	assert.Equal(t, 30, result.Age)
	assert.Equal(t, "john@example.com", result.Email)
}

func TestDecodeQuery(t *testing.T) {
	req := httptest.NewRequest("GET", "/?name=Jane&age=25&email=jane@example.com", nil)

	var result TestStruct
	err := request.DecodeQuery(req, &result)

	require.NoError(t, err)
	assert.Equal(t, "Jane", result.Name)
	assert.Equal(t, 25, result.Age)
	assert.Equal(t, "jane@example.com", result.Email)
}

func TestValueValidation(t *testing.T) {
	tests := []struct {
		name    string
		value   request.Value
		test    func(request.Value) error
		wantErr bool
	}{
		{
			name:    "required field with value",
			value:   request.Value("test"),
			test:    func(v request.Value) error { return v.Required("test_field") },
			wantErr: false,
		},
		{
			name:    "required field empty",
			value:   request.Value(""),
			test:    func(v request.Value) error { return v.Required("test_field") },
			wantErr: true,
		},
		{
			name:    "length validation valid",
			value:   request.Value("hello"),
			test:    func(v request.Value) error { return v.Length(3, 10) },
			wantErr: false,
		},
		{
			name:    "length validation too short",
			value:   request.Value("hi"),
			test:    func(v request.Value) error { return v.Length(3, 10) },
			wantErr: true,
		},
		{
			name:    "email validation valid",
			value:   request.Value("test@example.com"),
			test:    func(v request.Value) error { return v.Email() },
			wantErr: false,
		},
		{
			name:    "email validation invalid",
			value:   request.Value("invalid-email"),
			test:    func(v request.Value) error { return v.Email() },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.test(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValueConversions(t *testing.T) {
	t.Run("string operations", func(t *testing.T) {
		v := request.Value("  Hello World  ")
		assert.Equal(t, "Hello World", v.Trim().String())
		assert.Equal(t, "hello world", v.Trim().Lower().String())
		assert.Equal(t, "HELLO WORLD", v.Trim().Upper().String())
		assert.True(t, v.Contains("Hello"))
		assert.True(t, v.Trim().HasPrefix("Hello"))
		assert.True(t, v.Trim().HasSuffix("World"))
	})

	t.Run("numeric conversions", func(t *testing.T) {
		v := request.Value("42")
		assert.Equal(t, 42, v.Int())
		assert.Equal(t, int32(42), v.Int32())
		assert.Equal(t, int64(42), v.Int64())
		assert.Equal(t, 42.0, v.Float64())

		// Test range validation
		val, err := v.IntRange(40, 50)
		assert.NoError(t, err)
		assert.Equal(t, 42, val)

		_, err = v.IntRange(50, 60)
		assert.Error(t, err)
	})

	t.Run("boolean conversion", func(t *testing.T) {
		assert.True(t, request.Value("true").Bool())
		assert.True(t, request.Value("1").Bool())
		assert.False(t, request.Value("false").Bool())
		assert.False(t, request.Value("0").Bool())
		assert.True(t, request.Value("").BoolOr(true))
	})

	t.Run("time parsing", func(t *testing.T) {
		timeStr := "2023-01-01T10:00:00Z"
		v := request.Value(timeStr)

		parsedTime, err := v.RFC3339()
		assert.NoError(t, err)
		assert.Equal(t, 2023, parsedTime.Year())

		dateStr := "2023-01-01"
		dateVal := request.Value(dateStr)
		parsedDate, err := dateVal.Date()
		assert.NoError(t, err)
		assert.Equal(t, 2023, parsedDate.Year())
		assert.Equal(t, time.January, parsedDate.Month())
		assert.Equal(t, 1, parsedDate.Day())
	})

	t.Run("URL parsing", func(t *testing.T) {
		v := request.Value("https://example.com/path?query=value")
		url, err := v.URL()
		assert.NoError(t, err)
		assert.Equal(t, "https", url.Scheme)
		assert.Equal(t, "example.com", url.Host)
		assert.Equal(t, "/path", url.Path)
	})

	t.Run("CSV parsing", func(t *testing.T) {
		v := request.Value("apple, banana , cherry")
		parts := v.CSV()
		expected := []string{"apple", "banana", "cherry"}
		assert.Equal(t, expected, parts)

		empty := request.Value("")
		assert.Empty(t, empty.CSV())
	})

	t.Run("slice operations", func(t *testing.T) {
		v := request.Value("option1")
		options := []string{"option1", "option2", "option3"}
		assert.True(t, v.InSlice(options))
		assert.False(t, request.Value("option4").InSlice(options))
	})

	t.Run("pattern matching", func(t *testing.T) {
		v := request.Value("hello world")
		assert.True(t, v.Match("*"))
		assert.True(t, v.Match("hello*"))
		assert.True(t, v.Match("*world"))
		assert.True(t, v.Match("*lo wo*"))
		assert.False(t, v.Match("goodbye*"))
	})
}

func TestValueHelpers(t *testing.T) {
	req := httptest.NewRequest("GET", "/test?name=John&age=30&tags=go,web", nil)
	req.Header.Set("Authorization", "Bearer token123")

	t.Run("query values", func(t *testing.T) {
		name := request.QueryValue(req, "name")
		assert.Equal(t, "John", name.String())

		age := request.QueryValue(req, "age")
		assert.Equal(t, 30, age.Int())

		missing := request.QueryValue(req, "missing")
		assert.True(t, missing.IsEmpty())
		assert.Equal(t, "default", missing.StringOr("default"))
	})

	t.Run("header values", func(t *testing.T) {
		auth := request.HeaderValue(req, "Authorization")
		assert.True(t, auth.HasPrefix("Bearer"))

		token := request.Value(strings.TrimPrefix(auth.String(), "Bearer "))
		assert.Equal(t, "token123", token.String())
	})

	t.Run("multiple query values", func(t *testing.T) {
		// Add multiple values for testing
		req.URL.RawQuery = "tags=go&tags=web&tags=api"

		tags := request.QueryValues(req, "tags")
		assert.Len(t, tags, 3)
		assert.Equal(t, "go", tags[0].String())
		assert.Equal(t, "web", tags[1].String())
		assert.Equal(t, "api", tags[2].String())
	})
}

func TestBase64Operations(t *testing.T) {
	original := "Hello, World!"
	v := request.Value(original)

	encoded := v.ToBase64()
	assert.NotEmpty(t, encoded.String())

	decoded := encoded.FromBase64()
	assert.Equal(t, original, decoded.String())
}
