package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

var (
	legacyFixedIV = bytes.Repeat([]byte{'7'}, 16)
	modernMagic   = []byte("UP01")
)

// CalcEncryptSize returns ciphertext size for modern AES-CBC format:
// magic(4) + IV(16) + PKCS7-padded body.
func CalcEncryptSize(size int64) int64 {
	pad := aes.BlockSize - int(size%aes.BlockSize)
	return int64(len(modernMagic)+aes.BlockSize) + size + int64(pad)
}

// StreamEncrypt writes AES-CBC ciphertext: [UP01][random IV 16][padded CBC body].
func StreamEncrypt(reader io.Reader, writer io.Writer, key string, _ int64) error {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return err
	}

	if _, err := writer.Write(modernMagic); err != nil {
		return err
	}

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return err
	}
	if _, err := writer.Write(iv); err != nil {
		return err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	pending := make([]byte, 0, 64*1024)
	tmp := make([]byte, 32*1024)

	flushFullBlocks := func(keepLastBlock bool) error {
		n := len(pending)
		if n < aes.BlockSize {
			return nil
		}
		usable := n - n%aes.BlockSize
		if keepLastBlock {
			if usable == n {
				usable -= aes.BlockSize
			}
		}
		if usable <= 0 {
			return nil
		}
		out := make([]byte, usable)
		mode.CryptBlocks(out, pending[:usable])
		if _, err := writer.Write(out); err != nil {
			return err
		}
		pending = append(pending[:0], pending[usable:]...)
		return nil
	}

	for {
		nr, readErr := reader.Read(tmp)
		if nr > 0 {
			pending = append(pending, tmp[:nr]...)
			if err := flushFullBlocks(true); err != nil {
				return err
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}

	padded := Padding(pending, aes.BlockSize)
	out := make([]byte, len(padded))
	mode.CryptBlocks(out, padded)
	_, err = writer.Write(out)
	return err
}

// StreamDecrypt reads modern (UP01-prefixed) or legacy (fixed-IV) ciphertext.
func StreamDecrypt(reader io.Reader, writer io.Writer, key string, _ int64) error {
	all, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	if len(all) < aes.BlockSize {
		return fmt.Errorf("ciphertext too short")
	}

	keyBytes := []byte(key)

	if bytes.HasPrefix(all, modernMagic) {
		body := all[len(modernMagic):]
		if len(body) < aes.BlockSize*2 || len(body)%aes.BlockSize != 0 {
			return fmt.Errorf("invalid modern ciphertext length")
		}
		return decryptCBCToWriter(body[aes.BlockSize:], keyBytes, body[:aes.BlockSize], writer)
	}

	// Legacy fixed-IV format (no magic / IV prefix)
	if len(all)%aes.BlockSize != 0 {
		return fmt.Errorf("invalid ciphertext length")
	}
	return decryptCBCToWriter(all, keyBytes, legacyFixedIV, writer)
}

func decryptCBCToWriter(cipherText, key, iv []byte, writer io.Writer) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	if len(cipherText) == 0 || len(cipherText)%aes.BlockSize != 0 {
		return fmt.Errorf("invalid block alignment")
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	plain := make([]byte, len(cipherText))
	mode.CryptBlocks(plain, cipherText)
	plain, err = safeUnpadding(plain)
	if err != nil {
		return err
	}
	_, err = writer.Write(plain)
	return err
}

func safeUnpadding(src []byte) ([]byte, error) {
	n := len(src)
	if n == 0 {
		return nil, fmt.Errorf("empty plaintext")
	}
	pad := int(src[n-1])
	if pad == 0 || pad > aes.BlockSize || pad > n {
		return nil, fmt.Errorf("invalid padding")
	}
	for i := 0; i < pad; i++ {
		if src[n-1-i] != byte(pad) {
			return nil, fmt.Errorf("invalid padding")
		}
	}
	return src[:n-pad], nil
}
