package stringcase

import (
	"testing"

	"github.com/go-openapi/testify/assert"
)

func TestStringCaseConversions(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		kebab  string
		snake  string
		camel  string
		pascal string
		title  string
	}{
		{"simple words", "hello world", "hello-world", "hello_world", "helloWorld", "HelloWorld", "Hello World"},
		{"single word", "hello", "hello", "hello", "hello", "Hello", "Hello"},
		{"PascalCase", "HelloWorld", "hello-world", "hello_world", "helloWorld", "HelloWorld", "Hello World"},
		{"camelCase", "helloWorld", "hello-world", "hello_world", "helloWorld", "HelloWorld", "Hello World"},
		{"snake_case", "hello_world", "hello-world", "hello_world", "helloWorld", "HelloWorld", "Hello World"},
		{"kebab-case", "hello-world", "hello-world", "hello_world", "helloWorld", "HelloWorld", "Hello World"},
		{"Title Case", "Hello World", "hello-world", "hello_world", "helloWorld", "HelloWorld", "Hello World"},
		{"with initialism", "userID", "user-id", "user_id", "userID", "UserID", "User ID"},
		{"with initialism 2", "apiResponseID", "api-response-id", "api_response_id", "apiResponseID", "APIResponseID", "API Response ID"},
		// "HTTPS" and "HTTPs" share a lowercase form; the first listed wins.
		{"with initialism 3", "HTTPsServerID", "https-server-id", "https_server_id", "httpsServerID", "HTTPSServerID", "HTTPS Server ID"},
		// Regression: an initialism directly followed by another used to stay joined.
		{"adjacent initialisms", "XMLHTTPRequest", "xml-http-request", "xml_http_request", "xmlHTTPRequest", "XMLHTTPRequest", "XML HTTP Request"},
		// Edge-case.
		{"with initialism 4", "HTTPServerID", "http-server-id", "http_server_id", "httpServerID", "HTTPServerID", "HTTP Server ID"},
		{"with numbers", "user1ID", "user1-id", "user1_id", "user1ID", "User1ID", "User1 ID"},
		{"with underscores and dashes", "user_id-test", "user-id-test", "user_id_test", "userIDTest", "UserIDTest", "User ID Test"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_ToKebab", func(t *testing.T) {
			if got := ToKebab(tt.input); got != tt.kebab {
				t.Errorf("ToKebab(%q) = %q, want %q", tt.input, got, tt.kebab)
			}
		})
		t.Run(tt.name+"_ToSnake", func(t *testing.T) {
			if got := ToSnake(tt.input); got != tt.snake {
				t.Errorf("ToSnake(%q) = %q, want %q", tt.input, got, tt.snake)
			}
		})
		t.Run(tt.name+"_ToCamel", func(t *testing.T) {
			if got := ToCamel(tt.input); got != tt.camel {
				t.Errorf("ToCamel(%q) = %q, want %q", tt.input, got, tt.camel)
			}
		})
		t.Run(tt.name+"_ToPascal", func(t *testing.T) {
			if got := ToPascal(tt.input); got != tt.pascal {
				t.Errorf("ToPascal(%q) = %q, want %q", tt.input, got, tt.pascal)
			}
		})
		t.Run(tt.name+"_ToTitle", func(t *testing.T) {
			if got := ToTitle(tt.input); got != tt.title {
				t.Errorf("ToTitle(%q) = %q, want %q", tt.input, got, tt.title)
			}
		})
		t.Run(tt.name+"_Cycle", func(t *testing.T) {
			funcs := []func(string) string{
				ToSnake,
				ToKebab,
				ToPascal,
				ToTitle,
				ToCamel,
			}
			values := []string{
				tt.snake,
				tt.kebab,
				tt.pascal,
				tt.title,
				tt.camel,
			}

			got := tt.input
			for i, fn := range funcs {
				got = fn(got)
				if got != values[i] {
					t.Errorf("Cycle(%q) = %q, want %q", tt.input, got, values[i])
				}
			}
		})
	}
}

func TestTokenizer(t *testing.T) {
	withInitialism := NewTokenizer(DefaultInitialisms...)
	withoutInitialism := NewTokenizer()

	word := "https_server_api"

	is := assert.New(t)
	is.Equal("httpsServerAPI", withInitialism.Camel(word))
	is.Equal("httpsServerApi", withoutInitialism.Camel(word))
}

func TestTokenize(t *testing.T) {
	tests := map[string][]string{
		"userIDsAPIURL":   {"user", "ids", "api", "url"},
		"getHTTPResponse": {"get", "http", "response"},
		"userID1":         {"user", "id", "1"},
		"ÄpfelBaum":       {"äpfel", "baum"},
		"  --  ":          nil,
	}
	for in, want := range tests {
		assert.Equal(t, want, Tokenize(in, DefaultInitialisms), in)
	}
}
