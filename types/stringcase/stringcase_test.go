package stringcase

import "testing"

func TestStringCaseConversions(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		kebab   string
		snake   string
		camel   string
		pascal  string
		title   string
		fromKeb string
		fromSnk string
	}{
		{"simple words", "hello world", "hello-world", "hello_world", "helloWorld", "HelloWorld", "Hello World", "hello world", "hello world"},
		{"single word", "hello", "hello", "hello", "hello", "Hello", "Hello", "hello", "hello"},
		{"PascalCase", "HelloWorld", "hello-world", "hello_world", "helloWorld", "HelloWorld", "Hello World", "hello world", "hello world"},
		{"camelCase", "helloWorld", "hello-world", "hello_world", "helloWorld", "HelloWorld", "Hello World", "hello world", "hello world"},
		{"snake_case", "hello_world", "hello-world", "hello_world", "helloWorld", "HelloWorld", "Hello World", "hello world", "hello world"},
		{"kebab-case", "hello-world", "hello-world", "hello_world", "helloWorld", "HelloWorld", "Hello World", "hello world", "hello world"},
		{"Title Case", "Hello World", "hello-world", "hello_world", "helloWorld", "HelloWorld", "Hello World", "hello world", "hello world"},
		{"with initialism", "userID", "user-id", "user_id", "userID", "UserID", "User ID", "user id", "user id"},
		{"with initialism 2", "apiResponseID", "api-response-id", "api_response_id", "apiResponseID", "APIResponseID", "API Response ID", "api response id", "api response id"},
		{"with initialism 3", "HTTPServerID", "http-server-id", "http_server_id", "httpServerID", "HTTPServerID", "HTTP Server ID", "http server id", "http server id"},
		{"with numbers", "user1ID", "user1-id", "user1_id", "user1ID", "User1ID", "User1 ID", "user1 id", "user1 id"},
		{"with underscores and dashes", "user_id-test", "user-id-test", "user_id_test", "userIDTest", "UserIDTest", "User ID Test", "user id test", "user id test"},
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
		t.Run(tt.name+"_FromKebab", func(t *testing.T) {
			if got := FromKebab(tt.kebab); got != tt.fromKeb {
				t.Errorf("FromKebab(%q) = %q, want %q", tt.kebab, got, tt.fromKeb)
			}
		})
		t.Run(tt.name+"_FromSnake", func(t *testing.T) {
			if got := FromSnake(tt.snake); got != tt.fromSnk {
				t.Errorf("FromSnake(%q) = %q, want %q", tt.snake, got, tt.fromSnk)
			}
		})
	}
}
