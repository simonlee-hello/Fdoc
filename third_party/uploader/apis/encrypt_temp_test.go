package apis

import (
	"os"
	"path/filepath"
	"testing"

	"uploader/crypto"
)

func TestEncryptFileToTempUsesSourceDir(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "payload")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(srcDir, "plain.bin")
	if err := os.WriteFile(src, []byte("encrypt-temp-same-dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, key, err := crypto.NormalizeKey("temp-dir-secret", false)
	if err != nil {
		t.Fatal(err)
	}

	encPath, encSize, err := encryptFileToTemp(src, key)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(encPath)

	if filepath.Dir(encPath) != srcDir {
		t.Fatalf("enc temp dir=%s want source dir=%s (avoid os.TempDir/tmpfs)", filepath.Dir(encPath), srcDir)
	}
	want := crypto.CalcEncryptSize(int64(len("encrypt-temp-same-dir")))
	if encSize != want {
		t.Fatalf("enc size=%d want %d", encSize, want)
	}
	// Must not leave the only copy under the process default temp root when dirs differ.
	if filepath.Dir(encPath) == filepath.Clean(os.TempDir()) && srcDir != filepath.Clean(os.TempDir()) {
		t.Fatal("encrypted temp landed in os.TempDir")
	}
}
