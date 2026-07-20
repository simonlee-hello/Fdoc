package crypto

import (
	"bytes"
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
