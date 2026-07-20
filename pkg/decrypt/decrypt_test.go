package decrypt

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"uploader/crypto"
)

func TestFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	plainPath := filepath.Join(dir, "plain.tgz")
	plain := append([]byte{0x1f, 0x8b, 0x08, 0x00}, bytes.Repeat([]byte("FdocPack"), 20)...)
	if err := os.WriteFile(plainPath, plain, 0644); err != nil {
		t.Fatal(err)
	}

	_, key, err := crypto.NormalizeKey("test-secret", false)
	if err != nil {
		t.Fatal(err)
	}
	encPath := filepath.Join(dir, "cipher.bin")
	enc, err := os.Create(encPath)
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.Open(plainPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := crypto.StreamEncrypt(src, enc, key, 0); err != nil {
		t.Fatal(err)
	}
	_ = src.Close()
	_ = enc.Close()

	outPath := filepath.Join(dir, "out.tgz")
	res, err := File(encPath, Options{Key: "test-secret", Output: outPath})
	if err != nil {
		t.Fatal(err)
	}
	if res.Output != outPath {
		t.Fatalf("output path: %q", res.Output)
	}
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatal("decrypted bytes mismatch")
	}
}

func TestFileRejectsPlainGzip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "plain.gz")
	if err := os.WriteFile(p, []byte{0x1f, 0x8b, 0x08, 0x00, 0x01, 0x02, 0x03, 0x04}, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := File(p, Options{Key: "test-secret", Output: filepath.Join(dir, "o.tgz"), Force: true})
	if err == nil {
		t.Fatal("expected error for plaintext gzip")
	}
}

func TestFileRequiresKey(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.bin")
	if err := os.WriteFile(p, []byte("UP01"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := File(p, Options{Output: filepath.Join(dir, "o.tgz")})
	if err == nil {
		t.Fatal("expected key required")
	}
}

func TestDefaultOutputPath(t *testing.T) {
	got := defaultOutputPath("/tmp/a/dhcoy7.gz")
	if got != "/tmp/a/dhcoy7.tgz" {
		t.Fatalf("gz: got %q", got)
	}
	got = defaultOutputPath("cipher.bin")
	if got != "cipher.tgz" {
		t.Fatalf("bin: got %q", got)
	}
	got = defaultOutputPath("/tmp/a/dhcoy7.tgz")
	if got != "/tmp/a/dhcoy7.dec.tgz" {
		t.Fatalf("tgz input: got %q want .../dhcoy7.dec.tgz", got)
	}
	got = defaultOutputPath("/tmp/a/pack.tar.gz")
	if got != "/tmp/a/pack.dec.tgz" {
		t.Fatalf("tar.gz: got %q", got)
	}
}
