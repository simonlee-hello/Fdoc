package crypto

import (
	"bytes"
)

func Padding(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	paddingText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, paddingText...)
}

func unPadding(src []byte) []byte {
	out, err := safeUnpadding(src)
	if err != nil {
		return src
	}
	return out
}
