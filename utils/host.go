package utils

import (
	"net"
	"strings"
)

// IsLoopbackHost reports whether host is a loopback name or IP (incl. 127.0.0.0/8).
func IsLoopbackHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
