package scrub

import (
	"fmt"
	"path/filepath"
	"strings"
)

// delayedDeleteBatch returns a Windows .cmd script that waits briefly, deletes
// target, then removes itself. Used as a non-admin fallback when MoveFileEx
// DELAY_UNTIL_REBOOT is unavailable.
func delayedDeleteBatch(target, scriptPath string) string {
	// Normalize for cmd.exe; quote paths that may contain spaces.
	t := filepath.Clean(target)
	s := filepath.Clean(scriptPath)
	var b strings.Builder
	b.WriteString("@echo off\r\n")
	b.WriteString("ping -n 3 127.0.0.1 >nul\r\n")
	b.WriteString(fmt.Sprintf("del /f /q \"%s\"\r\n", t))
	b.WriteString(fmt.Sprintf("if exist \"%s\" del /f /q \"%s\"\r\n", s, s))
	return b.String()
}
