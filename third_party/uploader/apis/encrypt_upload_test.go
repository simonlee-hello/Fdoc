package apis_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"uploader/apis"
	"uploader/crypto"
)

type captureBackend struct {
	apis.Backend
	buf bytes.Buffer
}

func (c *captureBackend) DoUpload(_ string, size int64, file io.Reader) error {
	n, err := io.Copy(&c.buf, file)
	if err != nil {
		return err
	}
	if n != size {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func (c *captureBackend) PostUpload(string, int64) (string, error) {
	return "https://example.com/x", nil
}

func (c *captureBackend) LinkMatcher(string) bool { return true }

func TestUploadEncryptsBeforeSend(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "a.tar.gz")
	// minimal gzip-like bytes (not a real archive; just non-UP01 plaintext)
	raw := append([]byte{0x1f, 0x8b, 0x08, 0x00}, bytes.Repeat([]byte{0xab}, 100)...)
	if err := os.WriteFile(plain, raw, 0644); err != nil {
		t.Fatal(err)
	}

	_, key, err := crypto.NormalizeKey("test-secret", false)
	if err != nil {
		t.Fatal(err)
	}

	b := &captureBackend{}
	link, err := apis.UploadFileOpts(plain, b, apis.FileUploadOpts{
		NoBar:     true,
		Crypto:    true,
		CryptoKey: key,
	})
	if err != nil {
		t.Fatal(err)
	}
	if link == "" {
		t.Fatal("empty link")
	}
	got := b.buf.Bytes()
	wantSize := crypto.CalcEncryptSize(int64(len(raw)))
	if int64(len(got)) != wantSize {
		t.Fatalf("uploaded size %d want %d", len(got), wantSize)
	}
	if !bytes.HasPrefix(got, []byte("UP01")) {
		n := 8
		if len(got) < n {
			n = len(got)
		}
		t.Fatalf("uploaded body missing UP01 header: %x", got[:n])
	}
	if bytes.HasPrefix(got, []byte{0x1f, 0x8b}) {
		t.Fatal("uploaded plaintext gzip instead of ciphertext")
	}

	var out bytes.Buffer
	if err := crypto.StreamDecrypt(bytes.NewReader(got), &out, key, 0); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), raw) {
		t.Fatal("round-trip mismatch")
	}
}
