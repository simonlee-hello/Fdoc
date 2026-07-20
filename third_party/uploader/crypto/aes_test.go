package crypto

import (
	"bytes"
	"io"
	"testing"
	"uploader/utils"
)

func TestECB(t *testing.T) {
	raw := utils.GenRandBytes(16)
	key := utils.GenRandBytes(16)
	src := encryptAESECB(raw, key)
	dec := decryptAESECB(src, key)
	if !bytes.Equal(dec, raw) {
		t.Fatal("aes-ecb: failed")
	}
}

func TestCBC(t *testing.T) {
	raw := utils.GenRandBytes(16)
	iv := utils.GenRandBytes(16)
	key := utils.GenRandBytes(16)
	src := encryptAESCBC(raw, key, iv)
	dec := decryptAESCBC(src, key, iv)
	if !bytes.Equal(dec, raw) {
		t.Fatal("aes-cbc: failed")
	}
}

func TestStreamRoundTrip(t *testing.T) {
	raw := utils.GenRandBytes(1000)
	key := string(utils.GenRandBytes(32))
	var buf bytes.Buffer
	if err := StreamEncrypt(bytes.NewReader(raw), &buf, key, 0); err != nil {
		t.Fatal(err)
	}
	if int64(buf.Len()) != CalcEncryptSize(int64(len(raw))) {
		t.Fatalf("encrypt size mismatch: got %d want %d", buf.Len(), CalcEncryptSize(int64(len(raw))))
	}
	var out bytes.Buffer
	if err := StreamDecrypt(bytes.NewReader(buf.Bytes()), &out, key, 0); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), raw) {
		t.Fatal("stream round-trip failed")
	}
}

func TestCalcEncryptSize(t *testing.T) {
	key := string(bytes.Repeat([]byte("k"), 32))
	for _, n := range []int{0, 1, 15, 16, 17, 436, 1000} {
		raw := utils.GenRandBytes(n)
		var buf bytes.Buffer
		if err := StreamEncrypt(bytes.NewReader(raw), &buf, key, 0); err != nil {
			t.Fatal(err)
		}
		want := CalcEncryptSize(int64(n))
		if int64(buf.Len()) != want {
			t.Fatalf("n=%d: got %d want %d", n, buf.Len(), want)
		}
	}
}

func TestStreamLegacyDecrypt(t *testing.T) {
	raw := []byte("hello legacy encrypt payload!!") // 30 bytes
	key := string(bytes.Repeat([]byte("k"), 32))
	blockOut := encryptAESCBC(raw, []byte(key), legacyFixedIV)
	var out bytes.Buffer
	if err := StreamDecrypt(bytes.NewReader(blockOut), &out, key, 0); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), raw) {
		t.Fatalf("legacy decrypt failed: got %q", out.Bytes())
	}
}

func TestStreamDecryptEmptyPlaintext(t *testing.T) {
	key := string(bytes.Repeat([]byte("k"), 32))
	var enc bytes.Buffer
	if err := StreamEncrypt(bytes.NewReader(nil), &enc, key, 0); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := StreamDecrypt(bytes.NewReader(enc.Bytes()), &out, key, 0); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("want empty plaintext, got %d bytes", out.Len())
	}
}

func TestStreamDecryptChunkedReader(t *testing.T) {
	// 1MiB+odd so encrypt pads across many CBC blocks; feed ciphertext 1–7 bytes at a time.
	raw := utils.GenRandBytes(1<<20 + 3)
	key := string(utils.GenRandBytes(32))
	var enc bytes.Buffer
	if err := StreamEncrypt(bytes.NewReader(raw), &enc, key, 0); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := StreamDecrypt(&tinyReader{r: bytes.NewReader(enc.Bytes()), n: 7}, &out, key, 0); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), raw) {
		t.Fatal("chunked stream decrypt mismatch")
	}
}

func TestStreamDecryptRejectsPlainGzip(t *testing.T) {
	key := string(bytes.Repeat([]byte("k"), 32))
	gz := []byte{0x1f, 0x8b, 0x08, 0x00}
	err := StreamDecrypt(bytes.NewReader(gz), io.Discard, key, 0)
	if err == nil {
		t.Fatal("expected error for plaintext gzip")
	}
}

// tinyReader forces small Read sizes so decrypt cannot rely on a single big ReadAll.
type tinyReader struct {
	r io.Reader
	n int
}

func (t *tinyReader) Read(p []byte) (int, error) {
	if t.n > 0 && len(p) > t.n {
		p = p[:t.n]
	}
	return t.r.Read(p)
}
