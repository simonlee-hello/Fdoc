package scrub

import (
	"os"
	"path/filepath"
	"strings"
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

func TestDelayedDeleteBatch_ContainsTargetAndSelf(t *testing.T) {
	body := delayedDeleteBatch(`C:\Users\a\Fdoc.exe`, `C:\Temp\scrub.cmd`)
	if !strings.Contains(body, `del /f /q "C:\Users\a\Fdoc.exe"`) {
		t.Fatalf("missing target delete:\n%s", body)
	}
	if !strings.Contains(body, `del /f /q "C:\Temp\scrub.cmd"`) {
		t.Fatalf("missing self-delete:\n%s", body)
	}
	if !strings.Contains(body, "ping -n 3") {
		t.Fatalf("missing delay:\n%s", body)
	}
}
