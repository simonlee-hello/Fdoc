package utils

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestZipDir(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "proj")
	if err := os.MkdirAll(filepath.Join(sub, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "nested", "b.txt"), []byte("world"), 0644); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "proj.zip")
	if err := ZipDir(sub, out); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("empty zip")
	}

	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	if !names["a.txt"] || !names["nested/b.txt"] {
		t.Fatalf("unexpected entries: %v", names)
	}
}

func TestZipDirCompresses(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "data")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	// Highly compressible payload (~200KB of zeros/repeats).
	payload := make([]byte, 200*1024)
	for i := range payload {
		payload[i] = 'A'
	}
	rawPath := filepath.Join(sub, "big.txt")
	if err := os.WriteFile(rawPath, payload, 0644); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "data.zip")
	if err := ZipDir(sub, out); err != nil {
		t.Fatal(err)
	}
	zi, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if zi.Size() >= int64(len(payload)) {
		t.Fatalf("zip not compressed: zip=%d raw=%d", zi.Size(), len(payload))
	}

	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	found := false
	for _, f := range zr.File {
		if f.Name == "big.txt" {
			found = true
			if f.Method != zip.Deflate {
				t.Fatalf("method=%d want Deflate", f.Method)
			}
			if f.CompressedSize64 >= f.UncompressedSize64 {
				t.Fatalf("entry not compressed: %d >= %d", f.CompressedSize64, f.UncompressedSize64)
			}
		}
	}
	if !found {
		t.Fatal("big.txt missing")
	}
}

