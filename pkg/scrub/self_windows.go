//go:build windows

package scrub

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

func removeSelf() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("executable path: %w", err)
	}
	// Try immediate delete (often fails while mapped).
	if err := os.Remove(exe); err == nil || os.IsNotExist(err) {
		return exe, nil
	}
	// Rename then schedule delete on reboot (may require elevated privileges).
	dir := filepath.Dir(exe)
	tmp := filepath.Join(dir, filepath.Base(exe)+".deleted")
	if err := os.Rename(exe, tmp); err != nil {
		tmp = exe
	} else {
		exe = tmp
	}
	if err := moveFileExDelete(exe); err != nil {
		return exe, fmt.Errorf("delayed delete (often needs admin): %w", err)
	}
	return exe, nil
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
