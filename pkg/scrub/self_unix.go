//go:build unix

package scrub

import (
	"fmt"
	"os"
)

func removeSelf() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("executable path: %w", err)
	}
	if err := os.Remove(exe); err != nil && !os.IsNotExist(err) {
		return exe, err
	}
	return exe, nil
}
