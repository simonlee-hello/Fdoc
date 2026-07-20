package crypto

import (
	"strings"
	"testing"
)

func TestNormalizeKeyRejectsTooLong(t *testing.T) {
	_, _, err := NormalizeKey(strings.Repeat("x", 33), false)
	if err == nil || !strings.Contains(err.Error(), "longer than 32") {
		t.Fatalf("want longer-than-32 error, got %v", err)
	}
}

func TestNormalizeKeyRequiresNonEmpty(t *testing.T) {
	_, _, err := NormalizeKey("", false)
	if err == nil || !strings.Contains(err.Error(), "key required") {
		t.Fatalf("want key required, got %v", err)
	}
}
