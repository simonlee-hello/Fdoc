package utils

import (
	"errors"
	"os"
	"strings"
)

// IsAccessDenied reports whether err is a permission / TCC / ACL denial.
func IsAccessDenied(err error) bool {
	if err == nil {
		return false
	}
	if os.IsPermission(err) {
		return true
	}
	// Some platforms wrap or reword access errors.
	msg := strings.ToLower(err.Error())
	for _, needle := range []string{
		"permission denied",
		"operation not permitted",
		"access is denied",
		"denied access",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return IsAccessDenied(pathErr.Err)
	}
	return false
}
