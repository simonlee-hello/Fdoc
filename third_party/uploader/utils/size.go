package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var sizeRe = regexp.MustCompile(`(?i)^\s*(\d+(?:\.\d+)?)\s*([kmgt]?b)?`)

// ParseByteSize parses sizes like "100MB", "4GB", "5GB anon". Empty/none/- => 0 (unlimited).
func ParseByteSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "none") || s == "-" {
		return 0, nil
	}
	m := sizeRe.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	n, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, err
	}
	unit := strings.ToUpper(m[2])
	if unit == "" {
		unit = "B"
	}
	var mul float64 = 1
	switch unit {
	case "B":
		mul = 1
	case "KB":
		mul = 1024
	case "MB":
		mul = 1024 * 1024
	case "GB":
		mul = 1024 * 1024 * 1024
	case "TB":
		mul = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("invalid size unit in %q", s)
	}
	return int64(n * mul), nil
}

// FormatByteSize renders a human size; 0 means unlimited.
func FormatByteSize(n int64) string {
	if n <= 0 {
		return "unlimited"
	}
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
		tb = gb * 1024
	)
	switch {
	case n >= tb:
		return fmt.Sprintf("%.1fTB", float64(n)/float64(tb))
	case n >= gb:
		return fmt.Sprintf("%.1fGB", float64(n)/float64(gb))
	case n >= mb:
		return fmt.Sprintf("%.1fMB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.1fKB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%dB", n)
	}
}

// CheckUploadSize returns an error when size exceeds backend limit (limit<=0 = unlimited).
func CheckUploadSize(name string, size, limit int64, backend string) error {
	if limit <= 0 || size <= limit {
		return nil
	}
	if name == "" {
		name = "file"
	}
	return fmt.Errorf("%s is %s, backend %s limit is %s — abort before upload",
		name, FormatByteSize(size), backend, FormatByteSize(limit))
}
