//go:build windows

package scrub

import (
	"fmt"
	"os"
	"os/exec"
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
	moveErr := moveFileExDelete(exe)
	if moveErr == nil {
		return exe, nil
	}
	// Non-admin fallback: detached cmd that deletes after a short delay.
	if err := launchDelayedDelete(exe); err != nil {
		return exe, fmt.Errorf("delayed delete (often needs admin): moveFileEx: %v; cmd: %w", moveErr, err)
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

func launchDelayedDelete(path string) error {
	script := filepath.Join(os.TempDir(), fmt.Sprintf("fdoc_scrub_%d.cmd", os.Getpid()))
	body := delayedDeleteBatch(path, script)
	if err := os.WriteFile(script, []byte(body), 0644); err != nil {
		return err
	}
	// start "" <script> — empty title is required by cmd start syntax.
	cmd := exec.Command("cmd.exe", "/C", "start", "/MIN", "", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		_ = os.Remove(script)
		return err
	}
	_ = cmd.Process.Release()
	return nil
}
