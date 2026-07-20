package option

import (
	"strings"
	"testing"
)

func TestSanitizeTaskID(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"Op42", "op42"},
		{"My_Task!!", "mytask"},
		{"a-b-c", "a-b-c"},
		{strings.Repeat("x", 50), strings.Repeat("x", 40)},
	}
	for _, c := range cases {
		if got := sanitizeTaskID(c.in); got != c.want {
			t.Fatalf("sanitizeTaskID(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestValidateWebhookURL(t *testing.T) {
	ok := []string{
		"https://example.com/hook",
		"http://127.0.0.1:8080/h",
		"http://localhost/h",
		"http://127.0.0.2/h",
	}
	for _, u := range ok {
		if err := validateWebhookURL(u); err != nil {
			t.Fatalf("%s: %v", u, err)
		}
	}
	bad := []string{
		"http://example.com/h",
		"ftp://x",
		"://bad",
	}
	for _, u := range bad {
		if err := validateWebhookURL(u); err == nil {
			t.Fatalf("%s: expected error", u)
		}
	}
}
