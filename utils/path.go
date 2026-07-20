package utils

import (
	"path/filepath"
	"runtime"
	"strings"
)

// SamePath reports whether two filesystem paths refer to the same location.
// On Windows comparison is case-insensitive.
func SamePath(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if abs, err := filepath.Abs(a); err == nil {
		a = abs
	}
	if abs, err := filepath.Abs(b); err == nil {
		b = abs
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
