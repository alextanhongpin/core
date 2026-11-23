package email_test

import (
	"testing"

	"github.com/alextanhongpin/core/types/email"
)

func TestSplitAtLastAt(t *testing.T) {
	addr := `"john@home"@example.com`
	if got := email.Domain(addr); got != "example.com" {
		t.Fatalf("Domain = %q", got)
	}
	if got := email.LocalPart(addr); got != `"john@home"` {
		t.Fatalf("LocalPart = %q", got)
	}
	for _, s := range []string{"", "no-at", "@example.com", "john@"} {
		if email.Domain(s) != "" || email.LocalPart(s) != "" {
			t.Fatalf("%q: want empty parts", s)
		}
	}
}

func TestIsBusinessEmailCaseInsensitive(t *testing.T) {
	if email.IsBusinessEmail("John@GMAIL.com") {
		t.Fatal("gmail.com should be a consumer domain")
	}
	if !email.IsBusinessEmail("john@acme.io") {
		t.Fatal("acme.io should be a business domain")
	}
}
