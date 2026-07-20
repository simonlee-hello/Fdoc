package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseSize parses a size string like "1GB" into bytes.
// Empty/"0" are valid and return 0 (callers treat 0 as unlimited where applicable).
// Unknown units and non-numeric values return an error.
func ParseSize(maxsize string) (int64, error) {
	maxsize = strings.ToUpper(strings.ReplaceAll(maxsize, " ", ""))
	if maxsize == "" || maxsize == "0" {
		return 0, nil
	}

	numStr := maxsize
	unit := ""
	for i, char := range maxsize {
		if (char < '0' || char > '9') && char != '.' {
			numStr = maxsize[:i]
			unit = maxsize[i:]
			break
		}
	}
	if numStr == "" {
		return 0, fmt.Errorf("invalid size %q: missing number", maxsize)
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q: %w", maxsize, err)
	}
	if num < 0 {
		return 0, fmt.Errorf("invalid size %q: negative value", maxsize)
	}

	switch unit {
	case "K", "KB":
		return int64(num * 1024), nil
	case "M", "MB":
		return int64(num * 1024 * 1024), nil
	case "G", "GB":
		return int64(num * 1024 * 1024 * 1024), nil
	case "", "B", "BYTE", "BYTES":
		return int64(num), nil
	default:
		return 0, fmt.Errorf("unknown size unit %q in %q", unit, maxsize)
	}
}

// SizeToBytes is like ParseSize but returns 0 on error (prefer ParseSize for new code).
func SizeToBytes(maxsize string) int64 {
	n, err := ParseSize(maxsize)
	if err != nil {
		return 0
	}
	return n
}

func BytesToSize(bytes int64) string {
	unit := "Bytes"
	value := float64(bytes)

	if bytes >= 1024*1024*1024 {
		unit = "GB"
		value = float64(bytes) / (1024 * 1024 * 1024)
	} else if bytes >= 1024*1024 {
		unit = "MB"
		value = float64(bytes) / (1024 * 1024)
	} else if bytes >= 1024 {
		unit = "KB"
		value = float64(bytes) / 1024
	}
	return fmt.Sprintf("%.2f %s", value, unit)
}

func ConvertStringToList(input string) []string {
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmedPart := strings.TrimSpace(part)
		if trimmedPart != "" {
			result = append(result, trimmedPart)
		}
	}
	return result
}

// StringToMap converts "docx,pdf,.xlsx" into {".pdf": {}, ".docx": {}, ".xlsx": {}}.
func StringToMap(supportedExtensions string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, ext := range strings.Split(supportedExtensions, ",") {
		ext = strings.TrimSpace(ext)
		if ext == "" {
			continue
		}
		ext = strings.ToLower(ext)
		ext = strings.TrimPrefix(ext, ".")
		if ext != "" {
			m["."+ext] = struct{}{}
		}
	}
	return m
}

func TransformSlash(input string) string {
	return strings.ReplaceAll(input, `\`, `/`)
}
