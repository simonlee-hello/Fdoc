package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAllocatedSize_FallsBackToLogical(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "empty")
	if err != nil {
		t.Fatal(err)
	}
	name := f.Name()
	_ = f.Close()

	fi, err := os.Lstat(name)
	if err != nil {
		t.Fatal(err)
	}
	if got := AllocatedSize(fi); got < 0 {
		t.Fatalf("AllocatedSize=%d, want >= 0", got)
	}
	if LogicalSize(fi) != 0 {
		t.Fatalf("LogicalSize empty file = %d", LogicalSize(fi))
	}
}

func TestFileIdentity_SameFileStable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	fi1, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	fi2, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	id1, ok1 := FileIdentity(path, fi1)
	id2, ok2 := FileIdentity(path, fi2)
	if !ok1 || !ok2 {
		t.Fatal("expected file identity to be available")
	}
	if id1 != id2 {
		t.Fatalf("identity changed: %v vs %v", id1, id2)
	}
}

func TestSamePath(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "x")
	b := filepath.Join(dir, ".", "x")
	if !SamePath(a, b) {
		t.Fatalf("SamePath(%q,%q) = false", a, b)
	}
	if runtime.GOOS == "windows" {
		if !SamePath(`C:\Windows`, `c:\Windows`) {
			t.Fatal(`SamePath should treat C:\Windows and c:\Windows as equal`)
		}
	}
}
