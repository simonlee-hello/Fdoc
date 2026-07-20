//go:build unix

package pkg

import (
	"os/signal"
	"syscall"
)

// IgnoreSIGHUP prevents session hangup from killing long upload/callback work.
func IgnoreSIGHUP() {
	signal.Ignore(syscall.SIGHUP)
}
