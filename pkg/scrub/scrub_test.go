package scrub

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunRemovesArchiveAndTemps(t *testing.T) {
	dir := t.TempDir()
	arch := filepath.Join(dir, "a.tar.gz")
	tmp := filepath.Join(dir, "tmp.dat")
	if err := os.WriteFile(arch, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmp, []byte("y"), 0644); err != nil {
		t.Fatal(err)
	}
	Run(Options{Archive: arch, Temps: []string{tmp}, Self: false})
	if _, err := os.Stat(arch); !os.IsNotExist(err) {
		t.Fatalf("archive still exists: %v", err)
	}
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Fatalf("temp still exists: %v", err)
	}
}
