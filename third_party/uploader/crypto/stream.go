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
	outBuf := make([]byte, 0, 64*1024)

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
		if cap(outBuf) < usable {
			outBuf = make([]byte, usable)
		} else {
			outBuf = outBuf[:usable]
		}
		mode.CryptBlocks(outBuf, pending[:usable])
		if _, err := writer.Write(outBuf); err != nil {
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
// Decryption is streamed in CBC-block chunks so multi-GB archives do not need
// to fit in RAM (only a small pending buffer is held).
func StreamDecrypt(reader io.Reader, writer io.Writer, key string, _ int64) error {
	keyBytes := []byte(key)

	hdr := make([]byte, len(modernMagic))
	if _, err := io.ReadFull(reader, hdr); err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return fmt.Errorf("ciphertext too short")
		}
		return err
	}

	if bytes.Equal(hdr, modernMagic) {
		iv := make([]byte, aes.BlockSize)
		if _, err := io.ReadFull(reader, iv); err != nil {
			return fmt.Errorf("invalid modern ciphertext: missing IV: %w", err)
		}
		return streamDecryptCBC(reader, writer, keyBytes, iv)
	}

	// Legacy fixed-IV format (no magic / IV prefix)
	if hdr[0] == 0x1f && hdr[1] == 0x8b {
		return fmt.Errorf("file looks like plaintext gzip (not encrypted); re-upload with -encrypt -key, expect UP01 header")
	}
	return streamDecryptCBC(io.MultiReader(bytes.NewReader(hdr), reader), writer, keyBytes, legacyFixedIV)
}

// streamDecryptCBC decrypts AES-CBC from reader to writer, holding only a
// small pending buffer. PKCS7 padding is stripped from the final block(s).
func streamDecryptCBC(reader io.Reader, writer io.Writer, key, iv []byte) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	mode := cipher.NewCBCDecrypter(block, iv)

	pending := make([]byte, 0, 64*1024)
	tmp := make([]byte, 32*1024)
	outBuf := make([]byte, 0, 64*1024)

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
		if cap(outBuf) < usable {
			outBuf = make([]byte, usable)
		} else {
			outBuf = outBuf[:usable]
		}
		mode.CryptBlocks(outBuf, pending[:usable])
		if _, err := writer.Write(outBuf); err != nil {
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

	if len(pending) == 0 || len(pending)%aes.BlockSize != 0 {
		return fmt.Errorf("invalid block alignment")
	}
	out := make([]byte, len(pending))
	mode.CryptBlocks(out, pending)
	plain, err := safeUnpadding(out)
	if err != nil {
		return err
	}
	_, err = writer.Write(plain)
	return err
}

func decryptCBCToWriter(cipherText, key, iv []byte, writer io.Writer) error {
	return streamDecryptCBC(bytes.NewReader(cipherText), writer, key, iv)
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
