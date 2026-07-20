package crypto

import (
	"bytes"
	"testing"
	"uploader/utils"
)

func TestDES(t *testing.T) {
	raw := utils.GenRandBytes(16)
	key := utils.GenRandBytes(8)
	src, err := encryptDES(raw, key)
	if err != nil {
		t.Fatal("des: failed", err)
	}
	dec, err := decryptDES(src, key)
	if err != nil {
		t.Fatal("des: failed", err)
	}
	if !bytes.Equal(dec, raw) {
		t.Fatal("des: failed")
	}
}

func TestDESCBC(t *testing.T) {
	raw := utils.GenRandBytes(16)
	key := utils.GenRandBytes(8)
	iv := utils.GenRandBytes(8)
	src, err := EncryptDESCBC(raw, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := DecryptDESCBC(src, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(dec, raw) {
		t.Fatal("des-cbc round-trip failed")
	}
}
