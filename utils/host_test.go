package utils

import "testing"

func TestIsLoopbackHost(t *testing.T) {
	yes := []string{"localhost", "127.0.0.1", "::1", "127.0.0.2", "LOCALHOST"}
	for _, h := range yes {
		if !IsLoopbackHost(h) {
			t.Fatalf("%q should be loopback", h)
		}
	}
	no := []string{"example.com", "8.8.8.8", ""}
	for _, h := range no {
		if IsLoopbackHost(h) {
			t.Fatalf("%q should not be loopback", h)
		}
	}
}
