package option

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestApplyDefaultSkipDirs_WindowsRelativeToHome(t *testing.T) {
	if runtime.GOOS != "windows" {
		// Still assert the list content via applyDefaultSkipDirs with forced path:
		// call the helper that builds the windows list when GOOS is windows only.
		// On non-Windows, verify DefaultWindowsSkipDirs() returns relative entries.
	}
	dirs := DefaultWindowsSkipDirs()
	if dirs == "" {
		t.Fatal("DefaultWindowsSkipDirs empty")
	}
	parts := strings.Split(dirs, ",")
	foundTemp := false
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if filepath.IsAbs(p) {
			t.Fatalf("windows default skip should be relative to home, got absolute %q", p)
		}
		if strings.EqualFold(p, `AppData\Local\Temp`) || strings.EqualFold(p, `AppData/Local/Temp`) {
			foundTemp = true
		}
	}
	if !foundTemp {
		t.Fatalf("expected AppData\\Local\\Temp in defaults, got %q", dirs)
	}
}

func TestApplyDefaultSkipDirs_RespectsExplicit(t *testing.T) {
	info := &FlagInfo{SkipDirs: "custom"}
	info.applyDefaultSkipDirs()
	if info.SkipDirs != "custom" {
		t.Fatalf("got %q", info.SkipDirs)
	}
}

func TestResolveHomeDir_PrefersUserHomeDir(t *testing.T) {
	want, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("UserHomeDir unavailable: %v", err)
	}
	got, err := resolveHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("resolveHomeDir=%q want %q", got, want)
	}
}

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
