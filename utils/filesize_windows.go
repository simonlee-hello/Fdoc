//go:build windows

package utils

import (
	"os"
	"syscall"
)

// LogicalSize returns the file's logical size.
func LogicalSize(fi os.FileInfo) int64 {
	return fi.Size()
}

// AllocatedSize returns on-disk usage; on Windows this matches logical size.
func AllocatedSize(fi os.FileInfo) int64 {
	return fi.Size()
}

// FileIdentity returns volume serial + file index for hard-link dedupe / output-self.
func FileIdentity(path string, fi os.FileInfo) (id [2]uint64, ok bool) {
	f, err := os.Open(path)
	if err != nil {
		return id, false
	}
	defer f.Close()

	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(syscall.Handle(f.Fd()), &info); err != nil {
		return id, false
	}
	index := (uint64(info.FileIndexHigh) << 32) | uint64(info.FileIndexLow)
	return [2]uint64{uint64(info.VolumeSerialNumber), index}, true
}
