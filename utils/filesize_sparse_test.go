//go:build unix

package utils

import (
	"os"
	"os/exec"
	"testing"
)

func TestAllocatedSize_SparseFileSmallerThanLogical(t *testing.T) {
	path := t.TempDir() + "/sparse"
	// Create a 64MB sparse file
	cmd := exec.Command("truncate", "-s", "64m", path)
	if err := cmd.Run(); err != nil {
		t.Skipf("truncate not available: %v", err)
	}
	fi, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	logical := LogicalSize(fi)
	allocated := AllocatedSize(fi)
	if logical < 64*1024*1024 {
		t.Fatalf("logical=%d, want >= 64MB", logical)
	}
	if allocated >= logical {
		t.Fatalf("allocated=%d should be << logical=%d for sparse file", allocated, logical)
	}
}
