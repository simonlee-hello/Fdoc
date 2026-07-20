//go:build unix

package scrub

import (
	"Fdoc/logx"
	"os"
)

func removeSelf() {
	exe, err := os.Executable()
	if err != nil {
		logx.Warning("scrub self: executable path: %v", err)
		return
	}
	if err := os.Remove(exe); err != nil && !os.IsNotExist(err) {
		logx.Warning("scrub self: %v", err)
		return
	}
	logx.Debug("scrubbed self: %s", exe)
}
