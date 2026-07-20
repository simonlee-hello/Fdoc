//go:build windows

package scrub

import (
	"Fdoc/logx"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

func removeSelf() {
	exe, err := os.Executable()
	if err != nil {
		logx.Warning("scrub self: executable path: %v", err)
		return
	}
	// Try immediate delete (often fails while mapped).
	if err := os.Remove(exe); err == nil || os.IsNotExist(err) {
		logx.Debug("scrubbed self: %s", exe)
		return
	}
	// Rename then schedule delete on reboot.
	dir := filepath.Dir(exe)
	tmp := filepath.Join(dir, filepath.Base(exe)+".deleted")
	if err := os.Rename(exe, tmp); err != nil {
		logx.Warning("scrub self rename: %v", err)
		tmp = exe
	} else {
		exe = tmp
	}
	if err := moveFileExDelete(exe); err != nil {
		logx.Warning("scrub self delayed delete: %v", err)
		return
	}
	logx.Debug("scrub self scheduled: %s", exe)
}

func moveFileExDelete(path string) error {
	mod := syscall.NewLazyDLL("kernel32.dll")
	proc := mod.NewProc("MoveFileExW")
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	const MOVEFILE_DELAY_UNTIL_REBOOT = 0x4
	r, _, e := proc.Call(uintptr(unsafe.Pointer(p)), 0, MOVEFILE_DELAY_UNTIL_REBOOT)
	if r == 0 {
		if e != nil {
			return e
		}
		return syscall.EINVAL
	}
	return nil
}
