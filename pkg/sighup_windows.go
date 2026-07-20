//go:build windows

package pkg

// IgnoreSIGHUP is a no-op on Windows.
func IgnoreSIGHUP() {}
