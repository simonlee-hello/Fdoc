//go:build unix

package utils

import (
	"os"
	"syscall"
)

// LogicalSize returns the file's logical size (st_size).
func LogicalSize(fi os.FileInfo) int64 {
	return fi.Size()
}

// AllocatedSize returns approximate on-disk usage (st_blocks * 512).
// Sparse files and APFS clones often have allocated << logical.
func AllocatedSize(fi os.FileInfo) int64 {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return fi.Size()
	}
	return st.Blocks * 512
}

// FileIdentity returns a stable (dev, ino) pair for hard-link deduplication.
func FileIdentity(path string, fi os.FileInfo) (id [2]uint64, ok bool) {
	_ = path
	st, okSys := fi.Sys().(*syscall.Stat_t)
	if !okSys {
		return id, false
	}
	return [2]uint64{uint64(st.Dev), uint64(st.Ino)}, true
}
